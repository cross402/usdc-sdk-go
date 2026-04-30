package pay

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

// Intent status constants returned by the API.
const (
	StatusAwaitingPayment    = "AWAITING_PAYMENT"
	StatusPending            = "PENDING"
	StatusVerificationFailed = "VERIFICATION_FAILED"
	StatusSourceSettled      = "SOURCE_SETTLED"
	StatusTargetSettling     = "TARGET_SETTLING"
	StatusTargetSettled      = "TARGET_SETTLED"
	StatusExpired            = "EXPIRED"
	StatusPartialSettlement  = "PARTIAL_SETTLEMENT"
)

// CreateIntentRequest is the body for POST /intents.
// Exactly one of Email or Recipient must be set.
type CreateIntentRequest struct {
	Email      string `json:"email,omitempty"`
	Recipient  string `json:"recipient,omitempty"`
	Amount     string `json:"amount"`
	PayerChain string `json:"payer_chain"`
	// TargetChain is the settlement destination. Optional; defaults to "base"
	// when empty. See GetSupportedChains for valid values.
	TargetChain string `json:"target_chain,omitempty"`
}

// FeeBreakdown holds fee details from the API.
type FeeBreakdown struct {
	SourceChain           string `json:"source_chain"`
	SourceChainFee        string `json:"source_chain_fee"`
	TargetChain           string `json:"target_chain"`
	TargetChainFee        string `json:"target_chain_fee"`
	PlatformFee           string `json:"platform_fee"`
	PlatformFeePercentage string `json:"platform_fee_percentage"`
	TotalFee              string `json:"total_fee"`
}

// PaymentRequirements is used by the client to sign X402 authorization.
type PaymentRequirements struct {
	Scheme            string         `json:"scheme"`
	Network           string         `json:"network"`
	Amount            string         `json:"amount"`
	PayTo             string         `json:"payTo"`
	MaxTimeoutSeconds int            `json:"maxTimeoutSeconds"`
	Asset             string         `json:"asset"`
	Extra             map[string]any `json:"extra,omitempty"`
}

// IntentBase contains the fields shared across all intent response types.
type IntentBase struct {
	IntentID          string        `json:"intent_id"`
	MerchantRecipient string        `json:"merchant_recipient"`
	SendingAmount     string        `json:"sending_amount"`
	ReceivingAmount   string        `json:"receiving_amount"`
	EstimatedFee      string        `json:"estimated_fee"`
	FeeBreakdown      *FeeBreakdown `json:"fee_breakdown"`
	Status            string        `json:"status"`
	CreatedAt         string        `json:"created_at"`
	ExpiresAt         string        `json:"expires_at"`
	// AgentID is the owning agent's UUID. Populated only on v2 intents
	// (created via WithBearerAuth); empty for /api intents.
	AgentID string `json:"agent_id,omitempty"`
}

// CreateIntentResponse is the response for POST /intents (201).
type CreateIntentResponse struct {
	IntentBase

	Email               string               `json:"email,omitempty"`
	SourceRecipient     string               `json:"source_recipient,omitempty"`
	PayerChain          string               `json:"payer_chain"`
	TargetChain         string               `json:"target_chain"`
	PaymentRequirements *PaymentRequirements `json:"payment_requirements"`
}

// ExecuteIntentResponse is the response for POST /intents/{intent_id}/execute (200).
type ExecuteIntentResponse struct {
	IntentBase
}

// SourcePayment holds source-chain payment details from GetIntent.
type SourcePayment struct {
	Chain       string `json:"chain"`
	TxHash      string `json:"tx_hash"`
	SettleProof string `json:"settle_proof"`
	SettledAt   string `json:"settled_at"`
	ExplorerURL string `json:"explorer_url"`
}

// TargetPayment holds target-chain settlement details from GetIntent.
type TargetPayment struct {
	TxHash      string `json:"tx_hash"`
	SettleProof string `json:"settle_proof"`
	SettledAt   string `json:"settled_at"`
	ExplorerURL string `json:"explorer_url"`
}

// GetIntentResponse is the response for GET /intents?intent_id=... (200).
type GetIntentResponse struct {
	IntentBase

	PayerChain    string         `json:"payer_chain"`
	TargetChain   string         `json:"target_chain"`
	ReceiverEmail string         `json:"receiver_email,omitempty"`
	PayerWallet   string         `json:"payer_wallet,omitempty"`
	ErrorMessage  string         `json:"error_message,omitempty"`
	CompletedAt   string         `json:"completed_at,omitempty"`
	SourcePayment *SourcePayment `json:"source_payment,omitempty"`
	TargetPayment *TargetPayment `json:"target_payment,omitempty"`
}

// CreateIntent creates a payment intent (POST {prefix}/intents).
// Exactly one of req.Email or req.Recipient must be set.
func (c *Client) CreateIntent(ctx context.Context, req *CreateIntentRequest) (*CreateIntentResponse, error) {
	if req == nil {
		return nil, &ValidationError{Message: ErrNilParams.Error(), Err: ErrNilParams}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, &UnexpectedError{Err: err}
	}

	var out CreateIntentResponse

	err = c.do(ctx, &request{
		method: http.MethodPost,
		uri:    "/intents",
		body:   bytes.NewReader(body),
		result: &out,
	})
	if err != nil {
		return nil, err
	}

	return &out, nil
}

// ExecuteIntent triggers transfer on Base using the Agent wallet
// (POST {prefix}/intents/{intent_id}/execute). Requires auth.
func (c *Client) ExecuteIntent(ctx context.Context, intentID string) (*ExecuteIntentResponse, error) {
	if intentID == "" {
		return nil, &ValidationError{Message: ErrEmptyIntentID.Error(), Err: ErrEmptyIntentID}
	}

	if c.authFunc == nil {
		return nil, &ValidationError{Message: ErrMissingAuth.Error(), Err: ErrMissingAuth}
	}

	var out ExecuteIntentResponse

	err := c.do(ctx, &request{
		method: http.MethodPost,
		uri:    "/intents/" + url.PathEscape(intentID) + "/execute",
		result: &out,
	})
	if err != nil {
		return nil, err
	}

	return &out, nil
}

// IntentSummary is a list-row entry returned by ListIntents.
type IntentSummary struct {
	IntentBase

	PayerChain  string `json:"payer_chain"`
	TargetChain string `json:"target_chain"`
}

// ListIntentsResponse is the response for GET /v2/intents/list.
type ListIntentsResponse struct {
	Intents  []IntentSummary `json:"intents"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}

// ListIntents lists intents owned by the authenticated agent
// (GET /v2/intents/list?page=&page_size=). Requires auth.
//
// page is 1-indexed; pageSize must be in [1,100]. Pass page=0 / pageSize=0
// to use the server defaults (page 1, 20 per page).
func (c *Client) ListIntents(ctx context.Context, page, pageSize int) (*ListIntentsResponse, error) {
	if c.authFunc == nil {
		return nil, &ValidationError{Message: ErrMissingAuth.Error(), Err: ErrMissingAuth}
	}

	if page < 0 || pageSize < 0 || pageSize > 100 {
		return nil, &ValidationError{Message: ErrInvalidPagination.Error(), Err: ErrInvalidPagination}
	}

	q := url.Values{}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}

	if pageSize > 0 {
		q.Set("page_size", strconv.Itoa(pageSize))
	}

	uri := "/intents/list"
	if encoded := q.Encode(); encoded != "" {
		uri += "?" + encoded
	}

	var out ListIntentsResponse

	err := c.do(ctx, &request{
		method: http.MethodGet,
		uri:    uri,
		result: &out,
	})
	if err != nil {
		return nil, err
	}

	return &out, nil
}

// GetIntent returns intent status and receipt (GET {prefix}/intents?intent_id=...).
func (c *Client) GetIntent(ctx context.Context, intentID string) (*GetIntentResponse, error) {
	if intentID == "" {
		return nil, &ValidationError{Message: ErrEmptyIntentID.Error(), Err: ErrEmptyIntentID}
	}

	var out GetIntentResponse

	err := c.do(ctx, &request{
		method: http.MethodGet,
		uri:    "/intents?intent_id=" + url.QueryEscape(intentID),
		result: &out,
	})
	if err != nil {
		return nil, err
	}

	return &out, nil
}
