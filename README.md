# Pay SDK (Go)

[![Go Reference](https://pkg.go.dev/badge/github.com/agent-tech/agent-sdk-go.svg)](https://pkg.go.dev/github.com/agent-tech/agent-sdk-go)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue)

Go client for the Agent Tech v2 payment API — create intents, execute USDC transfers on Base, and query status. No wallet or signing on your side.

- **Zero dependencies** beyond the Go standard library
- **Two auth modes** — Bearer token or header-based API key
- **All payments settle on Base** chain via the backend Agent wallet

## Table of Contents

- [Install](#install)
- [Quick Start](#quick-start)
- [Authentication](#authentication)
- [API Methods](#api-methods)
- [Intent Lifecycle](#intent-lifecycle)
- [Supported Chains](#supported-chains)
- [Fee Breakdown](#fee-breakdown)
- [Error Handling](#error-handling)
- [Advanced](#advanced)

## Install

```bash
go get github.com/agent-tech/agent-sdk-go
```

## Quick Start

```go
package main

import (
    "context"
    "log"

    "github.com/agent-tech/agent-sdk-go"
)

func main() {
    client, err := pay.NewClient(
        "https://api-pay.agent.tech",
        pay.WithBearerAuth("your-client-id", "your-client-secret"),
    )
    if err != nil {
        log.Fatal(err)
    }
    ctx := context.Background()

    // 1. Create intent
    resp, err := client.CreateIntent(ctx, &pay.CreateIntentRequest{
        Email:      "merchant@example.com",
        Amount:     "100.50",
        PayerChain: "base",
    })
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Intent ID: %s", resp.IntentID)

    // 2. Execute transfer (backend signs with Agent wallet)
    exec, err := client.ExecuteIntent(ctx, resp.IntentID)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Status: %s", exec.Status)

    // 3. Query full receipt
    intent, err := client.GetIntent(ctx, resp.IntentID)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Final status: %s", intent.Status)
}
```

Or run the bundled example:

```bash
git clone https://github.com/agent-tech/agent-sdk-go
cd agent-sdk-go

PAY_BASE_URL=https://api-pay.agent.tech \
PAY_CLIENT_ID=your-client-id \
PAY_CLIENT_SECRET=your-client-secret \
go run ./cmd/example
```

Set `PAY_INTENT_ID` to skip creation and query an existing intent instead.

## Authentication

### Bearer token (recommended)

Base64-encodes `clientID:clientSecret` and sends it as `Authorization: Bearer <token>`.

```go
client, err := pay.NewClient(baseURL, pay.WithBearerAuth("client-id", "client-secret"))
```

### Header-based API key

Sends `X-Client-ID` and `X-API-Key` headers.

```go
client, err := pay.NewClient(baseURL, pay.WithAPIKeyAuth("client-id", "api-key"))
```

### Custom HTTP client

The default HTTP client uses a **30-second timeout**. Override with options:

```go
client, err := pay.NewClient(baseURL,
    pay.WithBearerAuth("id", "secret"),
    pay.WithHTTPClient(&http.Client{
        Timeout:   60 * time.Second,
        Transport: customTransport,
    }),
)
```

Or just change the timeout:

```go
client, err := pay.NewClient(baseURL,
    pay.WithBearerAuth("id", "secret"),
    pay.WithTimeout(60 * time.Second),
)
```

## API Methods

| Method | Endpoint | Description |
|---|---|---|
| `CreateIntent` | `POST /v2/intents` | Create a payment intent |
| `ExecuteIntent` | `POST /v2/intents/{id}/execute` | Execute transfer on Base with Agent wallet |
| `GetIntent` | `GET /v2/intents?intent_id=...` | Get intent status and receipt |

### CreateIntent

```go
resp, err := client.CreateIntent(ctx, &pay.CreateIntentRequest{
    Email:      "merchant@example.com", // or Recipient (exactly one required)
    Amount:     "100.50",               // 0.01–1,000,000 USDC, max 6 decimals
    PayerChain: "base",               // "base"
})
```

**`CreateIntentRequest` fields:**

| Field | JSON | Required | Description |
|---|---|---|---|
| `Email` | `email` | One of Email/Recipient | Recipient email address |
| `Recipient` | `recipient` | One of Email/Recipient | Recipient wallet address |
| `Amount` | `amount` | Yes | USDC amount as string (e.g. `"100.50"`) |
| `PayerChain` | `payer_chain` | Yes | Source chain: `base`|

### ExecuteIntent

No request body — the backend uses the Agent wallet to sign and transfer USDC on Base.

```go
exec, err := client.ExecuteIntent(ctx, resp.IntentID)
// exec.Status is typically "BASE_SETTLED"
```

### GetIntent (query status)

```go
intent, err := client.GetIntent(ctx, intentID)
switch intent.Status {
case pay.StatusBaseSettled:
    // use intent.BasePayment for receipt
case pay.StatusExpired, pay.StatusVerificationFailed:
    // terminal failure
default:
    // still processing — poll again
}
```

## Intent Lifecycle

Intents expire **10 minutes** after creation.

```
                          ┌──────────────────┐
                          │ AWAITING_PAYMENT  │
                          └────────┬─────────┘
                                   │
                      ┌────────────┼────────────┐
                      │            │             │
                      ▼            ▼             ▼
               ┌──────────┐ ┌──────────┐ ┌─────────────────────┐
               │ EXPIRED  │ │ PENDING  │ │ VERIFICATION_FAILED │
               └──────────┘ └────┬─────┘ └─────────────────────┘
                                 │
                                 ▼
                        ┌────────────────┐
                        │ SOURCE_SETTLED │
                        └───────┬────────┘
                                │
                                ▼
                        ┌───────────────┐
                        │ BASE_SETTLING │
                        └───────┬───────┘
                                │
                                ▼
                        ┌──────────────┐
                        │ BASE_SETTLED │
                        └──────────────┘
```

Use the status constants instead of bare strings:

| Constant | Value | Description |
|---|---|---|
| `pay.StatusAwaitingPayment` | `AWAITING_PAYMENT` | Intent created, waiting for execution |
| `pay.StatusPending` | `PENDING` | Execution initiated, processing |
| `pay.StatusVerificationFailed` | `VERIFICATION_FAILED` | Source payment verification failed (terminal) |
| `pay.StatusSourceSettled` | `SOURCE_SETTLED` | Source chain payment confirmed |
| `pay.StatusBaseSettling` | `BASE_SETTLING` | USDC transfer on Base in progress |
| `pay.StatusBaseSettled` | `BASE_SETTLED` | Transfer complete — check `base_payment` for receipt (terminal) |
| `pay.StatusExpired` | `EXPIRED` | Intent was not executed within 10 minutes (terminal) |

## Supported Chains

| Chain | Identifier | Role |
|---|---|---|
| base | `base` | Payer chain (source) |


All payments settle on **Base** regardless of the source chain. The `payer_chain` field in `CreateIntentRequest` specifies the source chain only.

## Fee Breakdown

The `FeeBreakdown` struct is returned in all intent response types (embedded via `IntentBase`):

| Field | JSON | Description |
|---|---|---|
| `SourceChain` | `source_chain` | Source chain identifier |
| `SourceChainFee` | `source_chain_fee` | Gas/network fee on the source chain |
| `TargetChain` | `target_chain` | Target chain (always `"base"`) |
| `TargetChainFee` | `target_chain_fee` | Gas/network fee on Base |
| `PlatformFee` | `platform_fee` | Platform service fee |
| `PlatformFeePercentage` | `platform_fee_percentage` | Platform fee as a percentage |
| `TotalFee` | `total_fee` | Sum of all fees |

**Amount rules:**
- Minimum: **0.01 USDC**
- Maximum: **1,000,000 USDC**
- Up to **6 decimal places** (e.g. `"0.000001"`, `"123.45"`)

## Error Handling

The SDK uses two error types:

**`APIError`** — returned for non-2xx HTTP responses from the API:

```go
var apiErr *pay.APIError
if errors.As(err, &apiErr) {
    log.Printf("HTTP %d: %s", apiErr.StatusCode, apiErr.Message)
}
```

**`ValidationError`** — returned when the SDK rejects a request before it reaches the API (e.g. nil request, empty intent ID):

```go
var valErr *pay.ValidationError
if errors.As(err, &valErr) {
    log.Printf("Invalid input: %s", valErr.Message)
}
```

| Status Code | Meaning |
|---|---|
| 400 | Bad request — invalid parameters, amount out of range, or malformed input |
| 401 | Unauthorized — missing or invalid credentials |
| 403 | Forbidden — insufficient permissions for this operation |
| 404 | Not found — intent does not exist |
| 429 | Rate limited — too many requests (60 req/min/IP typical) |
| 503 | Service unavailable — temporary backend issue |

## Advanced

### Custom HTTP client with retry

```go
client, err := pay.NewClient(baseURL,
    pay.WithBearerAuth(clientID, clientSecret),
    pay.WithHTTPClient(&http.Client{
        Timeout:   60 * time.Second,
        Transport: &retryTransport{base: http.DefaultTransport},
    }),
)
```

### Rate limiting

The API allows approximately **60 requests per IP per minute**. On HTTP 429, implement exponential backoff:

```go
var apiErr *pay.APIError
if errors.As(err, &apiErr) && apiErr.StatusCode == 429 {
    time.Sleep(backoff)
    // retry
}
```
