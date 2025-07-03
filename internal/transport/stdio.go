package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// StdioTransport implements MCP over stdio with proper JSON-RPC protocol and debug logging
type StdioTransport struct {
	reader *bufio.Reader
	writer io.Writer
	mutex  sync.Mutex
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport() Transport {
	log.Printf("Creating new stdio transport")
	return &StdioTransport{
		reader: bufio.NewReader(os.Stdin),
		writer: os.Stdout,
	}
}

// Start begins listening for stdio messages
func (t *StdioTransport) Start(ctx context.Context, info ServerInfo, handler RequestHandler) error {
	log.Printf("Starting stdio transport, ready to receive messages...")
	
	// Channel for incoming messages
	msgChan := make(chan json.RawMessage, 10)
	errChan := make(chan error, 10)

	// Goroutine to read messages
	go func() {
		defer close(msgChan)
		defer close(errChan)
		defer log.Printf("Message reader goroutine exiting")
		
		log.Printf("Starting message reader loop...")
		messageCount := 0
		
		for {
			select {
			case <-ctx.Done():
				log.Printf("Context cancelled, stopping message reader")
				return
			default:
			}
			
			log.Printf("Attempting to read message #%d...", messageCount+1)
			msg, err := t.readMessage()
			messageCount++
			
			if err != nil {
				log.Printf("Error reading message #%d: %v", messageCount, err)
				if err == io.EOF {
					log.Printf("EOF reached, but keeping connection alive...")
					// Don't terminate on EOF - MCP clients may send intermittent data
					time.Sleep(100 * time.Millisecond)
					continue
				}
				errChan <- err
				continue
			}
			
			log.Printf("Successfully read message #%d: %s", messageCount, string(msg))
			
			select {
			case msgChan <- msg:
				log.Printf("Message #%d sent to processing channel", messageCount)
			case <-ctx.Done():
				log.Printf("Context cancelled while sending message #%d", messageCount)
				return
			}
		}
	}()

	log.Printf("Starting message processing loop...")
	processedCount := 0

	// Process messages
	for {
		select {
		case <-ctx.Done():
			log.Printf("Context cancelled, stopping transport. Processed %d messages total.", processedCount)
			return ctx.Err()
			
		case err := <-errChan:
			if err != nil {
				log.Printf("Error from message reader: %v", err)
				// Log error but continue
				continue
			}
			
		case msg, ok := <-msgChan:
			if !ok {
				log.Printf("Message channel closed, keeping server alive...")
				// Don't exit immediately - Claude may reconnect
				time.Sleep(5 * time.Second)
				continue
			}

			processedCount++
			log.Printf("=== PROCESSING MESSAGE #%d ===", processedCount)
			log.Printf("Raw message: %s", string(msg))

			// Handle message
			log.Printf("Calling handler for message #%d...", processedCount)
			response, err := handler(ctx, msg)
			
			if err != nil {
				log.Printf("Handler returned error for message #%d: %v", processedCount, err)
				// Send error response
				if sendErr := t.sendErrorResponse(err, nil); sendErr != nil {
					log.Printf("Failed to send error response: %v", sendErr)
				}
				continue
			}

			// Send response if there is one
			if response != nil {
				log.Printf("Sending response for message #%d: %s", processedCount, string(response))
				if err := t.sendMessage(response); err != nil {
					log.Printf("Failed to send response for message #%d: %v", processedCount, err)
					continue
				}
				log.Printf("Response sent successfully for message #%d", processedCount)
			} else {
				log.Printf("No response needed for message #%d", processedCount)
			}
			
			log.Printf("=== COMPLETED MESSAGE #%d ===", processedCount)
		}
	}
}

// readMessage reads a JSON-RPC message from stdin with auto-detection
func (t *StdioTransport) readMessage() (json.RawMessage, error) {
	log.Printf("Reading message with auto-detection...")
	
	// Peek first line to detect format
	log.Printf("Peeking first line to detect format...")
	firstLine, err := t.reader.ReadString('\n')
	if err != nil {
		log.Printf("Error reading first line: %v", err)
		return nil, err
	}
	
	trimmedLine := strings.TrimSpace(firstLine)
	log.Printf("First line: %q", trimmedLine)
	
	// Check if it's JSON (starts with {) or header (contains :)
	if strings.HasPrefix(trimmedLine, "{") {
		log.Printf("Detected direct JSON format")
		// Direct JSON - validate and return
		var temp interface{}
		if err := json.Unmarshal([]byte(trimmedLine), &temp); err != nil {
			log.Printf("ERROR: Invalid JSON: %v", err)
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
		log.Printf("Valid JSON message: %s", trimmedLine)
		return json.RawMessage(trimmedLine), nil
	}
	
	// Must be headers format - continue with header parsing
	log.Printf("Detected headers format, parsing...")
	headers := make(map[string]string)
	
	// Process first line as header
	if strings.Contains(trimmedLine, ":") {
		parts := strings.SplitN(trimmedLine, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			headers[key] = value
			log.Printf("Parsed header: %s = %s", key, value)
		}
	}
	
	// Read remaining headers
	for {
		line, err := t.reader.ReadString('\n')
		if err != nil {
			log.Printf("Error reading header: %v", err)
			return nil, err
		}

		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			log.Printf("Headers complete")
			break
		}

		parts := strings.SplitN(trimmedLine, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			headers[key] = value
			log.Printf("Parsed header: %s = %s", key, value)
		}
	}

	log.Printf("All headers: %+v", headers)

	// Get content length
	contentLengthStr, ok := headers["Content-Length"]
	if !ok {
		return nil, fmt.Errorf("missing Content-Length header")
	}

	var contentLength int
	if _, err := fmt.Sscanf(contentLengthStr, "%d", &contentLength); err != nil {
		return nil, fmt.Errorf("invalid Content-Length: %w", err)
	}

	log.Printf("Reading %d bytes of content...", contentLength)

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

	log.Printf("Successfully read message: %s", string(content))
	return json.RawMessage(content), nil
}

// sendMessage sends a JSON-RPC message to stdout in direct JSON format (no headers)
func (t *StdioTransport) sendMessage(msg json.RawMessage) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	log.Printf("Sending direct JSON message: %s", string(msg))

	// Validate the message is proper JSON
	var temp interface{}
	if err := json.Unmarshal(msg, &temp); err != nil {
		log.Printf("ERROR: Invalid JSON message to send: %v", err)
		return fmt.Errorf("invalid JSON message: %w", err)
	}

	// Send JSON directly - no headers (Claude Desktop expects this)
	log.Printf("Writing %d bytes of JSON content directly...", len(msg))
	if _, err := t.writer.Write(msg); err != nil {
		log.Printf("ERROR: Failed to write JSON message: %v", err)
		return err
	}
	
	// Add newline for proper message separation
	if _, err := t.writer.Write([]byte("\n")); err != nil {
		log.Printf("ERROR: Failed to write newline: %v", err)
		return err
	}

	// Force flush - critical for stdio transport
	log.Printf("Flushing output...")
	if f, ok := t.writer.(*os.File); ok {
		f.Sync()
	}

	log.Printf("Direct JSON message sent successfully")
	return nil
}

// sendErrorResponse sends a JSON-RPC error response
func (t *StdioTransport) sendErrorResponse(err error, id interface{}) error {
	log.Printf("Creating error response for error: %v, id: %v", err, id)
	
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
		log.Printf("ERROR: Failed to marshal error response: %v", marshalErr)
		return marshalErr
	}

	log.Printf("Sending error response: %s", string(data))
	return t.sendMessage(data)
}

// Stop closes the transport
func (t *StdioTransport) Stop() error {
	log.Printf("Stopping stdio transport")
	// Nothing to close for stdio
	return nil
}