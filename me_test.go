package pay

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetMe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		handler http.HandlerFunc
		opts    []OptFn
		want    *Me
		wantErr bool
		errIs   error
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/v2/me" {
					http.Error(w, "bad route", http.StatusNotFound)
					return
				}

				if r.Header.Get("Authorization") == "" {
					http.Error(w, "missing auth", http.StatusUnauthorized)
					return
				}

				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(Me{
					AgentID:     "agent-1",
					AgentNumber: "A-001",
					Name:        "demo",
					Status:      "ACTIVE",
				})
			},
			opts: []OptFn{WithBearerAuth("id", "secret")},
			want: &Me{AgentID: "agent-1", AgentNumber: "A-001", Name: "demo", Status: "ACTIVE"},
		},
		{
			name:    "error - no auth",
			opts:    nil,
			wantErr: true,
			errIs:   ErrMissingAuth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			baseURL := "http://localhost"

			if tt.handler != nil {
				srv := httptest.NewServer(tt.handler)
				defer srv.Close()

				baseURL = srv.URL
			}

			c, err := NewClient(baseURL, tt.opts...)
			if err != nil {
				t.Fatalf("NewClient() failed: %v", err)
			}

			got, err := c.GetMe(t.Context())

			if tt.wantErr {
				if err == nil {
					t.Fatal("GetMe() expected error, got nil")
				}

				if tt.errIs != nil && !errors.Is(err, tt.errIs) {
					t.Errorf("expected %v, got %v", tt.errIs, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("GetMe() failed: %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("GetMe() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
