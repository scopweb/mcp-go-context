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
func (t *StdioTransport) readMessage() (json.RawMessage, error) {
	// Read headers
	headers := make(map[string]string)
	for {
		line, err := t.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	// Get content length
	contentLengthStr, ok := headers["Content-Length"]
	if !ok {
		return nil, fmt.Errorf("missing Content-Length header")
	}

	var contentLength int
	if _, err := fmt.Sscanf(contentLengthStr, "%d", &contentLength); err != nil {
		return nil, fmt.Errorf("invalid Content-Length: %w", err)
	}

	if contentLength <= 0 {
		return nil, fmt.Errorf("invalid content length: %d", contentLength)
	}

	// Read content
	content := make([]byte, contentLength)
	if _, err := io.ReadFull(t.reader, content); err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	// Validate JSON
	var temp interface{}
	if err := json.Unmarshal(content, &temp); err != nil {
		return nil, fmt.Errorf("invalid JSON content: %w", err)
	}

	return json.RawMessage(content), nil
}

// sendMessage sends a JSON-RPC message to stdout with proper formatting
func (t *StdioTransport) sendMessage(msg json.RawMessage) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Validate the message is proper JSON
	var temp interface{}
	if err := json.Unmarshal(msg, &temp); err != nil {
		return fmt.Errorf("invalid JSON message: %w", err)
	}

	// Write headers
	fmt.Fprintf(t.writer, "Content-Length: %d\r\n", len(msg))
	fmt.Fprintf(t.writer, "Content-Type: application/vnd.jsonrpc+json; charset=utf-8\r\n")
	fmt.Fprintf(t.writer, "\r\n")

	// Write content
	if _, err := t.writer.Write(msg); err != nil {
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
