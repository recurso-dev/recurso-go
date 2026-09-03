# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.7.0] - 2026-09-03

Closes the coverage gap against `cmd/api/openapi.yaml`: every in-scope API
path (browser session, portal, checkout, and inbound-webhook flows excluded)
now has a typed client method. 45 resources / 276 methods, up from 22 / 90.

### Added

New resource services on `Client`:

- `Auth` — `MFASetup`, `MFAVerify`, `MFADisable`, `Sessions`,
  `RevokeOtherSessions`, `RevokeSession` (per-user security endpoints under
  `/v1/auth`; the API accepts these with dashboard session-cookie auth only).
- `Billing` — `Status`, `Plans` (the tenant's own managed-cloud plan/trial).
- `Consents` — `Record`, `Revoke`.
- `Finance` — `Reconcile`, `RecordReconciliation`, `ReconciliationRuns`,
  `ReconciliationRun`, `ClosePack`, `RevRecReport`, `RevRecWaterfall`.
- `GatewayConnections` — `List`, `Create`, `Delete`, `SetWebhookSecret`.
- `Imports` — `PreviewStripe`, `CommitStripe`, `CompareStripe`,
  `PreviewRevenueCat`, `CommitRevenueCat`, `CompareRevenueCat`,
  `PreviewChargebee`, `CommitChargebee`, `CompareChargebee`,
  `CompareReports`, `CompareReport`, `CompareReportDocument`.
- `India` — `GSTR1`, `GSTR3B` (readable return plus the GSTN upload JSON).
- `IntegrationConnections` — `List`, `Create`, `Delete`, `SyncCRM`.
- `PaymentAttempts` — `List` (with `Pagination` metadata), `Get`.
- `Settings` — `GST`, `UpdateGST`, `ValidateGSTIN`, `TaxRegistrations`,
  `SetTaxRegistrations`, `TaxNexus`, `SetTaxNexus`, `TaxLiability`,
  `TaxNexusStatus`, `IRP`, `UpdateIRP`, `TestIRP`, `EUEInvoice`,
  `UpdateEUEInvoice`, `USTax`, `UpdateUSTax`, `InvoiceBranding`,
  `UpdateInvoiceBranding`, `MCP`, `UpdateMCP`.
- `SSO` — `Get`, `Upsert`, `Delete`.
- `System` — `Version` (unauthenticated, served beside the `/v1` prefix).
- `Users` — `List`, `Create`, `Invite`, `UpdateRole`, `Delete`.

New methods on existing services:

- `Accounting.Connect` (start the OAuth flow) and `Accounting.CallbackURL`.
- `Analytics.Ask`, `DunningOverview`, `DunningWeights`, `DunningHistory`,
  `DunningRecovered`, `MRRWaterfall`, `RevenueByPlan`, `RevenueByGeography`,
  `UnitEconomics`, `UsageStats`.
- `BillableMetrics.Charges` (plans priced on a meter) and
  `Plans.SimulateCharges`; `ChargeParams` gained `FilterKey`/`Filters`.
- `Coupons.Get`, `Disputes.Get`, `Developer.RevokeKey`,
  `Customers.FinancialSummary`.
- `CreditNotes.Approve`, `Reject`, `Void`, `JournalEntries`, `DownloadPDF`.
- `Invoices.JournalEntries`, `PaymentAttempts`, `StatusHistory`,
  `DownloadPDF`, `PreviewHTML`, `Send`, `EUEInvoice`, `RetryEUEInvoice`,
  `PaymentWall`.
- `Ledger.Transaction`, `TrialBalance`, `Export` (CSV),
  `DeferredRollforward`.
- `Subscriptions.FinancialSummary`, `CancelPreview`, `History`,
  `BillUsage`, `Consent`, `CancellationReasons`.

Infrastructure:

- GitHub Actions CI (`.github/workflows/ci.yml`): build, vet, gofmt check,
  and `go test -race` on Go 1.22 and stable, on every push and pull request.
- Non-JSON endpoints (printable documents, CSV exports) return `[]byte`;
  non-2xx responses on those paths still decode into `*APIError`.

### Changed

- Every path-parameter route is now built with `fmt.Sprintf` rather than
  string concatenation, so the cross-repo drift check
  (`scripts/sdk_drift.py`) can see it. No behavioural change.

## [1.6.0] and earlier

See the git tags `v1.2.0` … `v1.6.0` for prior releases.

[Unreleased]: https://github.com/recurso-dev/recurso-go/compare/v1.7.0...HEAD
[1.7.0]: https://github.com/recurso-dev/recurso-go/compare/v1.6.0...v1.7.0
