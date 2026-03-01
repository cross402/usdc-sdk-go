package pay

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// SubmitProofResponse is the response for POST /intents/{intent_id} (200).
type SubmitProofResponse = ExecuteIntentResponse

// SubmitProof submits settle_proof after the payer has completed X402 payment
// on the source chain (POST {prefix}/intents/{intent_id}).
func (c *Client) SubmitProof(ctx context.Context, intentID, settleProof string) (*SubmitProofResponse, error) {
	if intentID == "" {
		return nil, &ValidationError{Message: ErrEmptyIntentID.Error(), Err: ErrEmptyIntentID}
	}

	if settleProof == "" {
		return nil, &ValidationError{Message: ErrEmptySettleProof.Error(), Err: ErrEmptySettleProof}
	}

	body, err := json.Marshal(map[string]string{"settle_proof": settleProof})
	if err != nil {
		return nil, &UnexpectedError{Err: err}
	}

	var out SubmitProofResponse

	err = c.do(ctx, &request{
		method: http.MethodPost,
		uri:    "/intents/" + url.PathEscape(intentID),
		body:   bytes.NewReader(body),
		result: &out,
	})
	if err != nil {
		return nil, err
	}

	return &out, nil
}
