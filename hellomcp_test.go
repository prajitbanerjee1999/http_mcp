package main

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
	"time"

	mcp "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

func TestHttpRequestTool(t *testing.T) {
	// Build binary
	if err := exec.Command("go", "build", "-o", "hellomcp", "main.go").Run(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	cmd := exec.Command("./hellomcp")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout: %v", err)
	}
	stderr, _ := cmd.StderrPipe()

	go func() {
		buf := make([]byte, 1024)
		for {
			n, _ := stderr.Read(buf)
			if n > 0 {
				t.Logf("SERVER STDERR: %s", string(buf[:n]))
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer cmd.Process.Kill()

	client := mcp.NewClient(stdio.NewStdioServerTransportWithIO(stdout, stdin))

	// handshake
	initCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := client.Initialize(initCtx); err != nil {
		t.Fatalf("Client init failed: %v", err)
	}

	// Call http_request tool
	reqCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := client.CallTool(reqCtx, "http_request", map[string]interface{}{
		"method": "GET",
		"url":    "https://jsonplaceholder.typicode.com/posts/1",
	})
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}

	text := resp.Content[0].TextContent.Text
	fmt.Println("✅ Response:\n", text[:min(500, len(text))])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
