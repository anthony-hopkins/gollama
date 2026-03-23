// Package main is the entry point for the gollama CLI application.
// It provides a simple command-line interface to interact with an LLM server.
package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"gollama/client"
)

// main is the application's entry point. It sets up the CLI flags,
// validates user input, and dispatches the request based on the selected mode.
func main() {

	fmt.Println()
	fmt.Println("Welcome to DHS AI")
	fmt.Println()

	// --- CLI Flag Definitions ---
	mode := flag.String("mode", "nonstream", "Mode: stream or nonstream")
	prompt := flag.String("prompt", "", "Prompt to send to the model")
	baseURL := flag.String("baseURL", "", "Base URL of the llama server (e.g. http://localhost:8080)")

	flag.Parse()

	// --- Input Validation ---

	// Require prompt
	if *prompt == "" {
		panic("You must provide a prompt using --prompt \"your text here\"")
	}

	// Detect unquoted multi‑word prompt
	if len(flag.Args()) > 0 {
		panic("It looks like your --prompt value was not quoted. Use: --prompt \"your text here\"")
	}

	// Require baseURL
	if *baseURL == "" {
		panic("You must provide a baseURL using --baseURL")
	}

	// Validate mode
	if *mode != "stream" && *mode != "nonstream" {
		panic("Invalid mode. Use --mode stream or --mode nonstream")
	}

	// Initialize client
	c := client.New(*baseURL)

	// Dispatch
	switch *mode {
	case "stream":
		runStreaming(c, *prompt)
	case "nonstream":
		runNonStreaming(c, *prompt)
	}
}

// runNonStreaming sends a single prompt to the LLM and prints the full response at once.
func runNonStreaming(c *client.Client, prompt string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	req := client.ChatRequest{
		Model: "default", // ignored by llama.cpp server
		Messages: []client.Message{
			{Role: "user", Content: prompt},
		},
	}

	resp, err := c.Chat(ctx, req)
	if err != nil {
		panic(err)
	}

	fmt.Println("Non‑streaming response:")
	fmt.Println(resp.Choices[0].Message.Content)
}

// runStreaming sends a prompt to the LLM and prints tokens to the terminal as they arrive.
func runStreaming(c *client.Client, prompt string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	req := client.ChatStreamRequest{
		Model: "default", // ignored by llama.cpp server
		Messages: []client.Message{
			{Role: "user", Content: prompt},
		},
		Stream: true,
	}

	fmt.Println("Streaming response:")
	err := c.ChatStream(ctx, req, func(token string) {
		fmt.Print(token)
	})
	if err != nil {
		panic(err)
	}

	fmt.Println()
}
