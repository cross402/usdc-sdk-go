package pay

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCreateIntent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		handler   http.HandlerFunc
		opts      []OptFn
		req       *CreateIntentRequest
		want      *CreateIntentResponse
		wantErr   bool
		errTarget any // for errors.As
	}{
		{
			name: "success - with auth",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/v2/intents" {
					http.Error(w, "bad route", http.StatusNotFound)
					return
				}

				if r.Header.Get("Content-Type") != "application/json" {
					http.Error(w, "bad content-type", http.StatusBadRequest)
					return
				}

				if r.Header.Get("Authorization") == "" {
					http.Error(w, "missing auth", http.StatusUnauthorized)
					return
				}

				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(CreateIntentResponse{
					IntentBase: IntentBase{IntentID: "intent-1", Status: StatusAwaitingPayment},
				})
			},
			opts: []OptFn{WithBearerAuth("id", "secret")},
			req: &CreateIntentRequest{
				Email:      "test@example.com",
				Amount:     "10.00",
				PayerChain: "base",
			},
			want: &CreateIntentResponse{
				IntentBase: IntentBase{IntentID: "intent-1", Status: StatusAwaitingPayment},
			},
		},
		{
			name: "success - public mode",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/intents" {
					http.Error(w, "bad path", http.StatusNotFound)
					return
				}

				if r.Header.Get("Authorization") != "" {
					http.Error(w, "unexpected auth", http.StatusBadRequest)
					return
				}

				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(CreateIntentResponse{
					IntentBase:  IntentBase{IntentID: "intent-public-1", Status: StatusAwaitingPayment},
					PayerChain:  "solana",
					TargetChain: "base",
				})
			},
			req: &CreateIntentRequest{
				Email:      "test@example.com",
				Amount:     "10.00",
				PayerChain: "solana",
			},
			want: &CreateIntentResponse{
				IntentBase:  IntentBase{IntentID: "intent-public-1", Status: StatusAwaitingPayment},
				PayerChain:  "solana",
				TargetChain: "base",
			},
		},
		{
			name:    "success - explicit target chain serialized",
			handler: handleCreateIntentExplicitTarget,
			req: &CreateIntentRequest{
				Email:       "test@example.com",
				Amount:      "10.00",
				PayerChain:  "base",
				TargetChain: "solana",
			},
			want: &CreateIntentResponse{
				IntentBase:  IntentBase{IntentID: "intent-mc-1", Status: StatusAwaitingPayment},
				PayerChain:  "base",
				TargetChain: "solana",
			},
		},
		{
			name:    "success - empty target chain omitted from wire",
			handler: handleCreateIntentOmitTarget,
			req: &CreateIntentRequest{
				Email:      "test@example.com",
				Amount:     "10.00",
				PayerChain: "base",
			},
			want: &CreateIntentResponse{
				IntentBase:  IntentBase{IntentID: "intent-default-tc", Status: StatusAwaitingPayment},
				PayerChain:  "base",
				TargetChain: "base",
			},
		},
		{
			name:    "error - nil request",
			handler: func(_ http.ResponseWriter, _ *http.Request) {},
			opts:    []OptFn{WithBearerAuth("id", "secret")},
			req:     nil,
			wantErr: true,
		},
		{
			name: "error - API error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"message":"invalid amount"}`))
			},
			opts:      []OptFn{WithBearerAuth("id", "secret")},
			req:       &CreateIntentRequest{Amount: "0"},
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

			got, err := c.CreateIntent(t.Context(), tt.req)

			if tt.wantErr {
				if err == nil {
					t.Fatal("CreateIntent() expected error, got nil")
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
				t.Fatalf("CreateIntent() failed: %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("CreateIntent() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExecuteIntent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		handler  http.HandlerFunc
		opts     []OptFn
		intentID string
		want     *ExecuteIntentResponse
		wantErr  bool
		errIs    error // for errors.Is
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v2/intents/abc-123/execute" {
					http.Error(w, "bad path", http.StatusNotFound)
					return
				}

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(ExecuteIntentResponse{
					IntentBase: IntentBase{IntentID: "abc-123", Status: StatusTargetSettled},
				})
			},
			opts:     []OptFn{WithBearerAuth("id", "secret")},
			intentID: "abc-123",
			want: &ExecuteIntentResponse{
				IntentBase: IntentBase{IntentID: "abc-123", Status: StatusTargetSettled},
			},
		},
		{
			name:     "error - empty ID",
			opts:     []OptFn{WithBearerAuth("id", "secret")},
			intentID: "",
			wantErr:  true,
			errIs:    ErrEmptyIntentID,
		},
		{
			name:     "error - no auth",
			intentID: "some-id",
			wantErr:  true,
			errIs:    ErrMissingAuth,
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

			c, err := NewClient(baseURL, tt.opts...)
			if err != nil {
				t.Fatalf("NewClient() failed: %v", err)
			}

			got, err := c.ExecuteIntent(t.Context(), tt.intentID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("ExecuteIntent() expected error, got nil")
				}

				if tt.errIs != nil && !errors.Is(err, tt.errIs) {
					t.Errorf("expected %v, got %v", tt.errIs, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("ExecuteIntent() failed: %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ExecuteIntent() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetIntent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		handler  http.HandlerFunc
		opts     []OptFn
		intentID string
		want     *GetIntentResponse
		wantErr  bool
		errIs    error
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("intent_id") != "xyz" {
					http.Error(w, "bad query", http.StatusBadRequest)
					return
				}

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(GetIntentResponse{
					IntentBase: IntentBase{IntentID: "xyz", Status: StatusTargetSettled},
				})
			},
			opts:     []OptFn{WithBearerAuth("id", "secret")},
			intentID: "xyz",
			want: &GetIntentResponse{
				IntentBase: IntentBase{IntentID: "xyz", Status: StatusTargetSettled},
			},
		},
		{
			name: "success - multichain settlement decodes target_chain and target_payment",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"intent_id": "mc-1",
					"status": "TARGET_SETTLED",
					"payer_chain": "base",
					"target_chain": "solana",
					"merchant_recipient": "merchant@example.com",
					"target_payment": {
						"tx_hash": "tx-hash-123",
						"settle_proof": "proof-456",
						"settled_at": "2026-04-30T12:00:00Z",
						"explorer_url": "https://solscan.io/tx/tx-hash-123"
					}
				}`))
			},
			opts:     []OptFn{WithBearerAuth("id", "secret")},
			intentID: "mc-1",
			want: &GetIntentResponse{
				IntentBase: IntentBase{
					IntentID:          "mc-1",
					MerchantRecipient: "merchant@example.com",
					Status:            StatusTargetSettled,
				},
				PayerChain:  "base",
				TargetChain: "solana",
				TargetPayment: &TargetPayment{
					TxHash:      "tx-hash-123",
					SettleProof: "proof-456",
					SettledAt:   "2026-04-30T12:00:00Z",
					ExplorerURL: "https://solscan.io/tx/tx-hash-123",
				},
			},
		},
		{
			name:     "error - empty ID",
			opts:     []OptFn{WithBearerAuth("id", "secret")},
			intentID: "",
			wantErr:  true,
			errIs:    ErrEmptyIntentID,
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

			c, err := NewClient(baseURL, tt.opts...)
			if err != nil {
				t.Fatalf("NewClient() failed: %v", err)
			}

			got, err := c.GetIntent(t.Context(), tt.intentID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("GetIntent() expected error, got nil")
				}

				if tt.errIs != nil && !errors.Is(err, tt.errIs) {
					t.Errorf("expected %v, got %v", tt.errIs, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("GetIntent() failed: %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("GetIntent() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func handleCreateIntentExplicitTarget(w http.ResponseWriter, r *http.Request) {
	var body map[string]any

	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}

	if body["target_chain"] != "solana" {
		http.Error(w, "missing or wrong target_chain", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(CreateIntentResponse{
		IntentBase:  IntentBase{IntentID: "intent-mc-1", Status: StatusAwaitingPayment},
		PayerChain:  "base",
		TargetChain: "solana",
	})
}

func handleCreateIntentOmitTarget(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}

	if bytes.Contains(raw, []byte("target_chain")) {
		http.Error(w, "target_chain should be omitted when empty", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(CreateIntentResponse{
		IntentBase:  IntentBase{IntentID: "intent-default-tc", Status: StatusAwaitingPayment},
		PayerChain:  "base",
		TargetChain: "base",
	})
}

func TestListIntents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		handler  http.HandlerFunc
		opts     []OptFn
		page     int
		pageSize int
		want     *ListIntentsResponse
		wantErr  bool
		errIs    error
	}{
		{
			name: "success - explicit pagination",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/v2/intents/list" {
					http.Error(w, "bad route", http.StatusNotFound)
					return
				}

				if r.URL.Query().Get("page") != "2" || r.URL.Query().Get("page_size") != "5" {
					http.Error(w, "bad query", http.StatusBadRequest)
					return
				}

				if r.Header.Get("Authorization") == "" {
					http.Error(w, "missing auth", http.StatusUnauthorized)
					return
				}

				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(ListIntentsResponse{
					Intents: []IntentSummary{
						{
							IntentBase:  IntentBase{IntentID: "i-1", Status: StatusTargetSettled, AgentID: "agent-1"},
							PayerChain:  "base",
							TargetChain: "solana",
						},
					},
					Total:    1,
					Page:     2,
					PageSize: 5,
				})
			},
			opts:     []OptFn{WithBearerAuth("id", "secret")},
			page:     2,
			pageSize: 5,
			want: &ListIntentsResponse{
				Intents: []IntentSummary{
					{
						IntentBase:  IntentBase{IntentID: "i-1", Status: StatusTargetSettled, AgentID: "agent-1"},
						PayerChain:  "base",
						TargetChain: "solana",
					},
				},
				Total:    1,
				Page:     2,
				PageSize: 5,
			},
		},
		{
			name: "success - server defaults when zero",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.RawQuery != "" {
					http.Error(w, "expected no query string", http.StatusBadRequest)
					return
				}

				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(ListIntentsResponse{Intents: []IntentSummary{}, Page: 1, PageSize: 20})
			},
			opts: []OptFn{WithBearerAuth("id", "secret")},
			want: &ListIntentsResponse{Intents: []IntentSummary{}, Page: 1, PageSize: 20},
		},
		{
			name:    "error - no auth",
			wantErr: true,
			errIs:   ErrMissingAuth,
		},
		{
			name:     "error - invalid pagination",
			opts:     []OptFn{WithBearerAuth("id", "secret")},
			pageSize: 200,
			wantErr:  true,
			errIs:    ErrInvalidPagination,
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

			got, err := c.ListIntents(t.Context(), tt.page, tt.pageSize)

			if tt.wantErr {
				if err == nil {
					t.Fatal("ListIntents() expected error, got nil")
				}

				if tt.errIs != nil && !errors.Is(err, tt.errIs) {
					t.Errorf("expected %v, got %v", tt.errIs, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("ListIntents() failed: %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ListIntents() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
