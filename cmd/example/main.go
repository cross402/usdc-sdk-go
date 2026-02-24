// Example shows how to use the Pay SDK: create a client, create an intent,
// and optionally query intent status. Run from repo root:
//
//	PAY_BASE_URL=https://api-pay.agent.tech/api PAY_CLIENT_ID=id PAY_CLIENT_SECRET=secret go run ./cmd/example
//
// To only query an existing intent:
//
//	PAY_BASE_URL=... PAY_CLIENT_ID=... PAY_CLIENT_SECRET=... PAY_INTENT_ID=uuid go run ./cmd/example
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/agent-tech/agent-sdk-go"
)

func main() {
	baseURL := os.Getenv("PAY_BASE_URL")
	clientID := os.Getenv("PAY_CLIENT_ID")
	clientSecret := os.Getenv("PAY_CLIENT_SECRET")
	intentID := os.Getenv("PAY_INTENT_ID")

	if baseURL == "" || clientID == "" || clientSecret == "" {
		fmt.Fprintln(os.Stderr, "Set PAY_BASE_URL, PAY_CLIENT_ID, PAY_CLIENT_SECRET (and optionally PAY_INTENT_ID).")
		os.Exit(1)
	}

	client := pay.NewClient(baseURL, clientID, clientSecret)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if intentID != "" {
		intent, err := client.Intent(ctx, intentID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Intent: %v\n", err)
			os.Exit(1)
		}
		b, _ := json.MarshalIndent(intent, "", "  ")
		fmt.Println(string(b))
		return
	}

	req := &pay.CreateIntentRequest{
		Email:      "merchant@example.com",
		Amount:     "10.00",
		PayerChain: "solana",
	}
	resp, err := client.CreateIntent(ctx, req)
	if err != nil {
		var apiErr *pay.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode != 0 {
			fmt.Fprintf(os.Stderr, "CreateIntent API error %d: %s\n", apiErr.StatusCode, apiErr.Message)
		} else {
			fmt.Fprintf(os.Stderr, "CreateIntent: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Printf("Intent created: %s\n", resp.IntentID)
	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Payment requirements (for X402 signing):\n")
	b, _ := json.MarshalIndent(resp.PaymentRequirements, "", "  ")
	fmt.Println(string(b))
	fmt.Printf("\nNext: submit settle_proof with SubmitProof(ctx, %q, settleProof), then poll Intent(ctx, %q).\n", resp.IntentID, resp.IntentID)
}
