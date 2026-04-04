package wsrelay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type HTTPRequest struct {
	Method  string
	URL     string
	Headers http.Header
	Body    []byte
}

type HTTPResponse struct {
	Status  int
	Headers http.Header
	Body    []byte
}

type StreamEvent struct {
	Type    string
	Payload []byte
	Status  int
	Headers http.Header
	Err     error
}

func (m *Manager) NonStream(ctx context.Context, provider string, req *HTTPRequest) (*HTTPResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("wsrelay: request is nil")
	}
	msg := Message{ID: uuid.NewString(), Type: MessageTypeHTTPReq, Payload: encodeRequest(req)}
	respCh, err := m.Send(ctx, provider, msg)
	if err != nil {
		return nil, err
	}
	var (
		streamMode bool
		streamResp *HTTPResponse
		streamBody bytes.Buffer
	)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case msg, ok := <-respCh:
			if !ok {
				if streamMode {
					if streamResp == nil {
						streamResp = &HTTPResponse{Status: http.StatusOK, Headers: make(http.Header)}
					} else if streamResp.Headers == nil {
						streamResp.Headers = make(http.Header)
					}
					streamResp.Body = append(streamResp.Body[:0], streamBody.Bytes()...)
					return streamResp, nil
				}
				return nil, errors.New("wsrelay: connection closed during response")
			}
			payload, _ := msg.Payload.(map[string]any)
			switch msg.Type {
			case MessageTypeHTTPResp:
				resp := decodeResponse(payload)
				if streamMode && streamBody.Len() > 0 && len(resp.Body) == 0 {
					resp.Body = append(resp.Body[:0], streamBody.Bytes()...)
				}
				return resp, nil
			case MessageTypeError:
				return nil, decodeError(payload)
			case MessageTypeStreamStart, MessageTypeStreamChunk:
				if msg.Type == MessageTypeStreamStart {
					streamMode = true
					streamResp = decodeResponse(payload)
					if streamResp.Headers == nil {
						streamResp.Headers = make(http.Header)
					}
					streamBody.Reset()
					continue
				}
				if !streamMode {
					streamMode = true
					streamResp = &HTTPResponse{Status: http.StatusOK, Headers: make(http.Header)}
				}
				chunk := decodeChunk(payload)
				if len(chunk) > 0 {
					streamBody.Write(chunk)
				}
			case MessageTypeStreamEnd:
				if !streamMode {
					return &HTTPResponse{Status: http.StatusOK, Headers: make(http.Header)}, nil
				}
				if streamResp == nil {
					streamResp = &HTTPResponse{Status: http.StatusOK, Headers: make(http.Header)}
				} else if streamResp.Headers == nil {
					streamResp.Headers = make(http.Header)
				}
				streamResp.Body = append(streamResp.Body[:0], streamBody.Bytes()...)
				return streamResp, nil
			}
		}
	}
}

func (m *Manager) Stream(ctx context.Context, provider string, req *HTTPRequest) (<-chan StreamEvent, error) {
	if req == nil {
		return nil, fmt.Errorf("wsrelay: request is nil")
	}
	msg := Message{ID: uuid.NewString(), Type: MessageTypeHTTPReq, Payload: encodeRequest(req)}
	respCh, err := m.Send(ctx, provider, msg)
	if err != nil {
		return nil, err
	}
	out := make(chan StreamEvent)
	go func() {
		defer close(out)
		send := func(ev StreamEvent) bool {
			if ctx == nil {
				out <- ev
				return true
			}
			select {
			case <-ctx.Done():
				return false
			case out <- ev:
				return true
			}
		}
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-respCh:
				if !ok {
					_ = send(StreamEvent{Err: errors.New("wsrelay: stream closed")})
					return
				}
				payload, _ := msg.Payload.(map[string]any)
				switch msg.Type {
				case MessageTypeStreamStart:
					resp := decodeResponse(payload)
					if okSend := send(StreamEvent{Type: MessageTypeStreamStart, Status: resp.Status, Headers: resp.Headers}); !okSend {
						return
					}
				case MessageTypeStreamChunk:
					chunk := decodeChunk(payload)
					if okSend := send(StreamEvent{Type: MessageTypeStreamChunk, Payload: chunk}); !okSend {
						return
					}
				case MessageTypeStreamEnd:
					_ = send(StreamEvent{Type: MessageTypeStreamEnd})
					return
				case MessageTypeError:
					_ = send(StreamEvent{Type: MessageTypeError, Err: decodeError(payload)})
					return
				case MessageTypeHTTPResp:
					resp := decodeResponse(payload)
					_ = send(StreamEvent{Type: MessageTypeHTTPResp, Status: resp.Status, Headers: resp.Headers, Payload: resp.Body})
					return
				}
			}
		}
	}()
	return out, nil
}

func encodeRequest(req *HTTPRequest) map[string]any {
	headers := make(map[string]any, len(req.Headers))
	for key, values := range req.Headers {
		copyValues := make([]string, len(values))
		copy(copyValues, values)
		headers[key] = copyValues
	}
	return map[string]any{
		"method":  req.Method,
		"url":     req.URL,
		"headers": headers,
		"body":    string(req.Body),
		"sent_at": time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func decodeResponse(payload map[string]any) *HTTPResponse {
	if payload == nil {
		return &HTTPResponse{Status: http.StatusBadGateway, Headers: make(http.Header)}
	}
	resp := &HTTPResponse{Status: http.StatusOK, Headers: make(http.Header)}
	if status, ok := payload["status"].(float64); ok {
		resp.Status = int(status)
	}
	if headers, ok := payload["headers"].(map[string]any); ok {
		for key, raw := range headers {
			switch v := raw.(type) {
			case []any:
				for _, item := range v {
					if str, ok := item.(string); ok {
						resp.Headers.Add(key, str)
					}
				}
			case []string:
				for _, str := range v {
					resp.Headers.Add(key, str)
				}
			case string:
				resp.Headers.Set(key, v)
			}
		}
	}
	if body, ok := payload["body"].(string); ok {
		resp.Body = []byte(body)
	}
	return resp
}

func decodeChunk(payload map[string]any) []byte {
	if payload == nil {
		return nil
	}
	if data, ok := payload["data"].(string); ok {
		return []byte(data)
	}
	return nil
}

func decodeError(payload map[string]any) error {
	if payload == nil {
		return errors.New("wsrelay: unknown error")
	}
	message, _ := payload["error"].(string)
	status := 0
	if v, ok := payload["status"].(float64); ok {
		status = int(v)
	}
	if message == "" {
		message = "wsrelay: upstream error"
	}
	return fmt.Errorf("%s (status=%d)", message, status)
}

func ParseHTTPRequest(payload map[string]any) *HTTPRequest {
	if payload == nil {
		return nil
	}
	req := &HTTPRequest{
		Headers: make(http.Header),
	}
	if method, ok := payload["method"].(string); ok {
		req.Method = method
	}
	if url, ok := payload["url"].(string); ok {
		req.URL = url
	}
	if body, ok := payload["body"].(string); ok {
		req.Body = []byte(body)
	}
	if headers, ok := payload["headers"].(map[string]any); ok {
		for key, raw := range headers {
			switch v := raw.(type) {
			case []any:
				for _, item := range v {
					if str, ok := item.(string); ok {
						req.Headers.Add(key, str)
					}
				}
			case []string:
				for _, str := range v {
					req.Headers.Add(key, str)
				}
			case string:
				req.Headers.Set(key, v)
			}
		}
	}
	return req
}

func EncodeResponse(resp *http.Response) map[string]any {
	if resp == nil {
		return nil
	}
	headers := make(map[string]any, len(resp.Header))
	for key, values := range resp.Header {
		headers[key] = values
	}
	var body string
	if resp.Body != nil {
		if data, err := io.ReadAll(resp.Body); err == nil {
			body = string(data)
		}
	}
	return map[string]any{
		"status":  float64(resp.StatusCode),
		"headers": headers,
		"body":    body,
	}
}

func EncodeStreamChunk(data []byte) map[string]any {
	return map[string]any{"data": string(data)}
}

func EncodeStreamStart(resp *http.Response) map[string]any {
	return EncodeResponse(resp)
}

func ParseStreamChunk(payload map[string]any) []byte {
	return decodeChunk(payload)
}

func HTTPResponseToMap(resp *http.Response) map[string]any {
	headers := make(map[string]any)
	for key, values := range resp.Header {
		headers[key] = values
	}
	var bodyStr string
	if resp.Body != nil {
		if body, err := io.ReadAll(resp.Body); err == nil {
			bodyStr = string(body)
		}
	}
	return map[string]any{
		"status":  float64(resp.StatusCode),
		"headers": headers,
		"body":    bodyStr,
	}
}

func MarshalJSON(v map[string]any) []byte {
	data, _ := json.Marshal(v)
	return data
}