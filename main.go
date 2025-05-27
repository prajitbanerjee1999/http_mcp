package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	mcp "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

type HttpInput struct {
	Method string `json:"method" jsonschema:"required,description=HTTP method (GET/POST/etc.)"`
	URL    string `json:"url" jsonschema:"required,description=Target URL"`
	Body   string `json:"body,omitempty" jsonschema:"optional,description=Request body"`
}

func main() {
	log.SetOutput(os.Stderr)
	log.Println("✅ Starting MCP Server...")

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	server := mcp.NewServer(stdio.NewStdioServerTransport())

	if err := server.RegisterTool("http_request", "Send an HTTP request and return response", func(input HttpInput) (*mcp.ToolResponse, error) {
		req, err := http.NewRequest(strings.ToUpper(input.Method), input.URL, strings.NewReader(input.Body))
		if err != nil {
			return nil, fmt.Errorf("error creating request: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()
		req = req.WithContext(ctx)

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %v", err)
		}
		defer resp.Body.Close()

		var buf bytes.Buffer
		buf.WriteString(fmt.Sprintf("Status: %s\n\n", resp.Status))

		// Limit to 1 MB max
		limited := io.LimitReader(resp.Body, 1<<20)
		if _, err := io.Copy(&buf, limited); err != nil {
			return nil, fmt.Errorf("failed to read response: %v", err)
		}

		return mcp.NewToolResponse(mcp.NewTextContent(buf.String())), nil
	}); err != nil {
		log.Fatalf("❌ Tool registration failed: %v", err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Println("✅ MCP server ready and listening...")
		if err := server.Serve(); err != nil {
			log.Printf("❌ Server error: %v", err)
			stop <- syscall.SIGTERM
		}
	}()

	<-stop
	log.Println("🛑 MCP server shutting down.")
}
