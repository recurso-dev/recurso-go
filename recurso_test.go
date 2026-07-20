package recurso

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// testServer spins up a mock HTTP server, points a Client at it, and records
// the last request it received so tests can assert on method/path/auth/body.
type testServer struct {
	t      *testing.T
	server *httptest.Server
	client *Client

	// captured request details
	method string
	path   string
	query  string
	auth   string
	accept string
	ctype  string
	body   []byte
}

// newTestServer returns a server that replies with the given status code and
// raw JSON body for every request.
func newTestServer(t *testing.T, status int, respBody string) *testServer {
	t.Helper()
	ts := &testServer{t: t}
	ts.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts.method = r.Method
		ts.path = r.URL.Path
		ts.query = r.URL.RawQuery
		ts.auth = r.Header.Get("Authorization")
		ts.accept = r.Header.Get("Accept")
		ts.ctype = r.Header.Get("Content-Type")
		ts.body, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, respBody)
	}))
	t.Cleanup(ts.server.Close)
	ts.client = NewClient("test_key", WithBaseURL(ts.server.URL))
	return ts
}

// assertRequest checks the recorded method, path, and bearer auth header.
func (ts *testServer) assertRequest(method, path string) {
	ts.t.Helper()
	if ts.method != method {
		ts.t.Errorf("method = %q, want %q", ts.method, method)
	}
	if ts.path != path {
		ts.t.Errorf("path = %q, want %q", ts.path, path)
	}
	if ts.auth != "Bearer test_key" {
		ts.t.Errorf("Authorization = %q, want %q", ts.auth, "Bearer test_key")
	}
	if ts.accept != "application/json" {
		ts.t.Errorf("Accept = %q, want application/json", ts.accept)
	}
}

// bodyField decodes the recorded request body and returns a top-level field.
func (ts *testServer) bodyMap() map[string]any {
	ts.t.Helper()
	var m map[string]any
	if len(ts.body) == 0 {
		return m
	}
	if err := json.Unmarshal(ts.body, &m); err != nil {
		ts.t.Fatalf("request body is not a JSON object: %v (%s)", err, ts.body)
	}
	return m
}

func TestPlansCreate(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"id":"plan_1","name":"Pro","code":"PRO-USD","interval_unit":"month","interval_count":1,"active":true,"prices":[{"id":"pr_1","currency":"USD","amount":2900,"type":"recurring"}]}`)

	plan, err := ts.client.Plans.Create(context.Background(), &PlanCreateParams{
		Name: "Pro", Code: "PRO-USD", Amount: 2900, Currency: "USD", IntervalUnit: "month", IntervalCount: 1,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/plans")
	if ts.ctype != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ts.ctype)
	}
	body := ts.bodyMap()
	if body["code"] != "PRO-USD" {
		t.Errorf("body code = %v, want PRO-USD", body["code"])
	}
	if body["amount"].(float64) != 2900 {
		t.Errorf("body amount = %v, want 2900", body["amount"])
	}
	if plan.ID != "plan_1" || plan.Active != true {
		t.Errorf("plan = %+v", plan)
	}
	if len(plan.Prices) != 1 || plan.Prices[0].Amount != 2900 {
		t.Errorf("prices = %+v", plan.Prices)
	}
}

func TestPlansList(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[{"id":"plan_1","name":"Pro"},{"id":"plan_2","name":"Team"}]}`)
	plans, err := ts.client.Plans.List(context.Background(), &PlanListParams{Limit: 10, Page: 2, Q: "pro"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/plans")
	if ts.query != "limit=10&page=2&q=pro" {
		t.Errorf("query = %q", ts.query)
	}
	if len(plans) != 2 || plans[1].Name != "Team" {
		t.Errorf("plans = %+v", plans)
	}
}

func TestCustomersCreate(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"id":"cus_1","email":"jane@example.com","name":"Jane"}`)
	cus, err := ts.client.Customers.Create(context.Background(), &CustomerCreateParams{Email: "jane@example.com", Name: "Jane", Country: "US"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/customers")
	if ts.bodyMap()["country"] != "US" {
		t.Errorf("country not sent: %s", ts.body)
	}
	if cus.ID != "cus_1" {
		t.Errorf("cus = %+v", cus)
	}
}

func TestCustomersConsents(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"object":"list","data":[{"id":"con_1","consent_type":"recurring_billing","granted":true}]}`)
	consents, err := ts.client.Customers.Consents(context.Background(), "cus_1")
	if err != nil {
		t.Fatalf("Consents: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/customers/cus_1/consents")
	if len(consents) != 1 || !consents[0].Granted {
		t.Errorf("consents = %+v", consents)
	}
}

func TestSubscriptionsCreate(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"id":"sub_1","customer_id":"cus_1","plan_id":"plan_1","status":"active"}`)
	sub, err := ts.client.Subscriptions.Create(context.Background(), &SubscriptionCreateParams{CustomerID: "cus_1", PlanID: "plan_1"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/subscriptions")
	if sub.Status != "active" {
		t.Errorf("sub = %+v", sub)
	}
}

func TestSubscriptionsCancel(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"id":"sub_1","status":"canceled","cancel_at_period_end":true,"message":"ok"}`)
	immediately := false
	res, err := ts.client.Subscriptions.Cancel(context.Background(), "sub_1", &SubscriptionCancelParams{Reason: "too_expensive", Immediately: &immediately})
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/subscriptions/sub_1/cancel")
	if ts.bodyMap()["reason"] != "too_expensive" {
		t.Errorf("reason not sent: %s", ts.body)
	}
	if res.Status != "canceled" || !res.CancelAtPeriodEnd {
		t.Errorf("res = %+v", res)
	}
}

func TestSubscriptionsPause(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":{"id":"sub_1","status":"paused"}}`)
	sub, err := ts.client.Subscriptions.Pause(context.Background(), "sub_1")
	if err != nil {
		t.Fatalf("Pause: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/subscriptions/sub_1/pause")
	if sub.Status != "paused" {
		t.Errorf("sub = %+v", sub)
	}
}

func TestSubscriptionsPreviewChange(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"subscription_id":"sub_1","new_plan_id":"plan_2","total_amount":1500,"is_upgrade":true}`)
	preview, err := ts.client.Subscriptions.PreviewChange(context.Background(), "sub_1", "plan_2")
	if err != nil {
		t.Fatalf("PreviewChange: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/subscriptions/sub_1/preview-change")
	if ts.query != "plan_id=plan_2" {
		t.Errorf("query = %q", ts.query)
	}
	if !preview.IsUpgrade || preview.TotalAmount != 1500 {
		t.Errorf("preview = %+v", preview)
	}
}

func TestSubscriptionsAddAddon(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"id":"addon_1","subscription_id":"sub_1","plan_id":"plan_2","quantity":3}`)
	addon, err := ts.client.Subscriptions.AddAddon(context.Background(), "sub_1", &AddonCreateParams{PlanID: "plan_2", Quantity: 3})
	if err != nil {
		t.Fatalf("AddAddon: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/subscriptions/sub_1/addons")
	if addon.Quantity != 3 {
		t.Errorf("addon = %+v", addon)
	}
}

func TestSubscriptionsRemoveAddon(t *testing.T) {
	ts := newTestServer(t, http.StatusNoContent, ``)
	if err := ts.client.Subscriptions.RemoveAddon(context.Background(), "sub_1", "addon_1"); err != nil {
		t.Fatalf("RemoveAddon: %v", err)
	}
	ts.assertRequest(http.MethodDelete, "/subscriptions/sub_1/addons/addon_1")
}

func TestInvoicesList(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[{"id":"inv_1","total":11800,"currency":"INR"}]}`)
	invoices, err := ts.client.Invoices.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/invoices")
	if len(invoices) != 1 || invoices[0].Total != 11800 {
		t.Errorf("invoices = %+v", invoices)
	}
}

func TestInvoicesPDFURL(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, ``)
	got := ts.client.Invoices.PDFURL("inv_1")
	want := ts.server.URL + "/invoices/inv_1/pdf"
	if got != want {
		t.Errorf("PDFURL = %q, want %q", got, want)
	}
}

func TestCouponsCreate(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"id":"cpn_1","code":"SAVE20","discount_type":"percent","discount_value":20,"duration":"once"}`)
	coupon, err := ts.client.Coupons.Create(context.Background(), &CouponCreateParams{Code: "SAVE20", DiscountType: "percent", DiscountValue: 20, Duration: "once"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/coupons")
	if coupon.Code != "SAVE20" {
		t.Errorf("coupon = %+v", coupon)
	}
}

func TestUsageRecord(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"status":"recorded","event_id":"evt_1"}`)
	res, err := ts.client.Usage.Record(context.Background(), &UsageRecordParams{SubscriptionID: "sub_1", CustomerID: "cus_1", Dimension: "api_calls", Quantity: 42})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/usage/events")
	if res.EventID != "evt_1" || res.Status != "recorded" {
		t.Errorf("res = %+v", res)
	}
}

func TestUsageQuery(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[{"dimension":"api_calls","quantity":100}],"granularity":"day"}`)
	res, err := ts.client.Usage.Query(context.Background(), &UsageQueryParams{SubscriptionID: "sub_1", Granularity: "day"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/usage")
	if ts.query != "granularity=day&subscription_id=sub_1" {
		t.Errorf("query = %q", ts.query)
	}
	if len(res.Data) != 1 || res.Data[0].Quantity != 100 {
		t.Errorf("res = %+v", res)
	}
}

func TestCreditNotesCreate(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"data":{"id":"cn_1","amount":500,"currency":"USD","status":"issued"}}`)
	cn, err := ts.client.CreditNotes.Create(context.Background(), &CreditNoteCreateParams{CustomerID: "cus_1", Amount: 500, Currency: "USD"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/credit-notes")
	if cn.Amount != 500 || cn.Status != "issued" {
		t.Errorf("cn = %+v", cn)
	}
}

func TestQuotesLifecycle(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":{"id":"q_1","status":"sent"},"message":"sent"}`)
	quote, err := ts.client.Quotes.Send(context.Background(), "q_1")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/quotes/q_1/send")
	if quote.Status != "sent" {
		t.Errorf("quote = %+v", quote)
	}
}

func TestQuotesCreate(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"data":{"id":"q_1","status":"draft","total":10000}}`)
	quote, err := ts.client.Quotes.Create(context.Background(), &QuoteCreateParams{
		CustomerID: "cus_1",
		LineItems:  []LineItem{{Description: "Setup", Quantity: 1, UnitPrice: 10000, Amount: 10000}},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/quotes")
	if quote.Total != 10000 {
		t.Errorf("quote = %+v", quote)
	}
}

func TestWebhooksCreate(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"data":{"id":"wh_1","url":"https://x.test/hook","secret":"whsec_abc","events":["invoice.paid"]}}`)
	wh, err := ts.client.Webhooks.Create(context.Background(), &WebhookCreateParams{URL: "https://x.test/hook", Events: []string{"invoice.paid"}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/webhooks")
	if wh.Secret != "whsec_abc" || len(wh.Events) != 1 {
		t.Errorf("wh = %+v", wh)
	}
}

func TestWebhooksDeliveries(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[{"id":"del_1","status":"succeeded","attempts":1}]}`)
	dels, err := ts.client.Webhooks.Deliveries(context.Background(), "wh_1", &DeliveryListParams{Limit: 5, Status: "succeeded"})
	if err != nil {
		t.Fatalf("Deliveries: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/webhooks/wh_1/deliveries")
	if ts.query != "limit=5&status=succeeded" {
		t.Errorf("query = %q", ts.query)
	}
	if len(dels) != 1 || dels[0].Status != "succeeded" {
		t.Errorf("dels = %+v", dels)
	}
}

func TestEventsRedeliver(t *testing.T) {
	ts := newTestServer(t, http.StatusAccepted, `{"data":{"event_id":"evt_1","deliveries_queued":2}}`)
	res, err := ts.client.Events.Redeliver(context.Background(), "evt_1")
	if err != nil {
		t.Fatalf("Redeliver: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/events/evt_1/redeliver")
	if res.DeliveriesQueued != 2 {
		t.Errorf("res = %+v", res)
	}
}

func TestEventsTypes(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":["invoice.paid","subscription.created"]}`)
	types, err := ts.client.Events.Types(context.Background())
	if err != nil {
		t.Fatalf("Types: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/events/types")
	if len(types) != 2 || types[0] != "invoice.paid" {
		t.Errorf("types = %+v", types)
	}
}

func TestEntitlementsCheck(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"feature_key":"seats","granted":true,"limit_value":25}`)
	check, err := ts.client.Entitlements.Check(context.Background(), "cus_1", "seats")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/entitlements/check")
	if ts.query != "customer_id=cus_1&feature=seats" {
		t.Errorf("query = %q", ts.query)
	}
	if !check.Granted || check.LimitValue == nil || *check.LimitValue != 25 {
		t.Errorf("check = %+v", check)
	}
}

func TestEntitlementsSetForPlan(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[{"id":"ent_1","feature_key":"seats","kind":"limit","limit_value":10}]}`)
	limit := int64(10)
	ents, err := ts.client.Entitlements.SetForPlan(context.Background(), "plan_1", []EntitlementInput{{FeatureKey: "seats", Kind: "limit", LimitValue: &limit}})
	if err != nil {
		t.Fatalf("SetForPlan: %v", err)
	}
	ts.assertRequest(http.MethodPut, "/plans/plan_1/entitlements")
	// Body should be a bare JSON array.
	var arr []map[string]any
	if err := json.Unmarshal(ts.body, &arr); err != nil {
		t.Fatalf("body is not a JSON array: %v (%s)", err, ts.body)
	}
	if len(arr) != 1 || arr[0]["feature_key"] != "seats" {
		t.Errorf("body = %s", ts.body)
	}
	if len(ents) != 1 || ents[0].LimitValue == nil || *ents[0].LimitValue != 10 {
		t.Errorf("ents = %+v", ents)
	}
}

func TestAnalyticsMRR(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"currency":"USD","mrr":250000,"normalized_mrr":250000,"reporting_currency":"USD","fx":{"source":"live"}}`)
	mrr, err := ts.client.Analytics.MRR(context.Background())
	if err != nil {
		t.Fatalf("MRR: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/analytics/mrr")
	if mrr.MRR != 250000 || mrr.FX.Source != "live" {
		t.Errorf("mrr = %+v", mrr)
	}
}

func TestDeveloperCreateKey(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"id":"key_1","key_value":"rsk_live_secret","key_prefix":"rsk_live","is_active":true}`)
	key, err := ts.client.Developer.CreateKey(context.Background())
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/developer/keys")
	if key.KeyValue != "rsk_live_secret" {
		t.Errorf("key = %+v", key)
	}
}

func TestAccountGet(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":{"id":"ten_1","name":"Acme","email":"ops@acme.test","base_currency":"USD"}}`)
	acct, err := ts.client.Account.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/account")
	if acct.Name != "Acme" || acct.BaseCurrency != "USD" {
		t.Errorf("acct = %+v", acct)
	}
}

func TestReferralsGenerateCode(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":{"code":"REF-ABC123"}}`)
	code, err := ts.client.Referrals.GenerateCode(context.Background(), "cus_1")
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/referrals/generate-code")
	if ts.bodyMap()["customer_id"] != "cus_1" {
		t.Errorf("customer_id not sent: %s", ts.body)
	}
	if code.Code != "REF-ABC123" {
		t.Errorf("code = %+v", code)
	}
}

func TestGiftsPurchase(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"id":"gift_1","code":"GIFT-XYZ","status":"purchased","duration_months":3}`)
	gift, err := ts.client.Gifts.Purchase(context.Background(), &GiftPurchaseParams{BuyerCustomerID: "cus_1", PlanID: "plan_1", DurationMonths: 3})
	if err != nil {
		t.Fatalf("Purchase: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/gifts/purchase")
	if gift.Code != "GIFT-XYZ" || gift.DurationMonths != 3 {
		t.Errorf("gift = %+v", gift)
	}
}

func TestMandatesCreate(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"mandate":{"id":"mnd_1","status":"created","vpa":"jane@bank"},"auth_url":"https://pay.test/authorize/mnd_1"}`)
	res, err := ts.client.Mandates.Create(context.Background(), &MandateCreateParams{CustomerID: "cus_1", VPA: "jane@bank", MaxAmount: 100000, Frequency: "monthly"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/mandates")
	if res.Mandate.ID != "mnd_1" || res.AuthURL == "" {
		t.Errorf("res = %+v", res)
	}
}

func TestLedgerAccounts(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[{"id":"acc_1","name":"Accounts Receivable","type":"asset","balance":50000}]}`)
	accounts, err := ts.client.Ledger.Accounts(context.Background())
	if err != nil {
		t.Fatalf("Accounts: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/ledger/accounts")
	if len(accounts) != 1 || accounts[0].Balance != 50000 {
		t.Errorf("accounts = %+v", accounts)
	}
}

// TestErrorEnvelope verifies that a 4xx response decodes into a populated
// *APIError.
func TestErrorEnvelope(t *testing.T) {
	ts := newTestServer(t, http.StatusNotFound, `{"error":{"code":"NOT_FOUND","message":"plan not found"}}`)
	_, err := ts.client.Plans.List(context.Background(), nil)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
	if apiErr.Code != "NOT_FOUND" || apiErr.Message != "plan not found" {
		t.Errorf("apiErr = %+v", apiErr)
	}
	if apiErr.Error() == "" {
		t.Error("Error() returned empty string")
	}
}

// TestErrorNonJSON verifies a non-JSON error body still yields a *APIError.
func TestErrorNonJSON(t *testing.T) {
	ts := newTestServer(t, http.StatusInternalServerError, `upstream failure`)
	_, err := ts.client.Customers.List(context.Background(), nil)
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != 500 || apiErr.Message != "upstream failure" {
		t.Errorf("apiErr = %+v", apiErr)
	}
}

// --- Usage-based billing v1 (metering) ---

func TestBillableMetricsCreate(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"data":{"id":"bm_1","name":"API calls","code":"api_calls","aggregation_type":"sum"}}`)
	m, err := ts.client.BillableMetrics.Create(context.Background(), &BillableMetricParams{Name: "API calls", Code: "api_calls", AggregationType: "sum"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/billable-metrics")
	if m.ID != "bm_1" || m.Code != "api_calls" || m.AggregationType != "sum" {
		t.Errorf("m = %+v", m)
	}
}

func TestBillableMetricsList(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[{"id":"bm_1","code":"api_calls"},{"id":"bm_2","code":"active_users"}]}`)
	list, err := ts.client.BillableMetrics.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/billable-metrics")
	if len(list) != 2 || list[1].Code != "active_users" {
		t.Errorf("list = %+v", list)
	}
}

func TestBillableMetricsGetUpdateDelete(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":{"id":"bm_1","code":"api_calls","aggregation_type":"max"}}`)
	if _, err := ts.client.BillableMetrics.Get(context.Background(), "bm_1"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/billable-metrics/bm_1")

	if _, err := ts.client.BillableMetrics.Update(context.Background(), "bm_1", &BillableMetricParams{Name: "API calls", Code: "api_calls", AggregationType: "max"}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	ts.assertRequest(http.MethodPut, "/billable-metrics/bm_1")

	if err := ts.client.BillableMetrics.Delete(context.Background(), "bm_1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	ts.assertRequest(http.MethodDelete, "/billable-metrics/bm_1")
}

func TestPlansSetCharges(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[{"id":"chg_1","plan_id":"plan_1","metric_id":"bm_1","charge_model":"per_unit","amounts":{"INR":{"unit_amount":"0.0035"}}}]}`)
	up := int64(1000)
	charges, err := ts.client.Plans.SetCharges(context.Background(), "plan_1", []ChargeParams{
		{MetricID: "bm_1", ChargeModel: "graduated", Amounts: map[string]ChargeAmounts{
			"INR": {Tiers: []ChargeTier{{UpTo: &up, UnitAmount: "0.10"}, {UpTo: nil, UnitAmount: "0.05"}}},
		}},
	})
	if err != nil {
		t.Fatalf("SetCharges: %v", err)
	}
	ts.assertRequest(http.MethodPut, "/plans/plan_1/charges")
	if len(charges) != 1 || charges[0].ChargeModel != "per_unit" || charges[0].Amounts["INR"].UnitAmount != "0.0035" {
		t.Errorf("charges = %+v", charges)
	}
}

func TestPlansGetCharges(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[{"id":"chg_1","metric":{"code":"api_calls","aggregation_type":"sum"}}]}`)
	charges, err := ts.client.Plans.GetCharges(context.Background(), "plan_1")
	if err != nil {
		t.Fatalf("GetCharges: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/plans/plan_1/charges")
	if len(charges) != 1 || charges[0].Metric == nil || charges[0].Metric.Code != "api_calls" {
		t.Errorf("charges = %+v", charges)
	}
}

func TestSubscriptionsUsageAmount(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":{"subscription_id":"sub_1","currency":"INR","charges":[{"metric_code":"api_calls","quantity":45231,"amount":176155}],"total_amount":176155}}`)
	ua, err := ts.client.Subscriptions.UsageAmount(context.Background(), "sub_1")
	if err != nil {
		t.Fatalf("UsageAmount: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/subscriptions/sub_1/usage-amount")
	if ua.TotalAmount != 176155 || len(ua.Charges) != 1 || ua.Charges[0].Quantity != 45231 {
		t.Errorf("ua = %+v", ua)
	}
}

// --- Wallets, commitments, alerts, batch, audit (Lago-parity B/C) ---

func TestWalletsCreateAndTopUp(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"data":{"id":"wal_1","customer_id":"cus_1","currency":"INR","balance":0}}`)
	w, err := ts.client.Wallets.Create(context.Background(), &WalletCreateParams{CustomerID: "cus_1", Currency: "INR"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/wallets")
	if w.ID != "wal_1" || w.Currency != "INR" {
		t.Errorf("w = %+v", w)
	}

	ts2 := newTestServer(t, http.StatusCreated, `{"data":{"id":"wtx_1","wallet_id":"wal_1","type":"top_up","amount":500000,"balance_after":500000}}`)
	wtx, err := ts2.client.Wallets.TopUp(context.Background(), "wal_1", &WalletTopUpParams{Amount: 500000, Source: "manual"})
	if err != nil {
		t.Fatalf("TopUp: %v", err)
	}
	ts2.assertRequest(http.MethodPost, "/wallets/wal_1/top-up")
	if wtx.BalanceAfter != 500000 || wtx.Type != "top_up" {
		t.Errorf("wtx = %+v", wtx)
	}
}

func TestWalletsReadsAndAutoRecharge(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[{"id":"wal_1","currency":"INR","balance":100}]}`)
	list, err := ts.client.Wallets.ForCustomer(context.Background(), "cus_1")
	if err != nil || len(list) != 1 {
		t.Fatalf("ForCustomer: %v / %+v", err, list)
	}
	ts.assertRequest(http.MethodGet, "/customers/cus_1/wallets")

	ts2 := newTestServer(t, http.StatusOK, `{"data":[{"id":"wtx_1","type":"drain","amount":100,"balance_after":0}]}`)
	txs, err := ts2.client.Wallets.Transactions(context.Background(), "wal_1")
	if err != nil || len(txs) != 1 || txs[0].Type != "drain" {
		t.Fatalf("Transactions: %v / %+v", err, txs)
	}
	ts2.assertRequest(http.MethodGet, "/wallets/wal_1/transactions")

	ts3 := newTestServer(t, http.StatusOK, `{"data":{"id":"wal_1","auto_recharge_threshold":100000,"auto_recharge_amount":500000}}`)
	th, amt := int64(100000), int64(500000)
	w, err := ts3.client.Wallets.SetAutoRecharge(context.Background(), "wal_1", &WalletAutoRechargeParams{AutoRechargeThreshold: &th, AutoRechargeAmount: &amt})
	if err != nil || w.AutoRechargeThreshold == nil || *w.AutoRechargeThreshold != 100000 {
		t.Fatalf("SetAutoRecharge: %v / %+v", err, w)
	}
	ts3.assertRequest(http.MethodPut, "/wallets/wal_1/auto-recharge")
}

func TestSubscriptionsSetCommitment(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":{"id":"sub_1","commitment_amount":5000000}}`)
	sub, err := ts.client.Subscriptions.SetCommitment(context.Background(), "sub_1", 5000000)
	if err != nil {
		t.Fatalf("SetCommitment: %v", err)
	}
	ts.assertRequest(http.MethodPut, "/subscriptions/sub_1/commitment")
	if got := ts.bodyMap()["amount"]; got != float64(5000000) {
		t.Errorf("body amount = %v, want 5000000", got)
	}
	_ = sub
}

func TestUsageAlertsLifecycle(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"data":{"id":"ua_1","metric_code":"api_calls","threshold_type":"quantity","threshold":1000000}}`)
	a, err := ts.client.UsageAlerts.Create(context.Background(), &UsageAlertCreateParams{
		SubscriptionID: "sub_1", MetricCode: "api_calls", ThresholdType: "quantity", Threshold: 1000000,
	})
	if err != nil || a.Threshold != 1000000 {
		t.Fatalf("Create: %v / %+v", err, a)
	}
	ts.assertRequest(http.MethodPost, "/usage-alerts")

	ts2 := newTestServer(t, http.StatusOK, `{"data":[{"id":"ua_1"}]}`)
	list, err := ts2.client.UsageAlerts.List(context.Background(), "sub_1")
	if err != nil || len(list) != 1 {
		t.Fatalf("List: %v / %+v", err, list)
	}
	ts2.assertRequest(http.MethodGet, "/usage-alerts")
	if ts2.query != "subscription_id=sub_1" {
		t.Errorf("query = %q", ts2.query)
	}

	ts3 := newTestServer(t, http.StatusNoContent, ``)
	if err := ts3.client.UsageAlerts.Delete(context.Background(), "ua_1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	ts3.assertRequest(http.MethodDelete, "/usage-alerts/ua_1")
}

func TestUsageRecordBatch(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[{"index":0,"status":"recorded","event_id":"evt_1"},{"index":1,"status":"duplicate","event_id":"evt_0"}]}`)
	results, err := ts.client.Usage.RecordBatch(context.Background(), []UsageRecordParams{
		{SubscriptionID: "sub_1", CustomerID: "cus_1", Dimension: "api_calls", Quantity: 10, TransactionID: "t-1"},
		{SubscriptionID: "sub_1", CustomerID: "cus_1", Dimension: "api_calls", Quantity: 10, TransactionID: "t-0"},
	})
	if err != nil {
		t.Fatalf("RecordBatch: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/usage/events/batch")
	if len(results) != 2 || results[1].Status != "duplicate" {
		t.Errorf("results = %+v", results)
	}
}

func TestAuditLogsList(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[{"id":"al_1","actor":"api_key","action":"PUT /v1/plans/:id/charges","entity_type":"plans","status":200}]}`)
	logs, err := ts.client.AuditLogs.List(context.Background(), &AuditLogListParams{EntityType: "plans", Limit: 50})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/audit-logs")
	if ts.query != "entity_type=plans&limit=50" {
		t.Errorf("query = %q", ts.query)
	}
	if len(logs) != 1 || logs[0].EntityType != "plans" {
		t.Errorf("logs = %+v", logs)
	}
}

func TestPlansGetUpdate(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":{"id":"plan_1","name":"Pro","active":true}}`)
	plan, err := ts.client.Plans.Get(context.Background(), "plan_1")
	if err != nil || plan.Name != "Pro" {
		t.Fatalf("Get: %v / %+v", err, plan)
	}
	ts.assertRequest(http.MethodGet, "/plans/plan_1")

	ts2 := newTestServer(t, http.StatusOK, `{"id":"plan_1","name":"Pro","active":false}`)
	archived := false
	plan, err = ts2.client.Plans.Update(context.Background(), "plan_1", &PlanUpdateParams{Active: &archived})
	if err != nil || plan.Active {
		t.Fatalf("Update: %v / %+v", err, plan)
	}
	ts2.assertRequest(http.MethodPut, "/plans/plan_1")
	if ts2.bodyMap()["active"] != false {
		t.Errorf("body = %s", ts2.body)
	}
}

func TestCustomersGetUpdate(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":{"id":"cus_1","email":"jane@example.com"}}`)
	cus, err := ts.client.Customers.Get(context.Background(), "cus_1")
	if err != nil || cus.Email != "jane@example.com" {
		t.Fatalf("Get: %v / %+v", err, cus)
	}
	ts.assertRequest(http.MethodGet, "/customers/cus_1")

	ts2 := newTestServer(t, http.StatusOK, `{"data":{"id":"cus_1","name":"Jane Q. User"}}`)
	cus, err = ts2.client.Customers.Update(context.Background(), "cus_1", &CustomerUpdateParams{Name: "Jane Q. User"})
	if err != nil || cus.Name != "Jane Q. User" {
		t.Fatalf("Update: %v / %+v", err, cus)
	}
	ts2.assertRequest(http.MethodPut, "/customers/cus_1")
	body := ts2.bodyMap()
	if body["name"] != "Jane Q. User" {
		t.Errorf("body = %s", ts2.body)
	}
	if _, ok := body["active"]; ok {
		t.Errorf("omitted active was sent: %s", ts2.body)
	}
}

func TestCouponsUpdate(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"status":"deactivated"}`)
	res, err := ts.client.Coupons.Update(context.Background(), "cpn_1", &CouponUpdateParams{Active: false})
	if err != nil || res.Status != "deactivated" {
		t.Fatalf("Update: %v / %+v", err, res)
	}
	ts.assertRequest(http.MethodPut, "/coupons/cpn_1")
	if ts.bodyMap()["active"] != false {
		t.Errorf("body = %s", ts.body)
	}
}

func TestOrganizationsLifecycle(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"id":"org_1","name":"Acme Group","owner_email":"ceo@acme.com"}`)
	org, err := ts.client.Organizations.Create(context.Background(), &OrganizationCreateParams{Name: "Acme Group", OwnerEmail: "ceo@acme.com"})
	if err != nil || org.ID != "org_1" {
		t.Fatalf("Create: %v / %+v", err, org)
	}
	ts.assertRequest(http.MethodPost, "/organizations")

	ts2 := newTestServer(t, http.StatusOK, `{"data":{"id":"org_1","name":"Acme Holdings"}}`)
	org, err = ts2.client.Organizations.Update(context.Background(), "org_1", &OrganizationUpdateParams{Name: "Acme Holdings"})
	if err != nil || org.Name != "Acme Holdings" {
		t.Fatalf("Update: %v / %+v", err, org)
	}
	ts2.assertRequest(http.MethodPut, "/organizations/org_1")

	ts3 := newTestServer(t, http.StatusOK, `{"status":"added"}`)
	res, err := ts3.client.Organizations.AddTenant(context.Background(), "org_1", "ten_1")
	if err != nil || res.Status != "added" {
		t.Fatalf("AddTenant: %v / %+v", err, res)
	}
	ts3.assertRequest(http.MethodPost, "/organizations/org_1/tenants")
	if ts3.bodyMap()["tenant_id"] != "ten_1" {
		t.Errorf("body = %s", ts3.body)
	}

	ts4 := newTestServer(t, http.StatusOK, `{"data":{"normalized_mrr":250000,"reporting_currency":"USD","by_currency":[{"currency":"INR","total_mrr":9900000,"by_tenant":[{"tenant_id":"ten_1","mrr":9900000}]}]}}`)
	mrr, err := ts4.client.Organizations.MRR(context.Background(), "org_1")
	if err != nil || mrr.NormalizedMRR != 250000 || len(mrr.ByCurrency) != 1 {
		t.Fatalf("MRR: %v / %+v", err, mrr)
	}
	ts4.assertRequest(http.MethodGet, "/organizations/org_1/analytics/mrr")
}

func TestAccountingConnectToken(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"data":{"id":"acc_1","provider":"netsuite","realm_id":"ACME123","sync_status":"idle","is_active":true}}`)
	conn, err := ts.client.Accounting.ConnectToken(context.Background(), "netsuite", &AccountingConnectTokenParams{AccountID: "ACME123", AccessToken: "tok"})
	if err != nil || conn.Provider != "netsuite" || !conn.IsActive {
		t.Fatalf("ConnectToken: %v / %+v", err, conn)
	}
	ts.assertRequest(http.MethodPost, "/accounting/connect-token/netsuite")
	if ts.bodyMap()["account_id"] != "ACME123" {
		t.Errorf("body = %s", ts.body)
	}

	ts2 := newTestServer(t, http.StatusOK, `{"status":"sync_triggered"}`)
	res, err := ts2.client.Accounting.Sync(context.Background())
	if err != nil || res.Status != "sync_triggered" {
		t.Fatalf("Sync: %v / %+v", err, res)
	}
	ts2.assertRequest(http.MethodPost, "/accounting/sync")

	ts3 := newTestServer(t, http.StatusOK, `{"data":[{"id":"log_1","entity_type":"invoice","status":"synced"}]}`)
	logs, err := ts3.client.Accounting.SyncStatus(context.Background())
	if err != nil || len(logs) != 1 || logs[0].EntityType != "invoice" {
		t.Fatalf("SyncStatus: %v / %+v", err, logs)
	}
	ts3.assertRequest(http.MethodGet, "/accounting/sync/status")
}

func TestVirtualAccountsCreate(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"id":"va_1","customer_id":"cus_1","account_number":"2223330001","ifsc_code":"RAZR0000001","amount_expected":590000}`)
	va, err := ts.client.VirtualAccounts.Create(context.Background(), &VirtualAccountCreateParams{CustomerID: "cus_1", InvoiceID: "inv_1", Amount: 590000})
	if err != nil || va.AccountNumber != "2223330001" {
		t.Fatalf("Create: %v / %+v", err, va)
	}
	ts.assertRequest(http.MethodPost, "/virtual-accounts")
	if ts.bodyMap()["invoice_id"] != "inv_1" {
		t.Errorf("body = %s", ts.body)
	}
}

func TestOfflinePaymentsRecord(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"id":"op_1","payment_type":"bank_transfer","amount":580000,"tds_amount":10000,"currency":"INR"}`)
	p, err := ts.client.OfflinePayments.Record(context.Background(), &OfflinePaymentRecordParams{
		CustomerID: "cus_1", InvoiceID: "inv_1", PaymentType: "bank_transfer", Amount: 580000, TDSAmount: 10000, ReferenceNumber: "NEFT-1",
	})
	if err != nil || p.TDSAmount != 10000 {
		t.Fatalf("Record: %v / %+v", err, p)
	}
	ts.assertRequest(http.MethodPost, "/payments/offline")
	if ts.bodyMap()["reference_number"] != "NEFT-1" {
		t.Errorf("body = %s", ts.body)
	}
}

func TestChurnHighRiskAndAlerts(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[{"customer_id":"cus_1","score":85,"risk_level":"high"}]}`)
	scores, err := ts.client.Churn.HighRisk(context.Background(), 80)
	if err != nil || len(scores) != 1 || scores[0].Score != 85 {
		t.Fatalf("HighRisk: %v / %+v", err, scores)
	}
	ts.assertRequest(http.MethodGet, "/churn/high-risk")
	if ts.query != "threshold=80" {
		t.Errorf("query = %q", ts.query)
	}

	ts2 := newTestServer(t, http.StatusOK, `{"data":[{"id":"ca_1","customer_id":"cus_1","new_score":85,"acknowledged":false}]}`)
	alerts, err := ts2.client.Churn.Alerts(context.Background())
	if err != nil || len(alerts) != 1 || alerts[0].NewScore != 85 {
		t.Fatalf("Alerts: %v / %+v", err, alerts)
	}
	ts2.assertRequest(http.MethodGet, "/churn/alerts")

	ts3 := newTestServer(t, http.StatusOK, `{"status":"acknowledged"}`)
	res, err := ts3.client.Churn.AcknowledgeAlert(context.Background(), "ca_1")
	if err != nil || res.Status != "acknowledged" {
		t.Fatalf("AcknowledgeAlert: %v / %+v", err, res)
	}
	ts3.assertRequest(http.MethodPost, "/churn/alerts/ca_1/ack")
}

func TestCancelFlowsLifecycle(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"id":"cf_1","name":"Default","is_default":true,"cooldown_days":30}`)
	flow, err := ts.client.CancelFlows.Create(context.Background(), &CancelFlowCreateParams{Name: "Default", IsDefault: true})
	if err != nil || flow.ID != "cf_1" {
		t.Fatalf("Create: %v / %+v", err, flow)
	}
	ts.assertRequest(http.MethodPost, "/cancel-flows")

	ts2 := newTestServer(t, http.StatusOK, `[{"id":"cf_1","name":"Default"}]`)
	flows, err := ts2.client.CancelFlows.List(context.Background())
	if err != nil || len(flows) != 1 {
		t.Fatalf("List: %v / %+v", err, flows)
	}
	ts2.assertRequest(http.MethodGet, "/cancel-flows")

	ts3 := newTestServer(t, http.StatusCreated, `{"id":"cfs_1","flow_id":"cf_1","step_order":1,"step_type":"survey","config":{"reasons":["too_expensive"]}}`)
	step, err := ts3.client.CancelFlows.AddStep(context.Background(), "cf_1", &CancelFlowStepCreateParams{
		StepOrder: 1, StepType: "survey", Config: json.RawMessage(`{"reasons":["too_expensive"]}`),
	})
	if err != nil || step.StepType != "survey" {
		t.Fatalf("AddStep: %v / %+v", err, step)
	}
	ts3.assertRequest(http.MethodPost, "/cancel-flows/cf_1/steps")

	ts4 := newTestServer(t, http.StatusCreated, `{"session_id":"sess_1","flow_id":"cf_1","first_step":{"id":"cfs_1","step_type":"survey"}}`)
	start, err := ts4.client.CancelFlows.StartSession(context.Background(), &CancelFlowSessionStartParams{CustomerID: "cus_1", SubscriptionID: "sub_1"})
	if err != nil || start.SessionID != "sess_1" || start.FirstStep == nil {
		t.Fatalf("StartSession: %v / %+v", err, start)
	}
	ts4.assertRequest(http.MethodPost, "/cancel-flows/sessions/start")

	ts5 := newTestServer(t, http.StatusOK, `{"session_id":"sess_1","status":"saved","saved_by_offer":true,"completed":true}`)
	sub, err := ts5.client.CancelFlows.SubmitStep(context.Background(), "sess_1", &CancelFlowSubmitParams{StepIndex: 1, Response: map[string]any{"accepted": true}})
	if err != nil || !sub.SavedByOffer {
		t.Fatalf("SubmitStep: %v / %+v", err, sub)
	}
	ts5.assertRequest(http.MethodPost, "/cancel-flows/sessions/sess_1/submit")

	ts6 := newTestServer(t, http.StatusOK, `{"total_sessions":10,"saved_count":4,"save_rate":0.4,"reason_breakdown":{"too_expensive":6}}`)
	stats, err := ts6.client.CancelFlows.Stats(context.Background(), "cf_1")
	if err != nil || stats.SavedCount != 4 || stats.ReasonBreakdown["too_expensive"] != 6 {
		t.Fatalf("Stats: %v / %+v", err, stats)
	}
	ts6.assertRequest(http.MethodGet, "/cancel-flows/stats")
	if ts6.query != "flow_id=cf_1" {
		t.Errorf("query = %q", ts6.query)
	}
}

func TestDunningCampaignsLifecycle(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"id":"dc_1","name":"Failed payments","trigger_event":"payment_failed"}`)
	c, err := ts.client.DunningCampaigns.Create(context.Background(), &DunningCampaignCreateParams{Name: "Failed payments", TriggerEvent: "payment_failed"})
	if err != nil || c.TriggerEvent != "payment_failed" {
		t.Fatalf("Create: %v / %+v", err, c)
	}
	ts.assertRequest(http.MethodPost, "/dunning-campaigns")

	ts2 := newTestServer(t, http.StatusOK, `[{"id":"dc_1","name":"Failed payments","is_active":true}]`)
	list, err := ts2.client.DunningCampaigns.List(context.Background())
	if err != nil || len(list) != 1 {
		t.Fatalf("List: %v / %+v", err, list)
	}
	ts2.assertRequest(http.MethodGet, "/dunning-campaigns")

	ts3 := newTestServer(t, http.StatusCreated, `{"id":"dcs_1","campaign_id":"dc_1","step_order":1,"channel":"email","delay_hours":24,"is_payment_wall":false}`)
	step, err := ts3.client.DunningCampaigns.AddStep(context.Background(), "dc_1", &DunningStepCreateParams{StepOrder: 1, Channel: "email", DelayHours: 24, Subject: "Payment failed"})
	if err != nil || step.Channel != "email" {
		t.Fatalf("AddStep: %v / %+v", err, step)
	}
	ts3.assertRequest(http.MethodPost, "/dunning-campaigns/dc_1/steps")

	ts4 := newTestServer(t, http.StatusOK, `{"status":"deleted"}`)
	res, err := ts4.client.DunningCampaigns.DeleteStep(context.Background(), "dcs_1")
	if err != nil || res.Status != "deleted" {
		t.Fatalf("DeleteStep: %v / %+v", err, res)
	}
	ts4.assertRequest(http.MethodDelete, "/dunning-campaigns/steps/dcs_1")
}

func TestWebhooksUpdateStatus(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"status":"inactive"}`)
	res, err := ts.client.Webhooks.UpdateStatus(context.Background(), "wh_1", &WebhookStatusParams{Status: "inactive"})
	if err != nil || res.Status != "inactive" {
		t.Fatalf("UpdateStatus: %v / %+v", err, res)
	}
	ts.assertRequest(http.MethodPut, "/webhooks/wh_1/status")
	if ts.bodyMap()["status"] != "inactive" {
		t.Errorf("body = %s", ts.body)
	}
}

func TestUsageListEvents(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[{"id":"ue_1","subscription_id":"sub_1","customer_id":"cus_1","dimension":"api_calls","quantity":10,"transaction_id":"t-1"}]}`)
	events, err := ts.client.Usage.ListEvents(context.Background(), &UsageEventListParams{CustomerID: "cus_1", Dimension: "api_calls", Limit: 100, Offset: 50})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/usage/events")
	if ts.query != "customer_id=cus_1&dimension=api_calls&limit=100&offset=50" {
		t.Errorf("query = %q", ts.query)
	}
	if len(events) != 1 || events[0].Dimension != "api_calls" || events[0].Quantity != 10 {
		t.Errorf("events = %+v", events)
	}
}
