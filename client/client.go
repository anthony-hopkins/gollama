package client

import (
	"bufio"         // Used for reading the streaming response line-by-line.
	"bytes"         // Used for handling byte slices and creating buffers.
	"context"       // Used for managing request lifecycles (cancellation and timeouts).
	"encoding/json" // Used for converting Go structs to JSON and vice-versa.
	"errors"        // Used for error handling and comparisons.
	"fmt"           // Used for formatted I/O and creating error messages.
	"io"            // Used for basic I/O primitives.
	"net"           // Used for network-related settings like dialers.
	"net/http"      // The core package for making HTTP requests.
	"time"          // Used for defining durations and timeouts.
)

//
// ────────────────────────────────────────────────────────────────
//   CLIENT STRUCT & CONSTRUCTOR
// ────────────────────────────────────────────────────────────────
//

// Client acts as the central hub for making requests to the LLM.
// It keeps track of the server's address and holds a reusable HTTP client.
type Client struct {
	BaseURL string       // The base address of the API (e.g., http://localhost:8080).
	HTTP    *http.Client // The underlying Go HTTP client that performs the network work.
}

// New is a "constructor" function. In Go, it's common practice to have a function
// that sets up a struct with sensible default values.
func New(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTP: &http.Client{
			// Timeout is the total time allowed for a single request.
			// 60 seconds is usually enough for a standard LLM response.
			Timeout: 60 * time.Second,

			// Transport defines how the underlying network connections are handled.
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					// How long to wait for the initial TCP connection.
					Timeout: 5 * time.Second,
					// Keep-alive helps reuse connections, improving performance.
					KeepAlive: 30 * time.Second,
				}).DialContext,

				// Connection pooling: these settings allow the client to reuse
				// existing connections instead of opening new ones every time.
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
			},
		},
	}
}

//
// ────────────────────────────────────────────────────────────────
//   REQUEST / RESPONSE DATA STRUCTURES
// ────────────────────────────────────────────────────────────────
//

// Message represents a single "turn" in the conversation.
// The `json:"..."` tags tell the json package how to map Go fields to JSON keys.
type Message struct {
	Role    string `json:"role"`    // Who is speaking? (e.g., "system", "user", "assistant").
	Content string `json:"content"` // The actual text of the message.
}

// ChatRequest is the payload sent to the server for a normal chat completion.
type ChatRequest struct {
	Model     string    `json:"model"`                // Which model should the server use?
	Messages  []Message `json:"messages"`             // The history of the conversation.
	MaxTokens int       `json:"max_tokens,omitempty"` // Limit on response length.
}

// ChatResponse matches the JSON structure returned by the server.
// We only define the fields we actually need to read.
type ChatResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// ChatStreamRequest is used specifically for streaming.
// It includes a "Stream" flag to tell the server to send data piece by piece.
type ChatStreamRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

//
// ────────────────────────────────────────────────────────────────
//   NON‑STREAMING CHAT (Standard Request/Response)
// ────────────────────────────────────────────────────────────────
//

// Chat performs a "blocking" request. It sends the prompt and waits until
// the model has finished generating the entire response before returning.
func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// 1. Convert our Go struct into a JSON byte slice.
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	// 2. Prepare the HTTP POST request.
	// NewRequestWithContext allows us to cancel the request if it takes too long.
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.BaseURL+"/v1/chat/completions",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, fmt.Errorf("request creation error: %w", err)
	}

	// 3. Tell the server we are sending JSON data.
	httpReq.Header.Set("Content-Type", "application/json")

	// 4. Send the request and get the response.
	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	// Always close the body to free up network resources.
	defer resp.Body.Close()

	// 5. Handle errors from the server (e.g., 404 or 500).
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, string(b))
	}

	// 6. Decode the JSON response body into our Go struct.
	var out ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}

	return &out, nil
}

//
// ────────────────────────────────────────────────────────────────
//   STREAMING CHAT (Server-Sent Events)
// ────────────────────────────────────────────────────────────────
//

// ChatStream uses "Server-Sent Events" (SSE) to receive the response bit-by-bit.
// This is great for UX because the user sees the model "typing" in real-time.
func (c *Client) ChatStream(
	ctx context.Context,
	req ChatStreamRequest,
	onToken func(token string), // Callback function called for every new word/token.
) error {

	// 1. Prepare the request (similar to Chat).
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.BaseURL+"/v1/chat/completions",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return fmt.Errorf("request creation error: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// 2. Execute the request.
	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error: %s", string(b))
	}

	// 3. Instead of decoding everything at once, we use a bufio.Reader
	// to read the response stream line-by-line.
	reader := bufio.NewReader(resp.Body)

	for {
		select {
		case <-ctx.Done():
			// If the user cancels the context, we stop reading and return.
			return ctx.Err()

		default:
			// Read the stream until we hit a newline character.
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil // End of the stream reached.
				}
				return fmt.Errorf("stream read error: %w", err)
			}

			// SSE format uses "data: " as a prefix for each JSON chunk.
			if !bytes.HasPrefix(line, []byte("data: ")) {
				continue // Skip empty lines or metadata.
			}

			// Strip the "data: " prefix to get the raw JSON.
			payload := bytes.TrimPrefix(line, []byte("data: "))

			// The server sends "[DONE]" when the generation is complete.
			if bytes.Equal(payload, []byte("[DONE]")) {
				return nil
			}

			// Define a small temporary struct to parse this specific chunk.
			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}

			// Parse the JSON chunk.
			if err := json.Unmarshal(payload, &chunk); err != nil {
				continue // If a line is malformed, just try the next one.
			}

			// If we found a new piece of text (a "token"), pass it to the callback.
			if len(chunk.Choices) > 0 {
				token := chunk.Choices[0].Delta.Content
				if token != "" {
					onToken(token)
				}
			}
		}
	}
}
