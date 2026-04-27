package gemini_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/claymorepriscilla/albert-morning-agent/internal/gemini"
)

// groqResponse builds a valid Groq API JSON response.
func groqResponse(content string) string {
	b, _ := json.Marshal(map[string]any{
		"choices": []map[string]any{
			{"message": map[string]string{"content": content}},
		},
	})
	return string(b)
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		apiKey  string
		wantErr bool
	}{
		{name: "success — valid api key", apiKey: "valid-key", wantErr: false},
		{name: "error — empty api key", apiKey: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := gemini.NewClient(context.Background(), tt.apiKey)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c == nil {
				t.Fatal("expected non-nil client")
			}
			c.Close() // no-op, exercised for coverage
		})
	}
}

func TestSummarize(t *testing.T) {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		wantErr      bool
		wantContains string
	}{
		{
			name: "success — returns summary content",
			handler: func(w http.ResponseWriter, r *http.Request) {
				// Verify Authorization header is set
				if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				fmt.Fprint(w, groqResponse("📌 สรุปข่าว AI"))
			},
			wantContains: "📌 สรุปข่าว AI",
		},
		{
			name: "error — http 400 bad request",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "bad request", http.StatusBadRequest)
			},
			wantErr: true,
		},
		{
			name: "error — invalid json response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "not-json{{{")
			},
			wantErr: true,
		},
		{
			name: "error — empty choices array",
			handler: func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `{"choices":[]}`)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			gemini.SetEndpoint(srv.URL)

			c, err := gemini.NewClient(context.Background(), "test-key")
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}

			result, err := c.Summarize("AI", "- Headline 1\n- Headline 2", "27/04/2026")

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(result, tt.wantContains) {
				t.Errorf("result %q does not contain %q", result, tt.wantContains)
			}
		})
	}
}

// TestSummarize_ServerUnavailable covers the HTTP transport error path.
func TestSummarize_ServerUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // close immediately so the request fails

	gemini.SetEndpoint(srv.URL)
	c, _ := gemini.NewClient(context.Background(), "key")

	_, err := c.Summarize("topic", "news", "27/04/2026")
	if err == nil {
		t.Fatal("expected error when server is unavailable, got nil")
	}
}

// TestSummarize_Race verifies no data races on concurrent calls.
// Run with: go test -race ./internal/gemini/...
func TestSummarize_Race(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, groqResponse("📌 สรุป"))
	}))
	defer srv.Close()

	gemini.SetEndpoint(srv.URL)
	c, _ := gemini.NewClient(context.Background(), "key")

	const goroutines = 8
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			c.Summarize("AI", "news", "27/04/2026") //nolint:errcheck
		}()
	}
	wg.Wait()
}
