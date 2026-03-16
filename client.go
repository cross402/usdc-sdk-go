package pay

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	apiPathPrefix   = "/api"
	v2PathPrefix    = "/v2"
	defaultTimeout  = 30 * time.Second
	maxResponseSize = 10 << 20 // 10 MiB
)

// Client calls the payment API. When created with WithBearerAuth it uses the
// /v2 prefix; otherwise it uses the /api prefix (public mode, no auth required).
type Client struct {
	baseURL         string
	pathPrefix      string
	httpClient      *http.Client
	authFunc        func(*http.Request)
	hasCustomClient bool
	optErr          error
}

type request struct {
	method      string
	uri         string
	header      http.Header
	body        io.Reader
	result      any
	absoluteURI bool
}

// NewClient creates a Client for the given baseURL (the API root without path prefix).
// If an auth option is provided the client uses /v2; otherwise /api (public mode).
func NewClient(baseURL string, opts ...OptFn) (*Client, error) {
	if baseURL == "" {
		return nil, &ValidationError{Message: ErrEmptyBaseURL.Error(), Err: ErrEmptyBaseURL}
	}

	c := &Client{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		pathPrefix: apiPathPrefix,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.optErr != nil {
		return nil, c.optErr
	}

	return c, nil
}

func (c *Client) do(ctx context.Context, r *request) error {
	var fullURL string
	if r.absoluteURI {
		fullURL = fmt.Sprintf("%s%s", c.baseURL, r.uri)
	} else {
		fullURL = fmt.Sprintf("%s%s%s", c.baseURL, c.pathPrefix, r.uri)
	}

	header := http.Header{}
	if r.header != nil {
		header = r.header.Clone()
	}

	if r.body != nil {
		header.Set("Content-Type", "application/json")
	}

	if c.authFunc != nil {
		req := &http.Request{Header: header}
		c.authFunc(req)

		header = req.Header
	}

	req, err := http.NewRequestWithContext(ctx, r.method, fullURL, r.body)
	if err != nil {
		return &UnexpectedError{Err: err}
	}

	req.Header = header

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &UnexpectedError{Err: err}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))

	closeErr := resp.Body.Close()

	if err != nil {
		return &UnexpectedError{Err: err}
	}

	if closeErr != nil {
		return &UnexpectedError{Err: closeErr}
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return &RequestError{
			StatusCode: resp.StatusCode,
			Body:       string(body),
		}
	}

	if r.result == nil {
		return nil
	}

	err = json.Unmarshal(body, r.result)
	if err != nil {
		return &UnexpectedError{Err: err}
	}

	return nil
}
