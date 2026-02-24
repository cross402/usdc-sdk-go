package pay

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	_v2PathPrefix   = "/v2"
	_defaultTimeout = 30 * time.Second
)

// Client calls the v2 payment API with API Key authentication.
type Client struct {
	baseURL    string
	httpClient *http.Client
	authHeader string
	useHeaders bool
	clientID   string
	apiKey     string
}

// NewClient builds a client using Bearer auth (Base64 of clientID:clientSecret).
// baseURL is the API root without /v2 (e.g. https://api-pay.agent.tech/api); requests use {baseURL}/v2/...
func NewClient(baseURL, clientID, clientSecret string) *Client {
	token := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	return &Client{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{Timeout: _defaultTimeout},
		authHeader: "Bearer " + token,
	}
}

// NewClientWithAPIKey builds a client using X-Client-ID and X-API-Key headers.
// baseURL is the API root without /v2 (e.g. https://api-pay.agent.tech/api); requests use {baseURL}/v2/...
func NewClientWithAPIKey(baseURL, clientID, apiKey string) *Client {
	return &Client{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{Timeout: _defaultTimeout},
		useHeaders: true,
		clientID:   clientID,
		apiKey:     apiKey,
	}
}

// SetHTTPClient replaces the default HTTP client (e.g. for custom timeouts or transport).
func (c *Client) SetHTTPClient(hc *http.Client) {
	if hc != nil {
		c.httpClient = hc
	}
}

func (c *Client) setAuth(req *http.Request) {
	if c.useHeaders {
		req.Header.Set("X-Client-ID", c.clientID)
		req.Header.Set("X-API-Key", c.apiKey)
	} else {
		req.Header.Set("Authorization", c.authHeader)
	}
}

func (c *Client) do(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	u := c.baseURL + _v2PathPrefix + path
	req, err := http.NewRequestWithContext(ctx, method, u, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
	}
	c.setAuth(req)
	return c.httpClient.Do(req)
}

func (c *Client) parseError(resp *http.Response) error {
	var er ErrorResponse
	_ = json.NewDecoder(resp.Body).Decode(&er)
	resp.Body.Close()
	msg := er.Message
	if msg == "" {
		msg = resp.Status
	}
	return &APIError{StatusCode: resp.StatusCode, Message: msg}
}

// CreateIntent creates a payment intent (POST /v2/intents).
// Exactly one of req.Email or req.Recipient must be set.
func (c *Client) CreateIntent(ctx context.Context, req *CreateIntentRequest) (*CreateIntentResponse, error) {
	if req == nil {
		return nil, &APIError{StatusCode: 0, Message: "CreateIntentRequest is nil"}
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	resp, err := c.do(ctx, http.MethodPost, "/intents", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}
	var out CreateIntentResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out, nil
}

// ExecuteIntent triggers transfer on Base using the Agent wallet (POST /v2/intents/{intent_id}/execute).
// No body or settle_proof required; backend signs and transfers USDC to the intent recipient.
func (c *Client) ExecuteIntent(ctx context.Context, intentID string) (*ExecuteResponse, error) {
	if intentID == "" {
		return nil, &APIError{StatusCode: 0, Message: "intent_id is required"}
	}
	resp, err := c.do(ctx, http.MethodPost, "/intents/"+url.PathEscape(intentID)+"/execute", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}
	var out ExecuteResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out, nil
}

// Intent returns intent status and receipt (GET /v2/intents?intent_id=...).
func (c *Client) Intent(ctx context.Context, intentID string) (*GetIntentResponse, error) {
	if intentID == "" {
		return nil, &APIError{StatusCode: 0, Message: "intent_id is required"}
	}
	path := "/intents?intent_id=" + url.QueryEscape(intentID)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}
	var out GetIntentResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out, nil
}
