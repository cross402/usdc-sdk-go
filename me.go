package pay

import (
	"context"
	"net/http"
)

// Me describes the authenticated agent identified by the API key in use.
type Me struct {
	AgentID             string `json:"agent_id"`
	AgentNumber         string `json:"agent_number"`
	Name                string `json:"name"`
	Status              string `json:"status"`
	WalletAddress       string `json:"wallet_address,omitempty"`
	SolanaWalletAddress string `json:"solana_wallet_address,omitempty"`
}

// GetMe returns the calling agent's identity (GET /v2/me). Requires auth.
func (c *Client) GetMe(ctx context.Context) (*Me, error) {
	if c.authFunc == nil {
		return nil, &ValidationError{Message: ErrMissingAuth.Error(), Err: ErrMissingAuth}
	}

	var out Me

	err := c.do(ctx, &request{
		method: http.MethodGet,
		uri:    "/me",
		result: &out,
	})
	if err != nil {
		return nil, err
	}

	return &out, nil
}
