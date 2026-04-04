package wsrelay

// Message types for WebSocket communication
const (
	MessageTypeHTTPReq      = "http_req"
	MessageTypeHTTPResp     = "http_resp"
	MessageTypeStreamStart  = "stream_start"
	MessageTypeStreamChunk  = "stream_chunk"
	MessageTypeStreamEnd    = "stream_end"
	MessageTypeError        = "error"
	MessageTypePing         = "ping"
	MessageTypePong         = "pong"
)

// Message represents a WebSocket message
type Message struct {
	ID      string      `json:"id"`
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}