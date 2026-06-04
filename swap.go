package pay

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

// SwapJobStatus is the status of a registered swap intent job.
type SwapJobStatus string

const (
	SwapJobStatusPending  SwapJobStatus = "PENDING"
	SwapJobStatusDone     SwapJobStatus = "DONE"
	SwapJobStatusFailed   SwapJobStatus = "FAILED"
	SwapJobStatusCanceled SwapJobStatus = "CANCELED"
)

// SwapQuoteParams contains parameters for GET /api/swap/quote.
//
// Exactly one of FromAmount or ToAmount must be non-zero.
// FromAmount sets ExactIn mode; ToAmount sets ExactOut mode.
type SwapQuoteParams struct {
	Chain         string // required: source chain
	InputToken    string // required: token contract/mint to swap from
	OutputToken   string // required: token contract/mint to swap to
	FromAmount    uint64 // ExactIn: source-side amount in smallest unit
	ToAmount      uint64 // ExactOut: target-side amount in smallest unit
	SlippageBps   uint16 // optional: basis points, default 50 (0.5%)
	ToChain       string // optional: destination chain; empty = same-chain
	UserAddress   string // optional: signer address; if set, swap tx is included
	ToUserAddress string // optional: destination recipient; required for cross-family routes with UserAddress
}

// SwapQuoteData holds the quote details returned by the API.
type SwapQuoteData struct {
	InputToken      string `json:"input_token"`
	OutputToken     string `json:"output_token"`
	InputAmount     string `json:"input_amount"`
	OutputAmount    string `json:"output_amount"`
	MinOutputAmount string `json:"min_output_amount"`
	PriceImpactPct  string `json:"price_impact_pct"`
}

// SwapTransaction holds the transaction payload for the user to sign.
type SwapTransaction struct {
	Transaction          string `json:"transaction"`
	To                   string `json:"to,omitempty"`
	Value                string `json:"value,omitempty"`
	GasLimit             string `json:"gas_limit,omitempty"`
	ExpiresAt            int64  `json:"expires_at"`
	LastValidBlockHeight uint64 `json:"last_valid_block_height,omitempty"`
}

// SwapQuoteResponse is the response for GET /api/swap/quote.
type SwapQuoteResponse struct {
	Quote           SwapQuoteData    `json:"quote"`
	SwapTransaction *SwapTransaction `json:"swap_transaction,omitempty"`
}

// RegisterSwapIntentRequest is the body for POST /api/swap/intents.
type RegisterSwapIntentRequest struct {
	SourceTxHash       string `json:"source_tx_hash"`
	FromChain          string `json:"from_chain"`
	ToChain            string `json:"to_chain"`
	FromToken          string `json:"from_token"`
	ToToken            string `json:"to_token"`
	PayerAddress       string `json:"payer_address"`
	RecipientAddress   string `json:"recipient_address"`
	SendingTokenAmount string `json:"sending_token_amount"`
}

// RegisterSwapIntentResponse is the response for POST /api/swap/intents (201).
type RegisterSwapIntentResponse struct {
	IntentID string        `json:"intent_id"`
	Status   SwapJobStatus `json:"status"`
}

// GetSwapQuote fetches a quote for swapping tokens (GET /api/swap/quote).
// Exactly one of params.FromAmount or params.ToAmount must be non-zero.
func (c *Client) GetSwapQuote(ctx context.Context, params *SwapQuoteParams) (*SwapQuoteResponse, error) {
	if params == nil {
		return nil, &ValidationError{Message: ErrNilParams.Error(), Err: ErrNilParams}
	}

	if params.FromAmount != 0 && params.ToAmount != 0 {
		return nil, &ValidationError{
			Message: ErrExactInExactOutMutuallyExclusive.Error(),
			Err:     ErrExactInExactOutMutuallyExclusive,
		}
	}

	if params.FromAmount == 0 && params.ToAmount == 0 {
		return nil, &ValidationError{
			Message: ErrExactInExactOutMutuallyExclusive.Error(),
			Err:     ErrExactInExactOutMutuallyExclusive,
		}
	}

	q := url.Values{}
	q.Set("chain", params.Chain)
	q.Set("input_token", params.InputToken)
	q.Set("output_token", params.OutputToken)

	if params.FromAmount != 0 {
		q.Set("from_amount", strconv.FormatUint(params.FromAmount, 10))
	} else {
		q.Set("to_amount", strconv.FormatUint(params.ToAmount, 10))
	}

	if params.SlippageBps != 0 {
		q.Set("slippage_bps", strconv.FormatUint(uint64(params.SlippageBps), 10))
	}

	if params.ToChain != "" {
		q.Set("to_chain", params.ToChain)
	}

	if params.UserAddress != "" {
		q.Set("user_address", params.UserAddress)
	}

	if params.ToUserAddress != "" {
		q.Set("to_user_address", params.ToUserAddress)
	}

	var out SwapQuoteResponse

	err := c.do(ctx, &request{
		method:      http.MethodGet,
		uri:         "/api/swap/quote?" + q.Encode(),
		result:      &out,
		absoluteURI: true,
	})
	if err != nil {
		return nil, err
	}

	return &out, nil
}

// RegisterSwapIntent records a submitted swap transaction as a payment intent
// (POST /api/swap/intents). The returned intent_id can be used to track settlement.
func (c *Client) RegisterSwapIntent(ctx context.Context, req *RegisterSwapIntentRequest) (*RegisterSwapIntentResponse, error) {
	if req == nil {
		return nil, &ValidationError{Message: ErrNilParams.Error(), Err: ErrNilParams}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, &UnexpectedError{Err: err}
	}

	var out RegisterSwapIntentResponse

	err = c.do(ctx, &request{
		method:      http.MethodPost,
		uri:         "/api/swap/intents",
		body:        bytes.NewReader(body),
		result:      &out,
		absoluteURI: true,
	})
	if err != nil {
		return nil, err
	}

	return &out, nil
}

// GetSwapTokens returns the raw LiFi /v1/tokens JSON (GET /api/swap/tokens).
// chains and chainTypes are optional filter parameters passed through to LiFi.
// The response is returned as raw JSON to avoid coupling to LiFi's schema.
func (c *Client) GetSwapTokens(ctx context.Context, chains, chainTypes string) (json.RawMessage, error) {
	q := url.Values{}
	if chains != "" {
		q.Set("chains", chains)
	}

	if chainTypes != "" {
		q.Set("chainTypes", chainTypes)
	}

	uri := "/api/swap/tokens"
	if encoded := q.Encode(); encoded != "" {
		uri += "?" + encoded
	}

	var out json.RawMessage

	err := c.do(ctx, &request{
		method:      http.MethodGet,
		uri:         uri,
		result:      &out,
		absoluteURI: true,
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

// GetSwapChains returns the raw LiFi /v1/chains JSON (GET /api/swap/chains).
// chainTypes is an optional filter passed through to LiFi.
func (c *Client) GetSwapChains(ctx context.Context, chainTypes string) (json.RawMessage, error) {
	uri := "/api/swap/chains"
	if chainTypes != "" {
		uri += "?chainTypes=" + url.QueryEscape(chainTypes)
	}

	var out json.RawMessage

	err := c.do(ctx, &request{
		method:      http.MethodGet,
		uri:         uri,
		result:      &out,
		absoluteURI: true,
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

// GetSwapConnections returns the raw LiFi /v1/connections JSON (GET /api/swap/connections).
// All parameters are optional filters passed through to LiFi.
func (c *Client) GetSwapConnections(ctx context.Context, fromChain, toChain, fromToken, toToken string) (json.RawMessage, error) {
	q := url.Values{}
	if fromChain != "" {
		q.Set("fromChain", fromChain)
	}

	if toChain != "" {
		q.Set("toChain", toChain)
	}

	if fromToken != "" {
		q.Set("fromToken", fromToken)
	}

	if toToken != "" {
		q.Set("toToken", toToken)
	}

	uri := "/api/swap/connections"
	if encoded := q.Encode(); encoded != "" {
		uri += "?" + encoded
	}

	var out json.RawMessage

	err := c.do(ctx, &request{
		method:      http.MethodGet,
		uri:         uri,
		result:      &out,
		absoluteURI: true,
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

// ExecuteSwapRequest is the body for POST /v2/swap/execute.
// Requires Bearer auth (WithBearerAuth). The backend executes the swap on behalf
// of the authenticated user without requiring the caller to hold a private key.
type ExecuteSwapRequest struct {
	Chain       string `json:"chain"`
	FromToken   string `json:"from_token"`
	ToToken     string `json:"to_token"`
	FromAmount  uint64 `json:"from_amount"`
	SlippageBps uint16 `json:"slippage_bps,omitempty"`
	ToChain     string `json:"to_chain,omitempty"`
}

// ExecuteSwapResponse is the response for POST /v2/swap/execute (200).
type ExecuteSwapResponse struct {
	TxHash          string `json:"tx_hash"`
	Chain           string `json:"chain"`
	FromToken       string `json:"from_token"`
	ToToken         string `json:"to_token"`
	FromAmount      string `json:"from_amount"`
	EstimatedOutput string `json:"estimated_output"`
}

// ExecuteSwap submits a swap on behalf of the authenticated user without
// requiring a private key (POST /v2/swap/execute). Requires WithBearerAuth.
func (c *Client) ExecuteSwap(ctx context.Context, req *ExecuteSwapRequest) (*ExecuteSwapResponse, error) {
	if c.authFunc == nil {
		return nil, &ValidationError{Message: ErrMissingAuth.Error(), Err: ErrMissingAuth}
	}

	if req == nil {
		return nil, &ValidationError{Message: ErrNilParams.Error(), Err: ErrNilParams}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, &UnexpectedError{Err: err}
	}

	var out ExecuteSwapResponse

	err = c.do(ctx, &request{
		method: http.MethodPost,
		uri:    "/swap/execute",
		body:   bytes.NewReader(body),
		result: &out,
	})
	if err != nil {
		return nil, err
	}

	return &out, nil
}
