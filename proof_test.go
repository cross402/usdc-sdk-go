package pay

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSubmitProof(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		handler     http.HandlerFunc
		intentID    string
		settleProof string
		want        *SubmitProofResponse
		wantErr     bool
		errIs       error
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/intents/xyz-789" {
					http.Error(w, "bad path", http.StatusNotFound)
					return
				}

				var body struct {
					SettleProof string `json:"settle_proof"`
				}

				err := json.NewDecoder(r.Body).Decode(&body)
				if err != nil {
					http.Error(w, "bad body", http.StatusBadRequest)
					return
				}

				if body.SettleProof != "proof-base64-here" {
					http.Error(w, "bad proof", http.StatusBadRequest)
					return
				}

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(SubmitProofResponse{
					IntentBase: IntentBase{IntentID: "xyz-789", Status: StatusPending},
				})
			},
			intentID:    "xyz-789",
			settleProof: "proof-base64-here",
			want: &SubmitProofResponse{
				IntentBase: IntentBase{IntentID: "xyz-789", Status: StatusPending},
			},
		},
		{
			name:        "error - empty intent ID",
			intentID:    "",
			settleProof: "proof",
			wantErr:     true,
			errIs:       ErrEmptyIntentID,
		},
		{
			name:        "error - empty settle proof",
			intentID:    "intent-1",
			settleProof: "",
			wantErr:     true,
			errIs:       ErrEmptySettleProof,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var baseURL string

			if tt.handler != nil {
				srv := httptest.NewServer(tt.handler)
				defer srv.Close()

				baseURL = srv.URL
			} else {
				baseURL = "http://localhost"
			}

			c, err := NewClient(baseURL)
			if err != nil {
				t.Fatalf("NewClient() failed: %v", err)
			}

			got, err := c.SubmitProof(t.Context(), tt.intentID, tt.settleProof)

			if tt.wantErr {
				if err == nil {
					t.Fatal("SubmitProof() expected error, got nil")
				}

				if tt.errIs != nil && !errors.Is(err, tt.errIs) {
					t.Errorf("expected %v, got %v", tt.errIs, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("SubmitProof() failed: %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("SubmitProof() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSubmitProof_RejectsV2Client(t *testing.T) {
	t.Parallel()

	c, err := NewClient("http://localhost", WithBearerAuth("id", "secret"))
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	_, err = c.SubmitProof(t.Context(), "intent-1", "proof")
	if !errors.Is(err, ErrSubmitProofNotAllowed) {
		t.Errorf("expected ErrSubmitProofNotAllowed, got %v", err)
	}
}
