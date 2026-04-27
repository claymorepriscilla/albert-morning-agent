package config_test

import (
	"os"
	"strings"
	"testing"

	"github.com/claymorepriscilla/albert-morning-agent/internal/config"
)

// relevantKeys are all env vars managed by this package.
var relevantKeys = []string{
	"GROQ_API_KEY",
	"LINE_CHANNEL_ACCESS_TOKEN",
	"LINE_USER_ID",
	"LINE_BROADCAST",
}

// isolateEnv clears all relevant env vars and restores them after the test.
func isolateEnv(t *testing.T, env map[string]string) {
	t.Helper()
	type saved struct {
		val string
		set bool
	}
	snapshot := make(map[string]saved, len(relevantKeys))
	for _, k := range relevantKeys {
		v, exists := os.LookupEnv(k)
		snapshot[k] = saved{val: v, set: exists}
		os.Unsetenv(k)
	}
	for k, v := range env {
		os.Setenv(k, v)
	}
	t.Cleanup(func() {
		for k, s := range snapshot {
			if s.set {
				os.Setenv(k, s.val)
			} else {
				os.Unsetenv(k)
			}
		}
	})
}

func TestLoad(t *testing.T) {
	base := map[string]string{
		"GROQ_API_KEY":             "test-groq-key",
		"LINE_CHANNEL_ACCESS_TOKEN": "test-token",
		"LINE_USER_ID":             "U123456",
	}

	tests := []struct {
		name          string
		env           map[string]string
		wantErr       bool
		errContains   string
		wantBroadcast bool
		wantGroqKey   string
	}{
		{
			name:          "success — push mode all vars set",
			env:           base,
			wantErr:       false,
			wantBroadcast: false,
			wantGroqKey:   "test-groq-key",
		},
		{
			name: "success — broadcast mode LINE_USER_ID not required",
			env: map[string]string{
				"GROQ_API_KEY":             "key",
				"LINE_CHANNEL_ACCESS_TOKEN": "token",
				"LINE_BROADCAST":           "true",
			},
			wantErr:       false,
			wantBroadcast: true,
		},
		{
			name: "success — LINE_BROADCAST false still requires USER_ID",
			env: map[string]string{
				"GROQ_API_KEY":             "key",
				"LINE_CHANNEL_ACCESS_TOKEN": "token",
				"LINE_USER_ID":             "U999",
				"LINE_BROADCAST":           "false",
			},
			wantErr:       false,
			wantBroadcast: false,
		},
		{
			name:        "error — missing GROQ_API_KEY",
			env:         map[string]string{"LINE_CHANNEL_ACCESS_TOKEN": "token", "LINE_USER_ID": "U1"},
			wantErr:     true,
			errContains: "GROQ_API_KEY",
		},
		{
			name:        "error — missing LINE_CHANNEL_ACCESS_TOKEN",
			env:         map[string]string{"GROQ_API_KEY": "key", "LINE_USER_ID": "U1"},
			wantErr:     true,
			errContains: "LINE_CHANNEL_ACCESS_TOKEN",
		},
		{
			name:        "error — missing LINE_USER_ID in push mode",
			env:         map[string]string{"GROQ_API_KEY": "key", "LINE_CHANNEL_ACCESS_TOKEN": "token"},
			wantErr:     true,
			errContains: "LINE_USER_ID",
		},
		{
			name: "error — LINE_BROADCAST false without USER_ID",
			env: map[string]string{
				"GROQ_API_KEY":             "key",
				"LINE_CHANNEL_ACCESS_TOKEN": "token",
				"LINE_BROADCAST":           "false",
			},
			wantErr:     true,
			errContains: "LINE_USER_ID",
		},
		{
			name:    "error — all vars missing",
			env:     map[string]string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateEnv(t, tt.env)

			cfg, err := config.Load()

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Broadcast != tt.wantBroadcast {
				t.Errorf("Broadcast = %v, want %v", cfg.Broadcast, tt.wantBroadcast)
			}
			if tt.wantGroqKey != "" && cfg.GroqAPIKey != tt.wantGroqKey {
				t.Errorf("GroqAPIKey = %q, want %q", cfg.GroqAPIKey, tt.wantGroqKey)
			}
		})
	}
}
