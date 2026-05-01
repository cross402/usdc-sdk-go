# Pay SDK (Go)

[![Go Reference](https://pkg.go.dev/badge/github.com/cross402/usdc-sdk-go.svg)](https://pkg.go.dev/github.com/cross402/usdc-sdk-go)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue)

Go client for the Agent Tech payment API — create intents, execute USDC transfers across chains, and query status.

- **One unified client** — a single `Client` that auto-selects the API prefix based on auth:
  - With auth (`WithBearerAuth`) → `/v2` prefix — create intent → execute (backend signs with Agent wallet)
  - Without auth → `/api` prefix (public mode) — create intent → payer signs X402 & pays → submit settle_proof
- **Zero dependencies** beyond the Go standard library
- **Multichain to multichain** — pick any supported `payer_chain` and any supported `target_chain` (defaults to `base` when omitted)

## Table of Contents

- [Install](#install)
- [Quick Start](#quick-start)
- [Authenticated vs Public Mode](#authenticated-vs-public-mode)
- [Authentication](#authentication)
- [API Methods](#api-methods)
- [Intent Lifecycle](#intent-lifecycle)
- [Supported Chains](#supported-chains)
- [Fee Breakdown](#fee-breakdown)
- [Error Handling](#error-handling)
- [Advanced](#advanced)

## Install

```bash
go get github.com/cross402/usdc-sdk-go
```

## Quick Start

### Authenticated (v2 API)

```go
package main

import (
    "context"
    "log"

    "github.com/cross402/usdc-sdk-go"
)

func main() {
    client, err := pay.NewClient(
        "https://api-pay.agent.tech",
        pay.WithBearerAuth("your-api-key", "your-secret-key"),
    )
    if err != nil {
        log.Fatal(err)
    }
    ctx := context.Background()

    // 1. Create intent (multichain: pay from Base, settle on Solana)
    resp, err := client.CreateIntent(ctx, &pay.CreateIntentRequest{
        Email:       "merchant@example.com",
        Amount:      "100.50",
        PayerChain:  pay.ChainBase,
        TargetChain: pay.ChainSolanaMainnet, // omit to default to "base"
    })
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Intent ID: %s (settling on %s)", resp.IntentID, resp.TargetChain)

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

### Public mode (no auth)

```go
client, err := pay.NewClient("https://api-pay.agent.tech")
if err != nil {
    log.Fatal(err)
}
resp, err := client.CreateIntent(ctx, &pay.CreateIntentRequest{
    Email:      "merchant@example.com",
    Amount:     "100.50",
    PayerChain: "solana",
})
// Use resp.PaymentRequirements for payer to sign X402...
// After payment, submit the settle_proof:
proof, err := client.SubmitProof(ctx, resp.IntentID, settleProof)
```

Or run the bundled example:

```bash
git clone https://github.com/cross402/usdc-sdk-go
cd usdc-sdk-go

PAY_BASE_URL=https://api-pay.agent.tech \
PAY_API_KEY=your-api-key \
PAY_SECRET_KEY=your-secret-key \
go run ./example
```

Set `PAY_INTENT_ID` to skip creation and query an existing intent instead.

## Authenticated vs Public Mode

| | Authenticated (`/v2`) | Public (`/api`) |
|---|---|---|
| **Auth** | `WithBearerAuth` | None |
| **Flow** | CreateIntent → ExecuteIntent → GetIntent | CreateIntent → (payer pays) → SubmitProof → GetIntent |
| **Use when** | Integrator has no wallet; backend Agent signs | Integrator has payer's wallet; can sign X402 and submit settle_proof |

Both modes use the same `Client` — the prefix is selected automatically based on whether an auth option is provided.

## Authentication

### Bearer token (recommended)

Base64-encodes `apiKey:secretKey` and sends it as `Authorization: Bearer <token>`.

```go
client, err := pay.NewClient(baseURL, pay.WithBearerAuth("api-key", "secret-key"))
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

| Method | Auth Required | Endpoint | Description |
|---|---|---|---|
| `GetSupportedChains` | No | `GET /api/chains` | List payer and target chains the backend currently accepts |
| `CreateIntent` | No | `POST {prefix}/intents` | Create a payment intent |
| `ExecuteIntent` | Yes | `POST /v2/intents/{id}/execute` | Execute transfer with the Agent wallet (settles on the chosen target chain) |
| `SubmitProof` | No (public mode only) | `POST /api/intents/{id}` | Submit settle_proof after payer completes X402 payment |
| `GetIntent` | No / Yes | `GET {prefix}/intents?intent_id=...` | Get intent status and receipt. With auth, only intents owned by the calling agent are returned (404 otherwise — see below) |
| `ListIntents` | Yes | `GET /v2/intents/list` | Paginated list of intents owned by the calling agent (most recent first) |
| `GetMe` | Yes | `GET /v2/me` | Returns the calling agent's identity (id, number, name, status, wallets) |

### GetSupportedChains

```go
chains, err := client.GetSupportedChains(ctx)
// chains.Chains       — chains valid as PayerChain
// chains.TargetChains — chains valid as TargetChain
```

This endpoint is public-only on the backend; the SDK always hits `/api/chains` even when the client is configured with auth.

### CreateIntent

```go
resp, err := client.CreateIntent(ctx, &pay.CreateIntentRequest{
    Email:       "merchant@example.com",   // or Recipient (exactly one required)
    Amount:      "100.50",                 // 0.02–1,000,000 USDC, max 6 decimals
    PayerChain:  pay.ChainBase,            // source chain
    TargetChain: pay.ChainSolanaMainnet,   // optional — omit to settle on "base"
})
```

**`CreateIntentRequest` fields:**

| Field | JSON | Required | Description |
|---|---|---|---|
| `Email` | `email` | One of Email/Recipient | Recipient email address |
| `Recipient` | `recipient` | One of Email/Recipient | Recipient wallet address |
| `Amount` | `amount` | Yes | USDC amount as string (e.g. `"100.50"`) |
| `PayerChain` | `payer_chain` | Yes | Source chain (see [Supported Chains](#supported-chains)) |
| `TargetChain` | `target_chain` | No | Settlement chain. Omit to default to `"base"`. Validate against `GetSupportedChains().TargetChains`. |

### ExecuteIntent

No request body — the backend uses the Agent wallet to sign the source-chain transfer and settle on the chosen target chain. Requires auth.

```go
exec, err := client.ExecuteIntent(ctx, resp.IntentID)
// exec.Status is typically StatusTargetSettled
```

### SubmitProof

Submits a settle_proof after the payer completes X402 payment on the source chain. No auth required.

```go
proof, err := client.SubmitProof(ctx, intentID, settleProof)
```

### GetIntent (query status)

```go
intent, err := client.GetIntent(ctx, intentID)
switch intent.Status {
case pay.StatusTargetSettled:
    // use intent.TargetPayment for receipt; intent.TargetChain names the chain
case pay.StatusExpired, pay.StatusVerificationFailed:
    // terminal failure
default:
    // still processing — poll again
}
```

> **v2 ownership policy:** when the client is configured with `WithBearerAuth`, `GetIntent` (and `ExecuteIntent`/`SubmitProof`) only return intents owned by the calling agent. Looking up an intent owned by a different agent — or one created via the public `/api` flow — returns **HTTP 404**, not 403, so callers can't probe foreign intent IDs.

### ListIntents (paginated, owned by the calling agent)

```go
list, err := client.ListIntents(ctx, /*page*/ 1, /*pageSize*/ 20)
if err != nil {
    log.Fatal(err)
}

for _, it := range list.Intents {
    log.Printf("%s — %s (%s → %s)", it.IntentID, it.Status, it.PayerChain, it.TargetChain)
}

log.Printf("total=%d page=%d size=%d", list.Total, list.Page, list.PageSize)
```

- **Auth required.** Returns only intents owned by the calling agent.
- `page` is 1-indexed; the server caps it at 1,000,000.
- `pageSize` must be in `[1, 100]`. Pass `0` for either parameter to use the server defaults (page 1, 20 per page).
- Out-of-range values (`pageSize > 100`, `page < 0`) are rejected by the SDK before reaching the API.

### GetMe (authenticated identity)

```go
me, err := client.GetMe(ctx)
if err != nil {
    log.Fatal(err)
}

log.Printf("agent %s (%s) status=%s", me.AgentID, me.AgentNumber, me.Status)
log.Printf("base wallet=%s solana wallet=%s", me.WalletAddress, me.SolanaWalletAddress)
```

`GetMe` is served from middleware context — it does not hit the database — and is the cheapest way to verify an API key is live and bound to the agent the caller expects.

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
                       ┌─────────────────┐
                       │ TARGET_SETTLING │
                       └────────┬────────┘
                                │
                                ▼
                        ┌────────────────┐
                        │ TARGET_SETTLED │
                        └────────────────┘
```

Use the status constants instead of bare strings:

| Constant | Value | Description |
|---|---|---|
| `pay.StatusAwaitingPayment` | `AWAITING_PAYMENT` | Intent created, waiting for execution |
| `pay.StatusPending` | `PENDING` | Execution initiated, processing |
| `pay.StatusVerificationFailed` | `VERIFICATION_FAILED` | Source payment verification failed (terminal) |
| `pay.StatusSourceSettled` | `SOURCE_SETTLED` | Source chain payment confirmed |
| `pay.StatusTargetSettling` | `TARGET_SETTLING` | USDC transfer on the target chain in progress |
| `pay.StatusTargetSettled` | `TARGET_SETTLED` | Transfer complete — check `target_payment` for receipt (terminal) |
| `pay.StatusPartialSettlement` | `PARTIAL_SETTLEMENT` | Source settled but target failed — manual reconciliation required |
| `pay.StatusExpired` | `EXPIRED` | Intent was not executed within 10 minutes (terminal) |

## Supported Chains

The set of payer and target chains is configured at runtime by the backend. Call `GetSupportedChains` to discover what's currently enabled — the two lists are independent (a chain can be a valid payer without being a valid settlement destination, and vice versa).

```go
chains, err := client.GetSupportedChains(ctx)
// chains.Chains       — valid as PayerChain
// chains.TargetChains — valid as TargetChain (omitted defaults to "base")
```

Use the `Chain*` constants instead of bare strings:

| Chain | Testnet Constant | Mainnet Constant |
|---|---|---|
| Solana | `pay.ChainSolanaDevnet` (`"solana-devnet"`) | `pay.ChainSolanaMainnet` (`"solana-mainnet-beta"`) |
| Base | `pay.ChainBaseSepolia` (`"base-sepolia"`) | `pay.ChainBase` (`"base"`) |
| BSC | `pay.ChainBSCTestnet` (`"bsc-testnet"`) | `pay.ChainBSC` (`"bsc"`) |
| Polygon | `pay.ChainPolygonAmoy` (`"polygon-amoy"`) | `pay.ChainPolygon` (`"polygon"`) |
| Arbitrum | `pay.ChainArbitrumSepolia` (`"arbitrum-sepolia"`) | `pay.ChainArbitrum` (`"arbitrum"`) |
| Ethereum | `pay.ChainEthereumSepolia` (`"ethereum-sepolia"`) | `pay.ChainEthereum` (`"ethereum"`) |
| Monad | `pay.ChainMonadTestnet` (`"monad-testnet"`) | `pay.ChainMonad` (`"monad"`) |
| HyperEVM | `pay.ChainHyperEVMTestnet` (`"hyperevm-testnet"`) | `pay.ChainHyperEVM` (`"hyperevm"`) |
| SKALE Base | — | `pay.ChainSKALEBase` (`"skale-base"`, payer-only) |
| MegaETH | — | `pay.ChainMegaETH` (`"megaeth"`, payer-only) |

`ChainSKALEBase` and `ChainMegaETH` are accepted only as `PayerChain`; the backend rejects them as `TargetChain`.

```go
resp, err := client.CreateIntent(ctx, &pay.CreateIntentRequest{
    Email:       "merchant@example.com",
    Amount:      "100.50",
    PayerChain:  pay.ChainBase,           // use constants instead of bare strings
    TargetChain: pay.ChainSolanaMainnet,  // optional — omit to settle on "base"
})
```

## Fee Breakdown

The `FeeBreakdown` struct is returned in all intent response types (embedded via `IntentBase`):

| Field | JSON | Description |
|---|---|---|
| `SourceChain` | `source_chain` | Source chain identifier |
| `SourceChainFee` | `source_chain_fee` | Gas/network fee on the source chain |
| `TargetChain` | `target_chain` | Settlement chain identifier |
| `TargetChainFee` | `target_chain_fee` | Gas/network fee on the target chain |
| `PlatformFee` | `platform_fee` | Platform service fee |
| `PlatformFeePercentage` | `platform_fee_percentage` | Platform fee as a percentage |
| `TotalFee` | `total_fee` | Sum of all fees |

**Amount rules:**
- Minimum: **0.02 USDC**
- Maximum: **1,000,000 USDC**
- Up to **6 decimal places** (e.g. `"0.000001"`, `"123.45"`)

## Error Handling

The SDK uses typed errors and sentinel values for precise error matching.

### Error types

**`RequestError`** — returned for HTTP 4xx/5xx responses from the API:

```go
var reqErr *pay.RequestError
if errors.As(err, &reqErr) {
    log.Printf("HTTP %d: %s", reqErr.StatusCode, reqErr.Body)
}
```

**`ValidationError`** — returned when the SDK rejects a request before it reaches the API (e.g. empty intent ID). Wraps a sentinel error for `errors.Is` matching:

```go
var valErr *pay.ValidationError
if errors.As(err, &valErr) {
    log.Printf("Invalid input: %s", valErr.Message)
}
```

**`UnexpectedError`** — wraps unexpected internal errors (JSON marshal failure, request creation, etc.):

```go
var unexpErr *pay.UnexpectedError
if errors.As(err, &unexpErr) {
    log.Printf("Unexpected: %v", unexpErr.Err)
}
```

### Sentinel errors

Use `errors.Is` to check for specific validation failures:

```go
if errors.Is(err, pay.ErrEmptyIntentID) {
    // intent ID was empty
}
```

| Sentinel | Meaning |
|---|---|
| `ErrEmptyBaseURL` | `baseURL` was empty in `NewClient` |
| `ErrEmptyIntentID` | `intentID` was empty |
| `ErrEmptySettleProof` | `settleProof` was empty in `SubmitProof` |
| `ErrMissingAuth` | `ExecuteIntent`, `ListIntents`, or `GetMe` called without auth |
| `ErrNilParams` | `CreateIntentRequest` was nil |
| `ErrSubmitProofNotAllowed` | `SubmitProof` called with `WithBearerAuth` (use `ExecuteIntent` instead) |
| `ErrInvalidPagination` | `page < 0`, `pageSize < 0`, or `pageSize > 100` in `ListIntents` |

### HTTP status codes

| Status Code | Meaning |
|---|---|
| 400 | Bad request — invalid parameters, amount out of range, malformed input, or out-of-range pagination (`page`, `page_size`) |
| 401 | Unauthorized — missing or invalid credentials |
| 402 | Payment required — agent wallet has insufficient USDC to satisfy `ExecuteIntent` |
| 403 | Forbidden — reserved for future use; v2 ownership rejection returns 404, not 403 |
| 404 | Not found — intent does not exist *or* is owned by a different agent (uniform response prevents existence-leak probing) |
| 429 | Rate limited — too many requests (60 req/min/IP typical) |
| 503 | Service unavailable — temporary backend issue (e.g. proxy wallet temporarily out of funds) |

## Advanced

### Custom HTTP client with retry

```go
client, err := pay.NewClient(baseURL,
    pay.WithBearerAuth(apiKey, secretKey),
    pay.WithHTTPClient(&http.Client{
        Timeout:   60 * time.Second,
        Transport: &retryTransport{base: http.DefaultTransport},
    }),
)
```

### Rate limiting

The API allows approximately **60 requests per IP per minute**. On HTTP 429, implement exponential backoff:

```go
var reqErr *pay.RequestError
if errors.As(err, &reqErr) && reqErr.StatusCode == 429 {
    time.Sleep(backoff)
    // retry
}
```
