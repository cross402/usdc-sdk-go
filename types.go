package pay

// CreateIntentRequest is the body for POST /v2/intents.
// Exactly one of Email or Recipient must be set.
type CreateIntentRequest struct {
	Email      string `json:"email,omitempty"`
	Recipient  string `json:"recipient,omitempty"`
	Amount     string `json:"amount"`
	PayerChain string `json:"payer_chain"`
}

// FeeBreakdown holds fee details from the API.
type FeeBreakdown struct {
	SourceChain         string `json:"source_chain"`
	SourceChainFee      string `json:"source_chain_fee"`
	TargetChain         string `json:"target_chain"`
	TargetChainFee      string `json:"target_chain_fee"`
	PlatformFee         string `json:"platform_fee"`
	PlatformFeePercentage string `json:"platform_fee_percentage"`
	TotalFee            string `json:"total_fee"`
}

// PaymentRequirements is used by the client to sign X402 authorization.
type PaymentRequirements struct {
	Scheme            string            `json:"scheme"`
	Network           string            `json:"network"`
	Amount            string            `json:"amount"`
	PayTo             string            `json:"payTo"`
	MaxTimeoutSeconds int               `json:"maxTimeoutSeconds"`
	Asset             string            `json:"asset"`
	Extra             map[string]string `json:"extra,omitempty"`
}

// CreateIntentResponse is the response for POST /v2/intents (201).
type CreateIntentResponse struct {
	IntentID             string               `json:"intent_id"`
	Email                string               `json:"email,omitempty"`
	MerchantRecipient    string               `json:"merchant_recipient"`
	SourceRecipient      string               `json:"source_recipient,omitempty"`
	SendingAmount   string              `json:"sending_amount"`
	ReceivingAmount string              `json:"receiving_amount"`
	EstimatedFee    string              `json:"estimated_fee"`
	FeeBreakdown         FeeBreakdown         `json:"fee_breakdown"`
	PayerChain           string               `json:"payer_chain"`
	Status               string               `json:"status"`
	CreatedAt            string               `json:"created_at"`
	ExpiresAt            string               `json:"expires_at"`
	PaymentRequirements PaymentRequirements `json:"payment_requirements"`
}

// SubmitProofRequest is the body for POST /v2/intents/{intent_id}.
type SubmitProofRequest struct {
	SettleProof string `json:"settle_proof"`
}

// SubmitProofResponse is the response for POST /v2/intents/{intent_id} (200).
type SubmitProofResponse struct {
	IntentID          string       `json:"intent_id"`
	MerchantRecipient string       `json:"merchant_recipient"`
	SendingAmount     string       `json:"sending_amount"`
	ReceivingAmount   string       `json:"receiving_amount"`
	EstimatedFee      string       `json:"estimated_fee"`
	FeeBreakdown      FeeBreakdown `json:"fee_breakdown"`
	Status            string       `json:"status"`
	CreatedAt         string       `json:"created_at"`
	ExpiresAt         string       `json:"expires_at"`
}

// SourcePayment holds source-chain payment details from GetIntent.
type SourcePayment struct {
	Chain       string `json:"chain"`
	TxHash      string `json:"tx_hash"`
	SettleProof string `json:"settle_proof"`
	SettledAt   string `json:"settled_at"`
	ExplorerURL string `json:"explorer_url"`
}

// BasePayment holds Base-chain payment details from GetIntent.
type BasePayment struct {
	TxHash      string `json:"tx_hash"`
	SettleProof string `json:"settle_proof"`
	SettledAt   string `json:"settled_at"`
	ExplorerURL string `json:"explorer_url"`
}

// GetIntentResponse is the response for GET /v2/intents?intent_id=... (200).
type GetIntentResponse struct {
	IntentID          string        `json:"intent_id"`
	Status            string        `json:"status"`
	SendingAmount     string        `json:"sending_amount"`
	ReceivingAmount   string        `json:"receiving_amount"`
	EstimatedFee      string        `json:"estimated_fee"`
	FeeBreakdown      FeeBreakdown  `json:"fee_breakdown"`
	PayerChain        string        `json:"payer_chain"`
	MerchantRecipient string        `json:"merchant_recipient"`
	ReceiverEmail     string        `json:"receiver_email,omitempty"`
	PayerWallet       string        `json:"payer_wallet,omitempty"`
	ErrorMessage      string        `json:"error_message,omitempty"`
	CreatedAt         string        `json:"created_at"`
	ExpiresAt         string        `json:"expires_at"`
	CompletedAt       string        `json:"completed_at,omitempty"`
	SourcePayment     *SourcePayment `json:"source_payment,omitempty"`
	BasePayment       *BasePayment   `json:"base_payment,omitempty"`
}

// ErrorResponse is the common error body from the API.
type ErrorResponse struct {
	Error      string `json:"error"`
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode"`
}
