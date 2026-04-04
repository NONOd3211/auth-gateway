package providers

import (
	"auth-gateway/database"
	"auth-gateway/models"
	"fmt"
	"hash/fnv"
	"net/http"
	"sync"
	"time"
)

type Provider interface {
	Name() string
	Execute(req *http.Request, apiKey string) (*http.Response, error)
	IsQuotaError(resp *http.Response) bool
	GetQuotaInfo(resp *http.Response) (used, limit int64, err error)
}

type ProviderManager struct {
	providers map[string]Provider
	keys      []*models.APIKey
	mu        sync.RWMutex
}

func NewProviderManager() *ProviderManager {
	return &ProviderManager{
		providers: make(map[string]Provider),
		keys:       make([]*models.APIKey, 0),
	}
}

func (m *ProviderManager) RegisterProvider(p Provider) {
	m.providers[p.Name()] = p
}

func (m *ProviderManager) LoadAPIKeys() error {
	var keys []*models.APIKey
	if err := database.DB.Where("enabled = ?", true).Find(&keys).Error; err != nil {
		return err
	}
	m.mu.Lock()
	m.keys = keys
	m.mu.Unlock()
	return nil
}

// ReloadAPIKeys reloads API keys from database into memory cache
func (m *ProviderManager) ReloadAPIKeys() error {
	return m.LoadAPIKeys()
}

func (m *ProviderManager) GetAPIKeyForToken(tokenID string) (*models.APIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Find existing mapping
	var mapping models.TokenKeyMapping
	if err := database.DB.Where("token_id = ?", tokenID).First(&mapping).Error; err == nil {
		// Find the key
		for _, k := range m.keys {
			if k.ID == mapping.APIKeyID && k.Enabled && k.Healthy {
				return k, nil
			}
		}
	}

	// No mapping or key unavailable - assign new one based on hash
	if len(m.keys) == 0 {
		return nil, ErrNoAPIKeysAvailable
	}

	// Hash tokenID to select key index
	h := fnv.New32a()
	h.Write([]byte(tokenID))
	index := int(h.Sum32()) % len(m.keys)
	selectedKey := m.keys[index]

	// Check if selected key is healthy
	if !selectedKey.Healthy {
		// Try other keys in sequence
		for i := 1; i < len(m.keys); i++ {
			idx := (index + i) % len(m.keys)
			if m.keys[idx].Healthy && m.keys[idx].Enabled {
				selectedKey = m.keys[idx]
				break
			}
		}
		if !selectedKey.Healthy {
			return nil, ErrNoAPIKeysAvailable
		}
	}

	// Create mapping
	mapping = models.TokenKeyMapping{
		TokenID:    tokenID,
		APIKeyID:   selectedKey.ID,
		AssignedAt: time.Now(),
	}
	database.DB.Create(&mapping)

	return selectedKey, nil
}

func (m *ProviderManager) MarkKeyFailed(keyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var key models.APIKey
	if err := database.DB.First(&key, "id = ?", keyID).Error; err != nil {
		return err
	}

	now := time.Now()
	key.Healthy = false
	key.FailedAt = &now
	key.FailCount++

	// Update in-memory keys
	for _, k := range m.keys {
		if k.ID == keyID {
			k.Healthy = false
			k.FailedAt = &now
			k.FailCount++
			break
		}
	}

	return database.DB.Save(&key).Error
}

func (m *ProviderManager) MarkKeyHealthy(keyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var key models.APIKey
	if err := database.DB.First(&key, "id = ?", keyID).Error; err != nil {
		return err
	}

	key.Healthy = true
	key.FailedAt = nil
	key.FailCount = 0

	// Update in-memory keys
	for _, k := range m.keys {
		if k.ID == keyID {
			k.Healthy = true
			k.FailedAt = nil
			k.FailCount = 0
			break
		}
	}

	return database.DB.Save(&key).Error
}

func (m *ProviderManager) GetProvider(name string) Provider {
	return m.providers[name]
}

// GetProviderForModel selects the appropriate provider based on model name
// claude-* models go to anthropic, others go to minimax
func (m *ProviderManager) GetProviderForModel(model string) Provider {
	if len(model) >= 6 && model[:6] == "claude" {
		if p, ok := m.providers["anthropic"]; ok {
			return p
		}
	}
	// Default to minimax
	if p, ok := m.providers["minimax"]; ok {
		return p
	}
	return nil
}

var ErrNoAPIKeysAvailable = fmt.Errorf("no available API keys")