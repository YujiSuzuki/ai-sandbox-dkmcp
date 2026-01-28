package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func main() {
	url := "http://host.docker.internal:8080/sse"
	fmt.Printf("Connecting to: %s\n", url)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	client := &http.Client{}
	fmt.Println("Sending request...")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error connecting: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Connected! Status: %d\n", resp.StatusCode)
	fmt.Println("Headers:")
	for k, v := range resp.Header {
		fmt.Printf("  %s: %v\n", k, v)
	}
	fmt.Println("\nReading SSE stream:")

	scanner := bufio.NewScanner(resp.Body)
	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineCount++
		fmt.Printf("[Line %d] %s\n", lineCount, line)

		if strings.HasPrefix(line, "event:") {
			fmt.Printf("  -> Found event: %s\n", strings.TrimPrefix(line, "event:"))
		}
		if strings.HasPrefix(line, "data:") {
			fmt.Printf("  -> Found data: %s\n", strings.TrimPrefix(line, "data:"))
		}

		// Stop after reading endpoint event
		if lineCount > 10 {
			fmt.Println("\nStopping after 10 lines...")
			break
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Scanner error: %v\n", err)
	}

	if lineCount == 0 {
		fmt.Println("No data received from SSE stream")
	}

	fmt.Println("\nTest complete")
}
