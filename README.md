# Pay SDK (Go)

Go client for the v2 payment API (API Key–authenticated, Agent-bound intents and Base chain settlement). It is a **library (SDK)** that your program imports; you create a client, then call `CreateIntent`, `SubmitProof`, and `Intent` to integrate with the API.

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
       // Next: use resp.PaymentRequirements for X402, then SubmitProof, then Intent to poll.
   }
   ```

### Option 2: Run the example from this repo

Clone the repo and run the example (no need to publish the module first):

```bash
git clone https://github.com/agent-tech/agent-sdk-go
cd agent-sdk-go

# Create an intent (replace with your API credentials)
PAY_BASE_URL=https://api-pay.agent.tech/api \
PAY_CLIENT_ID=your-client-id \
PAY_CLIENT_SECRET=your-client-secret \
go run ./cmd/example

# Or only query an existing intent
PAY_INTENT_ID=your-intent-uuid \
PAY_BASE_URL=... PAY_CLIENT_ID=... PAY_CLIENT_SECRET=... \
go run ./cmd/example
```

The example prints the created intent ID and `payment_requirements`, or the full intent object when `PAY_INTENT_ID` is set.

---

## API overview

| Method           | What it does                          |
|------------------|----------------------------------------|
| `CreateIntent`   | Create a payment intent (POST /v2/intents). |
| `SubmitProof`    | Submit X402 settle proof (POST /v2/intents/{id}). |
| `Intent`         | Get intent status and receipt (GET /v2/intents?intent_id=...). |

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
// use resp.IntentID, resp.PaymentRequirements, etc.
```

### Submit settle proof

After the payer completes the X402 payment and has `settle_proof`:

```go
proofResp, err := client.SubmitProof(ctx, resp.IntentID, settleProof)
if err != nil {
    return err
}
// proofResp.Status is typically "PENDING"; poll Intent() until BASE_SETTLED or terminal.
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

1. **Create intent** – `CreateIntent` with `email` or `recipient`, `amount`, `payer_chain`. Use `intent_id` and `payment_requirements` for the next steps.
2. **Client signs X402** – Use `payment_requirements` off-chain to produce a signed X402 authorization (not part of this API).
3. **Payer pays on source chain** – Payer completes payment and obtains `settle_proof`.
4. **Submit proof** – `SubmitProof(ctx, intentID, settleProof)`. Then poll `Intent` until `status` is `BASE_SETTLED` or another terminal state.
5. **Receipt** – Use `Intent()` response (`base_payment`, etc.) as the merchant receipt.

---

## Errors

All non-2xx responses are returned as `*pay.APIError` with `StatusCode` and `Message`. Use `errors.As(err, &apiErr)` to handle them. The API may return 400, 401, 403, 404, 429 (rate limit), or 503.

---

## Rate limit

The v2 API is typically limited to 60 requests per IP per minute. On 429, implement retry with backoff.
