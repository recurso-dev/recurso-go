# recurso-go

Official Go SDK for the [Recurso](https://github.com/recurso-dev/recurso) billing API — 45 resources / 276 methods covering the full catalog: plans, customers, the subscription lifecycle (pause/resume/cancel, add-ons, plan-change and cancel previews, minimum commitments, financial summaries), invoices (drill-downs into journal entries, payment attempts, status history, printable documents, EU e-invoicing), usage-based billing (billable metrics, plan charges and simulation, prepaid wallets, usage alerts, batch ingestion, audit trail), coupons, quotes, entitlements, webhooks, events, credit notes (approval workflow, journal entries), gifts, referrals, mandates, the double-entry ledger (trial balance, deferred-revenue rollforward, CSV export), finance (reconciliation, month-end close pack, revenue recognition), analytics (MRR waterfall, unit economics, revenue by plan/geography, dunning insights, natural-language questions), Indian GST returns, tax/e-invoicing/branding settings, BYO gateway and integration connections, migrations from Stripe/RevenueCat/Chargebee, team members, SSO, and the tenant account.

Standard library only — no third-party dependencies. Requires Go 1.22+.

The client is hand-crafted but generated-in-spirit from the OpenAPI 3.1 description in [`cmd/api/openapi.yaml`](https://github.com/recurso-dev/recurso/blob/main/cmd/api/openapi.yaml): types and paths mirror the spec exactly, and monetary amounts are `int64` values in the currency's smallest unit (cents/paise).

## Install

```bash
go get github.com/recurso-dev/recurso-go
```

## Quickstart

```go
package main

import (
	"context"
	"errors"
	"log"

	recurso "github.com/recurso-dev/recurso-go"
)

func main() {
	ctx := context.Background()
	client := recurso.NewClient("sk_live_your_api_key")

	plan, err := client.Plans.Create(ctx, &recurso.PlanCreateParams{
		Name:          "Pro Plan",
		Code:          "PRO-USD",
		Amount:        2900, // minor units ($29.00)
		Currency:      "USD",
		IntervalUnit:  "month",
		IntervalCount: 1,
	})
	if err != nil {
		log.Fatal(err)
	}

	customer, err := client.Customers.Create(ctx, &recurso.CustomerCreateParams{
		Name:    "Jane User",
		Email:   "jane@example.com",
		Country: "US",
	})
	if err != nil {
		log.Fatal(err)
	}

	sub, err := client.Subscriptions.Create(ctx, &recurso.SubscriptionCreateParams{
		CustomerID: customer.ID,
		PlanID:     plan.ID,
	})
	if err != nil {
		// See "Error handling" below.
		var apiErr *recurso.APIError
		if errors.As(err, &apiErr) {
			log.Fatalf("recurso: %s (%s, HTTP %d)", apiErr.Message, apiErr.Code, apiErr.StatusCode)
		}
		log.Fatal(err)
	}

	log.Printf("created subscription %s (%s)", sub.ID, sub.Status)
}
```

## Authentication

Every request carries your API key as a bearer token
(`Authorization: Bearer <key>`). Pass the key to `NewClient`:

```go
client := recurso.NewClient("sk_live_your_api_key")
```

## Configuration

`NewClient` accepts functional options:

```go
client := recurso.NewClient(
	"sk_live_your_api_key",
	recurso.WithBaseURL("https://billing.example.com/v1"), // default: https://api.recurso.dev/v1
	recurso.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
)
```

- `WithBaseURL` targets a self-hosted or staging deployment. The base URL
  includes the `/v1` version prefix (`client.System.Version` strips it to
  reach the unversioned `/version` endpoint).
- `WithHTTPClient` supplies a custom `*http.Client` for timeouts, proxies, or
  instrumentation.

## Resource layout

Every endpoint hangs off a resource field on the `Client`. Every method
takes a `context.Context` first and returns `(T, error)`; printable
documents and CSV exports return `[]byte`.

| Resource | Field | Covers |
|---|---|---|
| Account | `client.Account` | Tenant profile |
| Accounting | `client.Accounting` | QuickBooks/Xero/NetSuite/Tally connections, OAuth start, sync |
| Analytics | `client.Analytics` | MRR (by entity, waterfall), invoice aging, unit economics, revenue by plan/geography, dunning insights, usage stats, natural-language questions |
| Audit logs | `client.AuditLogs` | Audit trail |
| Auth | `client.Auth` | TOTP MFA enrollment and login sessions (session-cookie flows) |
| Billable metrics | `client.BillableMetrics` | Meters, and the plans priced on them |
| Billing | `client.Billing` | The tenant's own managed-cloud plan and trial status |
| Cancel flows | `client.CancelFlows` | Retention flows, steps, sessions, stats |
| Churn | `client.Churn` | High-risk customers and alerts |
| Collections | `client.Collections` | Collections queue, funnel, failures, per-invoice actions |
| Consents | `client.Consents` | Record and revoke customer consents |
| Coupons | `client.Coupons` | Discount codes |
| Credit notes | `client.CreditNotes` | Issue, approve/reject/void, journal entries, printable document |
| Customers | `client.Customers` | CRUD, payment method, churn score, consents, credit statement, financial summary |
| Developer | `client.Developer` | API keys (create, list, revoke) |
| Disputes | `client.Disputes` | Invoice disputes |
| Dunning campaigns | `client.DunningCampaigns` | Campaigns and steps |
| Entities | `client.Entities` | Legal entities (Multi-Entity Books) |
| Entitlements | `client.Entitlements` | Plan and customer entitlements, checks |
| Events | `client.Events` | Event log, types, deliveries, redelivery |
| Finance | `client.Finance` | Reconciliation runs, month-end close pack, revenue recognition |
| Gateway connections | `client.GatewayConnections` | BYO Stripe/Razorpay connections and webhook secrets |
| Gifts | `client.Gifts` | Gift subscriptions |
| Imports | `client.Imports` | Stripe/RevenueCat/Chargebee preview, commit, compare; stored compare reports |
| India | `client.India` | GSTR-1 and GSTR-3B returns |
| Integration connections | `client.IntegrationConnections` | BYO tax/CRM/storage integrations, CRM sync |
| Invoices | `client.Invoices` | List/get, journal entries, payment attempts, status history, documents, send, e-invoicing (IRN and EU), payment wall |
| Ledger | `client.Ledger` | Accounts, entries, transactions, trial balance, deferred rollforward, CSV export |
| Mandates | `client.Mandates` | UPI Autopay / bank-debit mandates |
| Offline payments | `client.OfflinePayments` | Record and list offline payments |
| Organizations | `client.Organizations` | Multi-tenant organizations |
| Payment attempts | `client.PaymentAttempts` | Tenant-wide payments log |
| Plans | `client.Plans` | Catalog plans, usage charges, charge simulation |
| Quotes | `client.Quotes` | Quote-to-invoice lifecycle |
| Referrals | `client.Referrals` | Referral codes and qualification |
| Settings | `client.Settings` | GST, IRP, US sales tax (registrations, nexus, liability), EU e-invoicing, W-9, invoice branding, MCP |
| SSO | `client.SSO` | SAML SSO connection |
| Subscriptions | `client.Subscriptions` | Lifecycle, add-ons, charges, previews, usage, history, financial summary, consent, cancellation reasons |
| System | `client.System` | API build version |
| Usage | `client.Usage` | Usage events, batch ingestion, queries, dimensions |
| Usage alerts | `client.UsageAlerts` | Threshold alerts |
| Users | `client.Users` | Team members and invitations |
| Virtual accounts | `client.VirtualAccounts` | Bank virtual accounts |
| Wallets | `client.Wallets` | Prepaid wallets, top-ups, auto-recharge |
| Webhooks | `client.Webhooks` | Endpoints, status, deliveries |

Not covered, by design: browser session flows (`/auth/*`), the customer
portal (`/portal/*`), hosted checkout (`/checkout/*`), and inbound gateway
webhooks (`/webhooks/*`).

## Error handling

Any non-2xx response is returned as a `*recurso.APIError`, which decodes the
standard error envelope (`{"error": {"code", "message"}}`) and carries the HTTP
status code:

```go
_, err := client.Plans.List(ctx, nil)
var apiErr *recurso.APIError
if errors.As(err, &apiErr) {
	log.Printf("code=%s status=%d: %s", apiErr.Code, apiErr.StatusCode, apiErr.Message)
}
```

## Testing

The suite in `recurso_test.go` uses `net/http/httptest` to mock the API,
asserting request method/path/auth/body and typed response parsing across every
resource, plus the error path. Run it with:

```bash
go test ./...
```

CI (`.github/workflows/ci.yml`) runs `go build`, `go vet`, a `gofmt -l`
check, and `go test -race` on Go 1.22 and the current stable release.

## Releasing

Releases are git tags of the form `vX.Y.Z` on `main`; the Go module proxy
picks them up directly. Tags go up to `v1.6.0` today, so the next release is
`v1.7.0` (see `CHANGELOG.md`):

```bash
git tag -a v1.7.0 -m "v1.7.0"
git push origin v1.7.0
```

## License

MIT
