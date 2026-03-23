package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"gollama/client"
)

func main() {

	fmt.Println()
	fmt.Println("Welcome to DHS AI")
	fmt.Println()

	// Flags
	mode := flag.String("mode", "nonstream", "Mode: stream or nonstream")
	prompt := flag.String("prompt", "", "Prompt to send to the model")
	baseURL := flag.String("baseURL", "", "Base URL of the llama server (e.g. http://localhost:8080)")
	flag.Parse()

	// --- Validation ---
	if *prompt == "" {
		panic("You must provide a prompt using --prompt")
	}

	if *baseURL == "" {
		panic("You must provide a baseURL using --baseURL")
	}

	if *mode != "stream" && *mode != "nonstream" {
		panic("Invalid mode. Use --mode stream or --mode nonstream")
	}

	// Create client using provided baseURL
	c := client.New(*baseURL)

	switch *mode {
	case "stream":
		runStreaming(c, *prompt)
	case "nonstream":
		runNonStreaming(c, *prompt)
	}
}

func runNonStreaming(c *client.Client, prompt string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := client.ChatRequest{
		Model: "qwen2.5:7b",
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

func runStreaming(c *client.Client, prompt string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := client.ChatStreamRequest{
		Model: "qwen2.5:7b",
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
