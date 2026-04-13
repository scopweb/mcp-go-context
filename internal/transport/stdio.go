package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// StdioTransport implements MCP over stdio with proper JSON-RPC protocol
type StdioTransport struct {
	reader *bufio.Reader
	writer io.Writer
	mutex  sync.Mutex
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport() Transport {
	return &StdioTransport{
		reader: bufio.NewReader(os.Stdin),
		writer: os.Stdout,
	}
}

// Start begins listening for stdio messages
func (t *StdioTransport) Start(ctx context.Context, info ServerInfo, handler RequestHandler) error {
	// Read messages in a loop
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Read message
			msg, err := t.readMessage()
			if err != nil {
				if err == io.EOF {
					return nil
				}
				// Skip invalid messages, don't exit
				continue
			}

			// Handle message
			response, err := handler(ctx, msg)
			if err != nil {
				// Send error response
				t.sendErrorResponse(err, nil)
				continue
			}

			// Send response if there is one
			if response != nil {
				if err := t.sendMessage(response); err != nil {
					// Log error but continue
					continue
				}
			}
		}
	}
}

// readMessage reads a JSON-RPC message from stdin
// MCP stdio uses newline-delimited JSON messages (one JSON object per line)
func (t *StdioTransport) readMessage() (json.RawMessage, error) {
	// Read a line (message) from stdin
	line, err := t.reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return nil, err
		}
		return nil, fmt.Errorf("failed to read message: %w", err)
	}

	// Trim Windows CRLF and trailing newline
	line = strings.TrimSuffix(strings.TrimSuffix(line, "\r"), "\n")
	line = strings.TrimSpace(line)

	// Skip empty lines (common at startup)
	if line == "" {
		return nil, fmt.Errorf("empty line received")
	}

	// Validate it's valid JSON
	var temp interface{}
	if err := json.Unmarshal([]byte(line), &temp); err != nil {
		return nil, fmt.Errorf("invalid JSON message: %w", err)
	}

	return json.RawMessage(line), nil
}

// sendMessage sends a JSON-RPC message to stdout
// MCP stdio: messages are newline-delimited JSON, no headers
func (t *StdioTransport) sendMessage(msg json.RawMessage) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Validate the message is proper JSON
	var temp interface{}
	if err := json.Unmarshal(msg, &temp); err != nil {
		return fmt.Errorf("invalid JSON message: %w", err)
	}

	// Write JSON followed by newline (per MCP stdio spec)
	if _, err := t.writer.Write(msg); err != nil {
		return err
	}
	if _, err := t.writer.Write([]byte("\n")); err != nil {
		return err
	}

	// Ensure output is flushed
	if flusher, ok := t.writer.(interface{ Flush() error }); ok {
		return flusher.Flush()
	}

	return nil
}

// sendErrorResponse sends a JSON-RPC error response
func (t *StdioTransport) sendErrorResponse(err error, id interface{}) error {
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    -32603, // Internal error
			Message: err.Error(),
		},
	}

	data, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		return marshalErr
	}

	return t.sendMessage(data)
}

// Stop closes the transport
func (t *StdioTransport) Stop() error {
	// Nothing to close for stdio
	return nil
}
