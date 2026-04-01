package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

// StdioTransport implements MCP over stdio.
type StdioTransport struct {
	reader *bufio.Reader
	writer io.Writer
	mutex  sync.Mutex
}

// NewStdioTransport creates a new stdio transport.
func NewStdioTransport() Transport {
	return &StdioTransport{
		reader: bufio.NewReader(os.Stdin),
		writer: os.Stdout,
	}
}

// Start begins listening for stdio messages.
func (t *StdioTransport) Start(ctx context.Context, info ServerInfo, handler RequestHandler) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		msg, err := t.readMessage()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
				return nil
			}
			log.Printf("stdio read error: %v", err)
			continue
		}

		response, err := handler(ctx, msg)
		if err != nil {
			if sendErr := t.sendErrorResponse(err, nil); sendErr != nil {
				return sendErr
			}
			continue
		}

		if response == nil {
			continue
		}

		if err := t.sendMessage(response); err != nil {
			return err
		}
	}
}

// readMessage reads a JSON-RPC message from stdin with auto-detection.
func (t *StdioTransport) readMessage() (json.RawMessage, error) {
	for {
		firstLine, err := t.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		trimmedLine := strings.TrimSpace(firstLine)
		if trimmedLine == "" {
			continue
		}

		if strings.HasPrefix(trimmedLine, "{") {
			var temp interface{}
			if err := json.Unmarshal([]byte(trimmedLine), &temp); err != nil {
				return nil, fmt.Errorf("invalid JSON: %w", err)
			}
			return json.RawMessage(trimmedLine), nil
		}

		headers := make(map[string]string)
		if strings.Contains(trimmedLine, ":") {
			parts := strings.SplitN(trimmedLine, ":", 2)
			if len(parts) == 2 {
				headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		} else {
			return nil, fmt.Errorf("invalid stdio message preamble: %q", trimmedLine)
		}

		for {
			line, err := t.reader.ReadString('\n')
			if err != nil {
				return nil, err
			}

			trimmedHeader := strings.TrimSpace(line)
			if trimmedHeader == "" {
				break
			}

			parts := strings.SplitN(trimmedHeader, ":", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid header: %q", trimmedHeader)
			}
			headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}

		contentLengthStr, ok := headers["Content-Length"]
		if !ok {
			return nil, fmt.Errorf("missing Content-Length header")
		}

		var contentLength int
		if _, err := fmt.Sscanf(contentLengthStr, "%d", &contentLength); err != nil {
			return nil, fmt.Errorf("invalid Content-Length: %w", err)
		}

		content := make([]byte, contentLength)
		if _, err := io.ReadFull(t.reader, content); err != nil {
			return nil, fmt.Errorf("failed to read content: %w", err)
		}

		var temp interface{}
		if err := json.Unmarshal(content, &temp); err != nil {
			return nil, fmt.Errorf("invalid JSON content: %w", err)
		}

		return json.RawMessage(content), nil
	}
}

// sendMessage sends a JSON-RPC message to stdout in direct JSON format.
func (t *StdioTransport) sendMessage(msg json.RawMessage) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	var temp interface{}
	if err := json.Unmarshal(msg, &temp); err != nil {
		return fmt.Errorf("invalid JSON message: %w", err)
	}

	if _, err := t.writer.Write(msg); err != nil {
		return err
	}

	if _, err := t.writer.Write([]byte("\n")); err != nil {
		return err
	}

	return nil
}

// sendErrorResponse sends a JSON-RPC error response.
func (t *StdioTransport) sendErrorResponse(err error, id interface{}) error {
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    -32603,
			Message: err.Error(),
		},
	}

	data, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		return marshalErr
	}

	return t.sendMessage(data)
}

// Stop closes the transport.
func (t *StdioTransport) Stop() error {
	return nil
}
