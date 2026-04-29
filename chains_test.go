package pay

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetSupportedChains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		handler   http.HandlerFunc
		opts      []OptFn
		want      *SupportedChainsResponse
		wantErr   bool
		errTarget any
	}{
		{
			name: "success - public mode",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/api/chains" {
					http.Error(w, "bad route", http.StatusNotFound)
					return
				}

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(SupportedChainsResponse{
					Chains:       []string{"base", "solana", "skale-base", "megaeth"},
					TargetChains: []string{"base", "solana"},
				})
			},
			want: &SupportedChainsResponse{
				Chains:       []string{"base", "solana", "skale-base", "megaeth"},
				TargetChains: []string{"base", "solana"},
			},
		},
		{
			name: "success - auth-configured client still hits /api/chains",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/chains" {
					http.Error(w, "must use /api prefix even when authed", http.StatusNotFound)
					return
				}

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(SupportedChainsResponse{
					Chains:       []string{"base"},
					TargetChains: []string{"base"},
				})
			},
			opts: []OptFn{WithBearerAuth("id", "secret")},
			want: &SupportedChainsResponse{
				Chains:       []string{"base"},
				TargetChains: []string{"base"},
			},
		},
		{
			name: "error - API error surfaces RequestError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"message":"boom"}`))
			},
			wantErr:   true,
			errTarget: &RequestError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			c, err := NewClient(srv.URL, tt.opts...)
			if err != nil {
				t.Fatalf("NewClient() failed: %v", err)
			}

			got, err := c.GetSupportedChains(t.Context())

			if tt.wantErr {
				if err == nil {
					t.Fatal("GetSupportedChains() expected error, got nil")
				}

				if tt.errTarget != nil {
					var reqErr *RequestError
					if !errors.As(err, &reqErr) {
						t.Errorf("expected *RequestError, got %T: %v", err, err)
					}
				}

				return
			}

			if err != nil {
				t.Fatalf("GetSupportedChains() failed: %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("GetSupportedChains() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
