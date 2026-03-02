// Example shows how to use the Pay SDK (Client / v2 API flow): create a client,
// create an intent, execute transfer (backend uses Agent wallet; no proof needed),
// then optionally query intent. For the public API flow (CreateIntent → SubmitProof),
// use pay.NewPublicClient and see README.
// Run from repo root:
//
//	PAY_BASE_URL=https://api-pay.agent.tech PAY_CLIENT_ID=id PAY_CLIENT_SECRET=secret go run ./example
//
// Set PAY_EMAIL to override the default merchant email (merchant@example.com).
//
// To only query an existing intent:
//
//	PAY_BASE_URL=... PAY_CLIENT_ID=... PAY_CLIENT_SECRET=... PAY_INTENT_ID=uuid go run ./example
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	pay "github.com/agent-tech/agent-sdk-go"
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
	clientID := os.Getenv("PAY_CLIENT_ID")
	clientSecret := os.Getenv("PAY_CLIENT_SECRET")
	intentID := os.Getenv("PAY_INTENT_ID")

	if baseURL == "" || clientID == "" || clientSecret == "" {
		fmt.Fprintln(os.Stderr, "Set PAY_BASE_URL, PAY_CLIENT_ID, and PAY_CLIENT_SECRET.")
		os.Exit(1)
	}

	opts := []pay.OptFn{pay.WithBearerAuth(clientID, clientSecret)}

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

	email := os.Getenv("PAY_EMAIL")
	if email == "" {
		email = "merchant@example.com"
	}

	req := &pay.CreateIntentRequest{
		Email:      email,
		Amount:     "10.00",
		PayerChain: "base",
	}

	resp, err := client.CreateIntent(ctx, req)
	if err != nil {
		fatal("CreateIntent", err)
	}

	fmt.Printf("Intent created: %s\n", resp.IntentID)
	fmt.Printf("Status: %s\n", resp.Status)

	exec, err := client.ExecuteIntent(ctx, resp.IntentID)
	if err != nil {
		fatal("ExecuteIntent", err)
	}

	fmt.Printf("Execute result status: %s\n", exec.Status)
	printJSON(exec)
}
