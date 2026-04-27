package line_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/claymorepriscilla/albert-morning-agent/internal/line"
)

func TestSend_PushMode(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		text       string
		wantErr    bool
		wantTo     string
		wantInBody string
	}{
		{
			name: "success — payload has correct to and message",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			text:       "สวัสดีตอนเช้า",
			wantTo:     "U123",
			wantInBody: "สวัสดีตอนเช้า",
		},
		{
			name: "error — server returns 400",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "bad request", http.StatusBadRequest)
			},
			text:    "hello",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedBody []byte
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedBody, _ = io.ReadAll(r.Body)
				// Verify Authorization header
				auth := r.Header.Get("Authorization")
				if auth != "Bearer test-token" {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				tt.handler(w, r)
			}))
			defer srv.Close()

			line.SetPushURL(srv.URL)
			c := line.NewClient("test-token", "U123", false)
			err := c.Send(tt.text)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify payload structure
			var payload map[string]any
			if err := json.Unmarshal(capturedBody, &payload); err != nil {
				t.Fatalf("invalid JSON payload: %v", err)
			}
			if payload["to"] != tt.wantTo {
				t.Errorf("payload[to] = %v, want %q", payload["to"], tt.wantTo)
			}
			msgs, _ := payload["messages"].([]any)
			if len(msgs) == 0 {
				t.Fatal("messages array is empty")
			}
			msg, _ := msgs[0].(map[string]any)
			if msg["text"] != tt.wantInBody {
				t.Errorf("message text = %v, want %q", msg["text"], tt.wantInBody)
			}
		})
	}
}

func TestSend_BroadcastMode(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantErr bool
	}{
		{
			name: "success — no to field in payload",
			handler: func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				var payload map[string]any
				json.Unmarshal(body, &payload)
				if _, hasTo := payload["to"]; hasTo {
					http.Error(w, "broadcast must not have to field", http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name: "error — server returns 403",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "forbidden", http.StatusForbidden)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			line.SetBroadcastURL(srv.URL)
			c := line.NewClient("test-token", "", true)
			err := c.Send("broadcast message")

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// TestSend_ServerUnavailable covers the HTTP transport error path.
func TestSend_ServerUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	line.SetPushURL(srv.URL)
	c := line.NewClient("token", "U1", false)

	if err := c.Send("hello"); err == nil {
		t.Fatal("expected error when server is unavailable, got nil")
	}
}

// TestSend_Race verifies no data races on concurrent sends.
// Run with: go test -race ./internal/line/...
func TestSend_Race(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	line.SetPushURL(srv.URL)
	c := line.NewClient("token", "U1", false)

	const goroutines = 8
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			c.Send("concurrent message") //nolint:errcheck
		}()
	}
	wg.Wait()
}
