# Pay SDK (Go)

Go client for the v2 payment API (API Key–authenticated, Agent-bound intents and Base chain settlement). It is a **library (SDK)** that your program imports; you create a client, then call `CreateIntent`, `ExecuteIntent`, and `Intent` to integrate with the API. No wallet or signing on your side—the backend uses the Agent wallet to transfer USDC on Base.

---

## Quick start (new users)

### Option 1: Use the SDK in your own Go project

1. **Add the module**:

   ```bash
   go get github.com/agent-tech/agent-sdk-go
   ```

2. **In your code**, create a client and call the API:

   ```go
   package main

   import (
       "context"
       "log"

       "github.com/agent-tech/agent-sdk-go"
   )

   func main() {
       client := pay.NewClient(
           "https://api-pay.agent.tech/api",
           "your-client-id",
           "your-client-secret",
       )
       ctx := context.Background()

       resp, err := client.CreateIntent(ctx, &pay.CreateIntentRequest{
           Email:      "merchant@example.com",
           Amount:     "100.50",
           PayerChain: "solana",
       })
       if err != nil {
           log.Fatal(err)
       }
       log.Printf("Intent ID: %s", resp.IntentID)

       exec, err := client.ExecuteIntent(ctx, resp.IntentID)
       if err != nil {
           log.Fatal(err)
       }
       log.Printf("Status: %s", exec.Status)
       // Optional: client.Intent(ctx, resp.IntentID) for full receipt
   }
   ```

### Option 2: Run the example from this repo

Clone the repo and run the example (no need to publish the module first):

```bash
git clone https://github.com/agent-tech/agent-sdk
cd agent-sdk

# Create intent and execute transfer (replace with your API credentials)
PAY_BASE_URL=https://api-pay.agent.tech/api \
PAY_CLIENT_ID=your-client-id \
PAY_CLIENT_SECRET=your-client-secret \
go run ./cmd/example

# Or only query an existing intent
PAY_INTENT_ID=your-intent-uuid \
PAY_BASE_URL=... PAY_CLIENT_ID=... PAY_CLIENT_SECRET=... \
go run ./cmd/example
```

The example creates an intent, calls execute, then prints the result (or only queries intent when `PAY_INTENT_ID` is set).

---

## API overview

| Method         | What it does                                                |
|----------------|-------------------------------------------------------------|
| `CreateIntent` | Create a payment intent (POST /v2/intents).                 |
| `ExecuteIntent`| Execute transfer on Base with Agent wallet (POST /v2/intents/{id}/execute). No body or proof. |
| `Intent`       | Get intent status and receipt (GET /v2/intents?intent_id=...). |

### Create a client

**Bearer (recommended):** Base64 of `clientID:clientSecret` as `Authorization: Bearer <token>`.

```go
client := pay.NewClient("https://api-pay.agent.tech/api", "your-client-id", "your-client-secret")
```

**Headers:** `X-Client-ID` and `X-API-Key`.

```go
client := pay.NewClientWithAPIKey("https://api-pay.agent.tech/api", "your-client-id", "your-api-key")
```

### Create an intent

```go
resp, err := client.CreateIntent(ctx, &pay.CreateIntentRequest{
    Email:      "merchant@example.com",
    Amount:     "100.50",
    PayerChain: "solana",
})
if err != nil {
    var apiErr *pay.APIError
    if errors.As(err, &apiErr) {
        // handle apiErr.StatusCode, apiErr.Message
    }
    return err
}
// use resp.IntentID for ExecuteIntent
```

### Execute transfer

No wallet or settle_proof on your side. Backend signs with the Agent wallet and transfers USDC on Base.

```go
exec, err := client.ExecuteIntent(ctx, resp.IntentID)
if err != nil {
    return err
}
// exec.Status is typically "BASE_SETTLED"
```

### Query intent status

```go
intent, err := client.Intent(ctx, intentID)
if err != nil {
    return err
}
switch intent.Status {
case "BASE_SETTLED":
    // use intent.BasePayment, etc.
case "EXPIRED", "VERIFICATION_FAILED":
    // terminal failure
default:
    // keep polling
}
```

---

## Integration flow

1. **Create intent** – `CreateIntent` with `email` or `recipient`, `amount`, `payer_chain`. Get `intent_id`.
2. **Execute transfer** – `ExecuteIntent(ctx, intentID)`. No body; backend uses the Agent wallet to sign and transfer USDC on Base.
3. **Query receipt** – `Intent(ctx, intentID)` to get full status and `base_payment` as receipt.

---

## Errors

All non-2xx responses are returned as `*pay.APIError` with `StatusCode` and `Message`. Use `errors.As(err, &apiErr)` to handle them. The API may return 400, 401, 403, 404, 429 (rate limit), or 503.

---

## Rate limit

The v2 API is typically limited to 60 requests per IP per minute. On 429, implement retry with backoff.
