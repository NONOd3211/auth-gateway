package wsrelay

import (
	"auth-gateway/database"
	"auth-gateway/models"
	"auth-gateway/providers"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Server struct {
	manager        *Manager
	providerManager *providers.ProviderManager
}

func NewServer(pm *providers.ProviderManager) *Server {
	mgr := NewManager(Options{
		Path: "/",
		OnConnected: func(provider string) {
			log.Printf("[WS] Provider connected: %s", provider)
		},
		OnDisconnected: func(provider string, err error) {
			log.Printf("[WS] Provider disconnected: %s, err: %v", provider, err)
		},
	})
	return &Server{
		manager:        mgr,
		providerManager: pm,
	}
}

func (s *Server) Handler() http.Handler {
	return s.manager.Handler()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tokenID := r.URL.Query().Get("token")
	if tokenID == "" {
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			tokenID = strings.TrimPrefix(auth, "Bearer ")
		}
	}
	if tokenID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var token models.Token
	if err := database.DB.First(&token, "id = ?", tokenID).Error; err != nil {
		http.Error(w, "token not found", http.StatusUnauthorized)
		return
	}
	if token.IsExpired() {
		http.Error(w, "token expired", http.StatusForbidden)
		return
	}
	if !token.IsWithinWeeklyLimit() {
		http.Error(w, "weekly limit exceeded", http.StatusForbidden)
		return
	}
	token.CheckAndUpdateLimits()
	database.DB.Save(&token)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("[WS] Client connected: token=%s", token.ID)
	s.handleConnection(conn, &token)
}

func (s *Server) handleConnection(conn *websocket.Conn, token *models.Token) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		_, msgData, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WS] Read error: %v", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(msgData, &msg); err != nil {
			log.Printf("[WS] Failed to parse message: %v", err)
			continue
		}

		switch msg.Type {
		case MessageTypeHTTPReq:
			s.handleHTTPRequest(ctx, conn, &msg, token)
		case MessageTypePing:
			conn.WriteJSON(Message{ID: msg.ID, Type: MessageTypePong})
		default:
			log.Printf("[WS] Unknown message type: %s", msg.Type)
		}
	}
}

func (s *Server) handleHTTPRequest(ctx context.Context, conn *websocket.Conn, msg *Message, token *models.Token) {
	payload, ok := msg.Payload.(map[string]any)
	if !ok {
		s.sendError(conn, msg.ID, "invalid payload")
		return
	}

	httpReq := ParseHTTPRequest(payload)
	if httpReq == nil {
		s.sendError(conn, msg.ID, "failed to parse request")
		return
	}

	log.Printf("[WS] Request: method=%s url=%s", httpReq.Method, httpReq.URL)

	apiKey, err := s.providerManager.GetAPIKeyForToken(token.ID, token.APIKeyID)
	if err != nil {
		s.sendError(conn, msg.ID, "no available API keys")
		return
	}

	proxyReq, err := http.NewRequest(httpReq.Method, httpReq.URL, bytes.NewReader(httpReq.Body))
	if err != nil {
		s.sendError(conn, msg.ID, "failed to build request")
		return
	}

	for key, values := range httpReq.Headers {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	if apiKey.Key != "" {
		proxyReq.Header.Set("Authorization", "Bearer "+apiKey.Key)
	}

	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(proxyReq)
	if err != nil {
		s.sendError(conn, msg.ID, "upstream error: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[WS] Upstream error: status=%d body=%s", resp.StatusCode, string(body))
		s.sendError(conn, msg.ID, fmt.Sprintf("upstream error: status %d", resp.StatusCode))
		return
	}

	contentType := resp.Header.Get("Content-Type")
	isStreaming := strings.Contains(contentType, "text/event-stream") ||
		strings.Contains(contentType, "application/x-ndjson")

	if err := conn.WriteJSON(Message{
		ID:   msg.ID,
		Type: MessageTypeStreamStart,
		Payload: map[string]any{
			"status":  float64(resp.StatusCode),
			"headers": headerToMap(resp.Header),
		},
	}); err != nil {
		log.Printf("[WS] Failed to send stream start: %v", err)
		return
	}

	if isStreaming {
		s.streamResponse(ctx, conn, msg.ID, resp, token)
	} else {
		s.sendNonStreamingResponse(conn, msg.ID, resp)
	}
}

func (s *Server) streamResponse(ctx context.Context, conn *websocket.Conn, msgID string, resp *http.Response, token *models.Token) {
	var reader io.Reader
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			log.Printf("[WS] Failed to create gzip reader: %v", err)
			return
		}
		defer gzipReader.Close()
		reader = gzipReader
	} else {
		reader = resp.Body
	}

	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			if err := conn.WriteJSON(Message{
				ID:   msgID,
				Type: MessageTypeStreamChunk,
				Payload: map[string]any{
					"data": string(chunk),
				},
			}); err != nil {
				log.Printf("[WS] Failed to send chunk: %v", err)
				return
			}
		}
		if err != nil {
			if err != io.EOF {
				log.Printf("[WS] Read error: %v", err)
			}
			break
		}
	}

	conn.WriteJSON(Message{
		ID:   msgID,
		Type: MessageTypeStreamEnd,
	})

	log.Printf("[WS] Stream completed: token=%s", token.ID)
}

func (s *Server) sendNonStreamingResponse(conn *websocket.Conn, msgID string, resp *http.Response) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.sendError(conn, msgID, "failed to read response")
		return
	}

	conn.WriteJSON(Message{
		ID:   msgID,
		Type: MessageTypeHTTPResp,
		Payload: map[string]any{
			"status":  float64(resp.StatusCode),
			"headers": headerToMap(resp.Header),
			"body":    string(body),
		},
	})
}

func (s *Server) sendError(conn *websocket.Conn, msgID string, errMsg string) {
	conn.WriteJSON(Message{
		ID:   msgID,
		Type: MessageTypeError,
		Payload: map[string]any{
			"error": errMsg,
		},
	})
}

func headerToMap(h http.Header) map[string]any {
	result := make(map[string]any)
	for key, values := range h {
		result[key] = values
	}
	return result
}