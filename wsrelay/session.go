package wsrelay

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	PingInterval        = 30 * time.Second
	ReadTimeout         = 60 * time.Second
	WriteTimeout        = 10 * time.Second
	MaxMessageSize      = 64 * 1024 * 1024
	PendingRequestBuffer = 8
)

type pendingRequest struct {
	ch     chan Message
	closed atomic.Bool
}

func (pr *pendingRequest) close() {
	if pr.closed.CompareAndSwap(false, true) {
		close(pr.ch)
	}
}

type session struct {
	conn    *websocket.Conn
	manager *Manager
	id      string
	provider string
	closed     chan struct{}
	closeOnce  sync.Once
	writeMutex sync.Mutex
	pending    sync.Map
}

func newSession(conn *websocket.Conn, mgr *Manager, id string) *session {
	s := &session{
		conn:    conn,
		manager: mgr,
		id:      id,
		closed:  make(chan struct{}),
	}
	return s
}

func (s *session) run(ctx context.Context) {
	defer s.cleanup(errClosed)
	go s.startHeartbeat(ctx)
	for {
		var msg Message
		if err := s.conn.ReadJSON(&msg); err != nil {
			s.cleanup(err)
			return
		}
		s.dispatch(msg)
	}
}

func (s *session) dispatch(msg Message) {
	switch msg.Type {
	case MessageTypePing:
		_ = s.send(context.Background(), Message{ID: msg.ID, Type: MessageTypePong})
		return
	}
	if value, ok := s.pending.Load(msg.ID); ok {
		req := value.(*pendingRequest)
		select {
		case req.ch <- msg:
		default:
		}
		if msg.Type == MessageTypeHTTPResp || msg.Type == MessageTypeError || msg.Type == MessageTypeStreamEnd {
			if actual, loaded := s.pending.LoadAndDelete(msg.ID); loaded {
				actual.(*pendingRequest).close()
			}
		}
		return
	}
}

func (s *session) startHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(PingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.closed:
			return
		case <-ticker.C:
			if err := s.send(ctx, Message{Type: MessageTypePing}); err != nil {
				return
			}
		}
	}
}

func (s *session) send(ctx context.Context, msg Message) error {
	s.writeMutex.Lock()
	defer s.writeMutex.Unlock()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closed:
		return errClosed
	default:
	}
	s.conn.SetWriteDeadline(time.Now().Add(WriteTimeout))
	return s.conn.WriteJSON(msg)
}

func (s *session) request(ctx context.Context, msg Message) (<-chan Message, error) {
	if s.closed == nil {
		return nil, errors.New("session is closed")
	}
	req := &pendingRequest{ch: make(chan Message, PendingRequestBuffer)}
	if _, loaded := s.pending.LoadOrStore(msg.ID, req); loaded {
		return nil, errors.New("duplicate message ID")
	}
	if err := s.send(ctx, msg); err != nil {
		s.pending.Delete(msg.ID)
		return nil, err
	}
	return req.ch, nil
}

func (s *session) cleanup(err error) {
	s.closeOnce.Do(func() {
		close(s.closed)
		s.pending.Range(func(key, value interface{}) bool {
			if req, ok := value.(*pendingRequest); ok {
				req.close()
			}
			return true
		})
		if s.conn != nil {
			s.conn.Close()
		}
		if s.manager != nil {
			s.manager.handleSessionClosed(s, err)
		}
	})
}

var errClosed = errors.New("session closed")