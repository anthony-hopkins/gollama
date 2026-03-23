// Package main is the entry point for the gollama CLI application.
// It provides a simple command-line interface to interact with an LLM server.
package main

import (
	"context" // context is used to handle cancellation and timeouts for network requests.
	"flag"    // flag provides simple command-line argument parsing.
	"fmt"     // fmt is used for basic terminal input and output.
	"time"    // time is used for setting request timeouts.

	"gollama/client" // gollama/client is the custom package for interacting with the LLM API.
)

// main is the application's entry point. It sets up the CLI flags,
// validates user input, and dispatches the request based on the selected mode.
func main() {

	// Print a welcoming message.
	fmt.Println()
	fmt.Println("Welcome to DHS AI")
	fmt.Println()

	// --- CLI Flag Definitions ---
	// Define and initialize flags for mode, prompt, and baseURL.
	// Each flag is given a default value and a short description.
	mode := flag.String("mode", "nonstream", "Mode: stream or nonstream")
	prompt := flag.String("prompt", "", "Prompt to send to the model")
	baseURL := flag.String("baseURL", "", "Base URL of the llama server (e.g. http://localhost:8080)")

	// Parse the flags from the command-line arguments.
	flag.Parse()

	// --- Input Validation ---

	// Ensure the user provided a prompt.
	if *prompt == "" {
		panic("You must provide a prompt using --prompt \"your text here\"")
	}

	// Detect if the user forgot to quote a multi‑word prompt.
	// flag.Args() contains any arguments that weren't part of a defined flag.
	if len(flag.Args()) > 0 {
		panic("It looks like your --prompt value was not quoted. Use: --prompt \"your text here\"")
	}

	// Ensure the user provided the base URL of the LLM server.
	if *baseURL == "" {
		panic("You must provide a baseURL using --baseURL")
	}

	// Ensure the provided mode is one of the supported values.
	if *mode != "stream" && *mode != "nonstream" {
		panic("Invalid mode. Use --mode stream or --mode nonstream")
	}

	// Initialize the LLM client using the provided baseURL.
	c := client.New(*baseURL)

	// Execute the appropriate function based on the requested mode.
	switch *mode {
	case "stream":
		runStreaming(c, *prompt)
	case "nonstream":
		runNonStreaming(c, *prompt)
	}
}

// runNonStreaming sends a single prompt to the LLM and prints the full response at once.
func runNonStreaming(c *client.Client, prompt string) {
	// Create a context with a 30-second timeout for the network request.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel() // Ensure resources are cleaned up when the function returns.

	// Prepare the chat request structure.
	req := client.ChatRequest{
		Model: "qwen2.5:7b", // Hardcoded model for this version.
		Messages: []client.Message{
			{Role: "user", Content: prompt},
		},
	}

	// Make the synchronous API call.
	resp, err := c.Chat(ctx, req)
	if err != nil {
		panic(err) // Exit with an error if the request fails.
	}

	// Print the result.
	fmt.Println("Non‑streaming response:")
	fmt.Println(resp.Choices[0].Message.Content)
}

// runStreaming sends a prompt to the LLM and prints tokens to the terminal as they arrive.
func runStreaming(c *client.Client, prompt string) {
	// Create a context with a 30-second timeout for the network request.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel() // Ensure resources are cleaned up when the function returns.

	// Prepare the streaming chat request structure.
	req := client.ChatStreamRequest{
		Model: "qwen2.5:7b", // Hardcoded model for this version.
		Messages: []client.Message{
			{Role: "user", Content: prompt},
		},
		Stream: true, // Tell the API we want a streaming response.
	}

	fmt.Println("Streaming response:")
	// Call ChatStream with a callback function that handles each new token.
	err := c.ChatStream(ctx, req, func(token string) {
		fmt.Print(token) // Print each token immediately without a newline.
	})
	if err != nil {
		panic(err) // Exit with an error if the streaming fails.
	}

	// Add a final newline after the full stream is finished.
	fmt.Println()
}
