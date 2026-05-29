package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
)

// StdioTransport implements MCP over stdio with proper JSON-RPC protocol
// Supports both HTTP-style headers (Claude Desktop) and simple LDJ mode
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
// Supports both HTTP-style headers (Claude Desktop) and simple LDJ mode
func (t *StdioTransport) readMessage() (json.RawMessage, error) {
	// First, peek to see if we have HTTP headers or direct JSON
	peek, err := t.reader.Peek(1)
	if err != nil {
		if err == io.EOF {
			return nil, err
		}
		return nil, fmt.Errorf("failed to peek: %w", err)
	}

	// Check if it starts with '{' (direct JSON - LDJ mode)
	if peek[0] == '{' {
		return t.readLDJMessage()
	}

	// Otherwise, try HTTP-style with Content-Length headers
	return t.readHTTPMessage()
}

// readLDJMessage reads a simple newline-delimited JSON message
func (t *StdioTransport) readLDJMessage() (json.RawMessage, error) {
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

	// Skip empty lines
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

// readHTTPMessage reads HTTP-style message with Content-Length header
func (t *StdioTransport) readHTTPMessage() (json.RawMessage, error) {
	headers := make(map[string]string)

	// Read headers
	for {
		line, err := t.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil, err
			}
			return nil, fmt.Errorf("failed to read header: %w", err)
		}

		// Trim CRLF and newline
		line = strings.TrimSuffix(strings.TrimSuffix(line, "\r"), "\n")

		// Empty line marks end of headers
		if line == "" {
			break
		}

		// Parse header: "Header-Name: value"
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	// Get Content-Length
	contentLengthStr, ok := headers["Content-Length"]
	if !ok {
		return nil, fmt.Errorf("missing Content-Length header")
	}

	contentLength, err := strconv.Atoi(contentLengthStr)
	if err != nil || contentLength <= 0 {
		return nil, fmt.Errorf("invalid Content-Length: %s", contentLengthStr)
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

// sendMessage sends a JSON-RPC message to stdout
// Uses LDJ mode (newline-delimited JSON) which is the MCP stdio standard
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

	// Ensure output is flushed immediately
	if f, ok := t.writer.(interface{ Flush() error }); ok {
		if err := f.Flush(); err != nil {
			return err
		}
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
