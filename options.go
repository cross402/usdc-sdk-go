package pay

import (
	"encoding/base64"
	"net/http"
	"time"
)

// OptFn configures a Client. Pass one or more options to NewClient.
type OptFn func(*Client)

// WithBearerAuth sets Bearer token authentication using Base64-encoded
// apiKey:secretKey.
func WithBearerAuth(apiKey, secretKey string) OptFn {
	return func(c *Client) {
		if apiKey == "" || secretKey == "" {
			c.optErr = &ValidationError{Message: ErrMissingAuth.Error(), Err: ErrMissingAuth}
			return
		}

		token := base64.StdEncoding.EncodeToString([]byte(apiKey + ":" + secretKey))
		c.authFunc = func(req *http.Request) {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		c.pathPrefix = v2PathPrefix
	}
}

// WithHTTPClient replaces the default HTTP client.
// When set, WithTimeout has no effect regardless of option ordering.
func WithHTTPClient(hc *http.Client) OptFn {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
			c.hasCustomClient = true
		}
	}
}

// WithTimeout sets the timeout on the default HTTP client.
// Ignored if WithHTTPClient is also provided.
func WithTimeout(d time.Duration) OptFn {
	return func(c *Client) {
		if !c.hasCustomClient {
			c.httpClient.Timeout = d
		}
	}
}
