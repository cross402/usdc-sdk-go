// Example shows how to use the Pay SDK (Client / v2 API flow): list supported
// chains, create an intent (with an optional target chain for multichain
// settlement), execute transfer (backend uses Agent wallet; no proof needed),
// then optionally query intent. For the public API flow (CreateIntent →
// SubmitProof), create a Client without auth options and see README.
// Run from repo root:
//
//	PAY_BASE_URL=https://api-pay.agent.tech PAY_API_KEY=key PAY_SECRET_KEY=secret go run ./example
//
// Set PAY_EMAIL to override the default merchant email (merchant@example.com).
// Set PAY_PAYER_CHAIN (default "base") and PAY_TARGET_CHAIN (default "" → backend
// settles on "base") to drive multichain-to-multichain payments.
//
// To only query an existing intent:
//
//	PAY_BASE_URL=... PAY_API_KEY=... PAY_SECRET_KEY=... PAY_INTENT_ID=uuid go run ./example
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	pay "github.com/cross402/usdc-sdk-go"
)

const defaultTimeout = 30 * time.Second

func fatal(msg string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
	os.Exit(1)
}

func printJSON(v any) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}

func main() {
	baseURL := os.Getenv("PAY_BASE_URL")
	apiKey := os.Getenv("PAY_API_KEY")
	secretKey := os.Getenv("PAY_SECRET_KEY")
	intentID := os.Getenv("PAY_INTENT_ID")

	if baseURL == "" || apiKey == "" || secretKey == "" {
		fmt.Fprintln(os.Stderr, "Set PAY_BASE_URL, PAY_API_KEY, and PAY_SECRET_KEY.")
		os.Exit(1)
	}

	opts := []pay.OptFn{pay.WithBearerAuth(apiKey, secretKey)}

	client, err := pay.NewClient(baseURL, opts...)
	if err != nil {
		fatal("NewClient", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	if intentID != "" {
		intent, getErr := client.GetIntent(ctx, intentID)
		if getErr != nil {
			fatal("GetIntent", getErr)
		}

		printJSON(intent)

		return
	}

	chains, err := client.GetSupportedChains(ctx)
	if err != nil {
		fatal("GetSupportedChains", err)
	}

	fmt.Printf("Payer chains:  %v\n", chains.Chains)
	fmt.Printf("Target chains: %v\n", chains.TargetChains)

	email := os.Getenv("PAY_EMAIL")
	if email == "" {
		email = "merchant@example.com"
	}

	payerChain := os.Getenv("PAY_PAYER_CHAIN")
	if payerChain == "" {
		payerChain = pay.ChainBase
	}

	req := &pay.CreateIntentRequest{
		Email:       email,
		Amount:      "10.00",
		PayerChain:  payerChain,
		TargetChain: os.Getenv("PAY_TARGET_CHAIN"), // empty → backend defaults to "base"
	}

	resp, err := client.CreateIntent(ctx, req)
	if err != nil {
		fatal("CreateIntent", err)
	}

	fmt.Printf("Intent created: %s (payer=%s, target=%s)\n", resp.IntentID, resp.PayerChain, resp.TargetChain)
	fmt.Printf("Status: %s\n", resp.Status)

	exec, err := client.ExecuteIntent(ctx, resp.IntentID)
	if err != nil {
		fatal("ExecuteIntent", err)
	}

	fmt.Printf("Execute result status: %s\n", exec.Status)

	if exec.Status == pay.StatusTargetSettled {
		fmt.Println("Settlement complete on target chain.")
	}

	printJSON(exec)
}
