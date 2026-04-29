package pay

import (
	"context"
	"net/http"
)

// SupportedChainsResponse is the response for GET /api/chains. Chains lists
// chains valid as PayerChain on CreateIntentRequest; TargetChains lists chains
// valid as TargetChain. The two lists are independent — a chain can be a valid
// payer without being a valid settlement destination (and vice versa).
type SupportedChainsResponse struct {
	Chains       []string `json:"chains"`
	TargetChains []string `json:"target_chains"`
}

// GetSupportedChains lists payer and target chains currently enabled by the
// backend (GET /api/chains). The endpoint is public-only and unaffected by
// auth options, so this method always hits the /api prefix even when the
// client is configured with WithBearerAuth.
func (c *Client) GetSupportedChains(ctx context.Context) (*SupportedChainsResponse, error) {
	var out SupportedChainsResponse

	err := c.do(ctx, &request{
		method:      http.MethodGet,
		uri:         apiPathPrefix + "/chains",
		absoluteURI: true,
		result:      &out,
	})
	if err != nil {
		return nil, err
	}

	return &out, nil
}
