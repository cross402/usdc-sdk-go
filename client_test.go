package pay

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

// --- NewClient ---

func TestNewClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		baseURL    string
		opts       []OptFn
		wantPrefix string
		wantErr    bool
	}{
		{
			name:       "success - with bearer auth uses v2 prefix",
			baseURL:    "http://localhost",
			opts:       []OptFn{WithBearerAuth("id", "secret")},
			wantPrefix: v2PathPrefix,
		},
		{
			name:       "success - without auth uses api prefix",
			baseURL:    "http://localhost",
			wantPrefix: apiPathPrefix,
		},
		{
			name:       "success - trailing slash trimmed",
			baseURL:    "http://localhost/",
			opts:       []OptFn{WithBearerAuth("id", "secret")},
			wantPrefix: v2PathPrefix,
		},
		{
			name:    "error - empty baseURL",
			baseURL: "",
			wantErr: true,
		},
		{
			name:    "error - empty credentials",
			baseURL: "http://localhost",
			opts:    []OptFn{WithBearerAuth("", "secret")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := NewClient(tt.baseURL, tt.opts...)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("NewClient() failed: %v", err)
				}

				return
			}

			if tt.wantErr {
				t.Error("NewClient() expected error, got nil")
				return
			}

			if c.pathPrefix != tt.wantPrefix {
				t.Errorf("pathPrefix = %q, want %q", c.pathPrefix, tt.wantPrefix)
			}

			if tt.baseURL == "http://localhost/" && c.baseURL != "http://localhost" {
				t.Errorf("baseURL = %q, want trailing slash trimmed", c.baseURL)
			}
		})
	}
}

// --- Options ---

func TestClientOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		opts  []OptFn
		check func(t *testing.T, c *Client)
	}{
		{
			name: "success - timeout",
			opts: []OptFn{WithTimeout(5 * time.Second)},
			check: func(t *testing.T, c *Client) {
				t.Helper()

				if c.httpClient.Timeout != 5*time.Second {
					t.Errorf("timeout = %v, want 5s", c.httpClient.Timeout)
				}
			},
		},
		{
			name: "success - custom client ignores timeout",
			opts: []OptFn{
				WithHTTPClient(&http.Client{Timeout: 90 * time.Second}),
				WithTimeout(5 * time.Second),
			},
			check: func(t *testing.T, c *Client) {
				t.Helper()

				if c.httpClient.Timeout != 90*time.Second {
					t.Errorf("timeout = %v, want 90s", c.httpClient.Timeout)
				}
			},
		},
		{
			name: "success - nil http client ignored",
			opts: []OptFn{WithHTTPClient(nil)},
			check: func(t *testing.T, c *Client) {
				t.Helper()

				if c.httpClient == nil {
					t.Error("httpClient should not be nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := NewClient("http://localhost", tt.opts...)
			if err != nil {
				t.Fatalf("NewClient() failed: %v", err)
			}

			tt.check(t, c)
		})
	}
}

// --- Error types ---

func TestErrorTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		check func(t *testing.T)
	}{
		{
			name: "RequestError string",
			check: func(t *testing.T) {
				t.Helper()

				e := &RequestError{StatusCode: 400, Body: "bad request"}

				want := "request failed with status 400: bad request"
				if e.Error() != want {
					t.Errorf("Error() = %q, want %q", e.Error(), want)
				}
			},
		},
		{
			name: "ValidationError string",
			check: func(t *testing.T) {
				t.Helper()

				e := &ValidationError{Message: "field required"}

				want := "validation: field required"
				if e.Error() != want {
					t.Errorf("Error() = %q, want %q", e.Error(), want)
				}
			},
		},
		{
			name: "UnexpectedError unwrap",
			check: func(t *testing.T) {
				t.Helper()

				inner := errors.New("inner")

				e := &UnexpectedError{Err: inner}
				if !errors.Is(e, inner) {
					t.Error("errors.Is should match inner error")
				}
			},
		},
		{
			name: "ValidationError unwrap sentinel",
			check: func(t *testing.T) {
				t.Helper()

				e := &ValidationError{Message: "intent_id is required", Err: ErrEmptyIntentID}
				if !errors.Is(e, ErrEmptyIntentID) {
					t.Error("errors.Is should match ErrEmptyIntentID")
				}
			},
		},
		{
			name: "errors.As RequestError",
			check: func(t *testing.T) {
				t.Helper()

				var target *RequestError

				err := error(&RequestError{StatusCode: 404, Body: "not found"})
				if !errors.As(err, &target) {
					t.Error("errors.As should match *RequestError")
				}
			},
		},
		{
			name: "errors.As ValidationError",
			check: func(t *testing.T) {
				t.Helper()

				var target *ValidationError

				err := error(&ValidationError{Message: "bad"})
				if !errors.As(err, &target) {
					t.Error("errors.As should match *ValidationError")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.check(t)
		})
	}
}

// --- Auth & error parsing ---

func TestAuthAndErrorParsing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		handler http.HandlerFunc
		opts    []OptFn
		check   func(t *testing.T, err error)
	}{
		{
			name: "non-JSON error body preserved in RequestError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("not json"))
			},
			opts: []OptFn{WithBearerAuth("id", "secret")},
			check: func(t *testing.T, err error) {
				t.Helper()

				var reqErr *RequestError
				if !errors.As(err, &reqErr) {
					t.Fatalf("expected *RequestError, got %v", err)
				}

				if reqErr.StatusCode != http.StatusInternalServerError {
					t.Errorf("StatusCode = %d, want 500", reqErr.StatusCode)
				}

				if reqErr.Body != "not json" {
					t.Errorf("Body = %q, want %q", reqErr.Body, "not json")
				}
			},
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

			_, err = c.GetIntent(t.Context(), "test-id")
			tt.check(t, err)
		})
	}
}
