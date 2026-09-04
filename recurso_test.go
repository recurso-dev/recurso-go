package recurso

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

func TestPlansListCurrencyIntervalFilters(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[]}`)
	_, err := ts.client.Plans.List(context.Background(), &PlanListParams{Currency: "INR", IntervalUnit: "month"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if ts.query != "currency=INR&interval_unit=month" {
		t.Errorf("query = %q", ts.query)
	}
}

func TestSubscriptionsListPlanAndDateFilters(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[]}`)
	after := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	_, err := ts.client.Subscriptions.List(context.Background(), &SubscriptionListParams{
		PlanID:       "plan_1",
		StartedAfter: after,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if ts.query != "plan_id=plan_1&started_after=2026-08-01T00%3A00%3A00Z" {
		t.Errorf("query = %q", ts.query)
	}
}

func TestEventsListTypeFilter(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[]}`)
	_, err := ts.client.Events.List(context.Background(), &EventListParams{Type: "invoice.paid", Limit: 5})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if ts.query != "limit=5&type=invoice.paid" {
		t.Errorf("query = %q", ts.query)
	}
}

func TestInvoicesGet(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":{"id":"inv_1","status":"paid"}}`)
	inv, err := ts.client.Invoices.Get(context.Background(), "inv_1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if inv.ID != "inv_1" || ts.path != "/invoices/inv_1" {
		t.Errorf("inv = %+v path = %q", inv, ts.path)
	}
}

func TestInvoicesListScopedFilters(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[]}`)
	_, err := ts.client.Invoices.List(context.Background(), &InvoiceListParams{CustomerID: "cus_1", SubscriptionID: "sub_1"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if ts.query != "customer_id=cus_1&subscription_id=sub_1" {
		t.Errorf("query = %q", ts.query)
	}
}

func TestCreditNotesGet(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":{"id":"cn_1","status":"issued"}}`)
	cn, err := ts.client.CreditNotes.Get(context.Background(), "cn_1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if cn.ID != "cn_1" || ts.path != "/credit-notes/cn_1" {
		t.Errorf("cn = %+v path = %q", cn, ts.path)
	}
}

func TestSubscriptionsListCustomerFilter(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[]}`)
	_, err := ts.client.Subscriptions.List(context.Background(), &SubscriptionListParams{CustomerID: "cus_1"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if ts.query != "customer_id=cus_1" {
		t.Errorf("query = %q", ts.query)
	}
}

func TestEventsListObjectFilter(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[]}`)
	_, err := ts.client.Events.List(context.Background(), &EventListParams{ObjectID: "inv_1"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if ts.query != "object_id=inv_1" {
		t.Errorf("query = %q", ts.query)
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
	// The API wraps every response in {"data": ...}; the old client (and this
	// test) decoded the raw body, silently returning zeros in production.
	ts := newTestServer(t, http.StatusOK, `{"data":{"currency":"USD","mrr":250000,"normalized_mrr":250000,"reporting_currency":"USD","fx":{"source":"live"}}}`)
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

// --- SDK catch-up 2026-07-27: collections, entities, analytics, alert edit ---

func TestCollectionsQueue(t *testing.T) {
	ts := newTestServer(t, 200, `{"data":[{"id":"inv_1","status":"past_due","amount_remaining":12500,"last_payment_error":"insufficient_funds","managed_by":"worker"}],"meta":{"page":1,"per_page":25,"total":1}}`)
	items, err := ts.client.Collections.Queue(context.Background(), &CollectionsQueueParams{
		Status: "past_due", ManagedBy: "worker", Page: 1, PerPage: 25,
	})
	if err != nil {
		t.Fatalf("Queue: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/collections/queue")
	for _, want := range []string{"status=past_due", "managed_by=worker", "per_page=25"} {
		if !strings.Contains(ts.query, want) {
			t.Errorf("query %q missing %q", ts.query, want)
		}
	}
	if len(items) != 1 || items[0].AmountRemaining != 12500 || items[0].LastPaymentError != "insufficient_funds" {
		t.Fatalf("items = %+v", items)
	}
}

func TestCollectionsActions(t *testing.T) {
	ts := newTestServer(t, 200, `{"data":{"status":"requeued"}}`)
	if _, err := ts.client.Collections.RetryNow(context.Background(), "inv_1"); err != nil {
		t.Fatalf("RetryNow: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/collections/invoices/inv_1/retry-now")

	ts2 := newTestServer(t, 200, `{"data":{"dunning_paused":true}}`)
	res, err := ts2.client.Collections.PauseDunning(context.Background(), "inv_1", true)
	if err != nil {
		t.Fatalf("PauseDunning: %v", err)
	}
	ts2.assertRequest(http.MethodPost, "/collections/invoices/inv_1/pause")
	if ts2.bodyMap()["paused"] != true || !res.DunningPaused {
		t.Fatalf("pause body/result wrong: %v %+v", ts2.bodyMap(), res)
	}

	ts3 := newTestServer(t, 200, `{"data":{"status":"uncollectible"}}`)
	if _, err := ts3.client.Collections.MarkUncollectible(context.Background(), "inv_1"); err != nil {
		t.Fatalf("MarkUncollectible: %v", err)
	}
	ts3.assertRequest(http.MethodPost, "/collections/invoices/inv_1/mark-uncollectible")
}

func TestCollectionsFunnel(t *testing.T) {
	ts := newTestServer(t, 200, `{"data":{"reporting_currency":"USD","past_due":{"count":4,"amount":40000},"recovery_rate":0.75,"rate_window_days":90}}`)
	f, err := ts.client.Collections.Funnel(context.Background())
	if err != nil {
		t.Fatalf("Funnel: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/analytics/collections/funnel")
	if f.RecoveryRate != 0.75 || f.RateWindowDays != 90 || f.PastDue.Amount != 40000 {
		t.Fatalf("funnel = %+v", f)
	}
}

func TestEntitiesCRUDAndOverview(t *testing.T) {
	ts := newTestServer(t, 201, `{"data":{"id":"ent_2","name":"Branch","invoice_prefix":"BR"}}`)
	e, err := ts.client.Entities.Create(context.Background(), &EntityParams{Name: "Branch", InvoicePrefix: "BR", CountryCode: "US"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/entities")
	if ts.bodyMap()["name"] != "Branch" || e.InvoicePrefix != "BR" {
		t.Fatalf("create body/result wrong: %v %+v", ts.bodyMap(), e)
	}

	ts2 := newTestServer(t, 200, `{"data":{"reporting_currency":"USD","total_mrr":400000,"entities":[{"entity_id":"ent_1","mrr":300000,"is_primary":true}]}}`)
	ov, err := ts2.client.Entities.Overview(context.Background())
	if err != nil {
		t.Fatalf("Overview: %v", err)
	}
	ts2.assertRequest(http.MethodGet, "/analytics/entities-overview")
	if ov.TotalMRR != 400000 || len(ov.Entities) != 1 || ov.Entities[0].MRR != 300000 {
		t.Fatalf("overview = %+v", ov)
	}
}

// The MRR envelope fix: the API wraps in {"data": ...}; the old client decoded
// the raw body and silently returned zeros.
func TestAnalyticsMRRUnwrapsEnvelope(t *testing.T) {
	ts := newTestServer(t, 200, `{"data":{"mrr":123400,"normalized_mrr":123400,"reporting_currency":"USD"}}`)
	m, err := ts.client.Analytics.MRR(context.Background())
	if err != nil {
		t.Fatalf("MRR: %v", err)
	}
	if m.NormalizedMRR != 123400 || m.ReportingCurrency != "USD" {
		t.Fatalf("MRR did not unwrap the data envelope: %+v", m)
	}
}

func TestAnalyticsMRRByEntityAndAging(t *testing.T) {
	ts := newTestServer(t, 200, `{"data":{"reporting_currency":"USD","total_mrr":101,"entities":[{"entity_id":"e1","normalized_mrr":51}]}}`)
	b, err := ts.client.Analytics.MRRByEntity(context.Background())
	if err != nil || b.TotalMRR != 101 {
		t.Fatalf("MRRByEntity: %+v err=%v", b, err)
	}
	ts.assertRequest(http.MethodGet, "/analytics/mrr/by-entity")

	ts2 := newTestServer(t, 200, `{"data":{"reporting_currency":"USD","total_outstanding":7000,"buckets":[{"label":"1-30","count":1,"amount":7000}]}}`)
	rep, err := ts2.client.Analytics.InvoiceAging(context.Background(), "ent_1")
	if err != nil || rep.TotalOutstanding != 7000 {
		t.Fatalf("InvoiceAging: %+v err=%v", rep, err)
	}
	ts2.assertRequest(http.MethodGet, "/analytics/invoice-aging")
	if !strings.Contains(ts2.query, "entity_id=ent_1") {
		t.Errorf("query %q missing entity_id", ts2.query)
	}
}

func TestUsageAlertsUpdate(t *testing.T) {
	ts := newTestServer(t, 200, `{"data":{"id":"al_1","threshold_type":"quantity","threshold":5000}}`)
	a, err := ts.client.UsageAlerts.Update(context.Background(), "al_1", &UsageAlertUpdateParams{
		ThresholdType: "quantity", Threshold: 5000,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	ts.assertRequest(http.MethodPut, "/usage-alerts/al_1")
	if ts.bodyMap()["threshold"] != float64(5000) || a.Threshold != 5000 {
		t.Fatalf("update body/result wrong: %v %+v", ts.bodyMap(), a)
	}
}

func TestWalletsClose(t *testing.T) {
	ts := newTestServer(t, 200, `{"data":{"refunded":4000,"forfeited":1000}}`)
	res, err := ts.client.Wallets.Close(context.Background(), "wal_1")
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/wallets/wal_1/close")
	if res.Refunded != 4000 || res.Forfeited != 1000 {
		t.Fatalf("close result = %+v", res)
	}
}

func TestCustomersCreditStatement(t *testing.T) {
	ts := newTestServer(t, 200, `{"data":{"customer_id":"cus_1","balances":[{"currency":"USD","balance":9000}],"summary":[{"currency":"USD","total_issued":10000,"total_applied":1000,"current_balance":9000}]}}`)
	st, err := ts.client.Customers.CreditStatement(context.Background(), "cus_1")
	if err != nil {
		t.Fatalf("CreditStatement: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/customers/cus_1/credit-statement")
	if len(st.Balances) != 1 || st.Balances[0].Balance != 9000 || st.Summary[0].CurrentBalance != 9000 {
		t.Fatalf("statement = %+v", st)
	}
}

func TestGiftsCancel(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":{"gift":{"id":"gift_1","status":"canceled"},"credit_note":{"id":"cn_1","amount":29900,"status":"issued"},"invoice_voided":false}}`)
	res, err := ts.client.Gifts.Cancel(context.Background(), "gift_1")
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/gifts/gift_1/cancel")
	if res.Gift.Status != "canceled" || res.CreditNote == nil || res.CreditNote.Amount != 29900 || res.InvoiceVoided {
		t.Errorf("res = %+v", res)
	}
}

func TestDisputesList(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[{"id":"dsp_1","invoice_id":"inv_1","reason":"double charge","status":"open"}]}`)
	disputes, err := ts.client.Disputes.List(context.Background(), &DisputeListParams{Status: "open", Limit: 50, Offset: 100})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/disputes")
	if ts.query != "limit=50&offset=100&status=open" {
		t.Errorf("query = %q", ts.query)
	}
	if len(disputes) != 1 || disputes[0].Reason != "double charge" {
		t.Errorf("disputes = %+v", disputes)
	}
}

func TestDisputesResolve(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"status":"resolved"}`)
	if err := ts.client.Disputes.Resolve(context.Background(), "dsp_1", &DisputeResolveParams{Note: "refunded"}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/disputes/dsp_1/resolve")
}

func TestQuotesListPaging(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[]}`)
	if _, err := ts.client.Quotes.List(context.Background(), &QuoteListParams{Limit: 200, Offset: 400}); err != nil {
		t.Fatalf("List: %v", err)
	}
	if ts.query != "limit=200&offset=400" {
		t.Errorf("query = %q", ts.query)
	}
}

// apiCall is one table-driven request/response check: it drives a client
// method against a mock server and asserts method, path, query, and the
// decoded result.
type apiCall struct {
	name   string
	status int
	body   string
	method string
	path   string
	query  string
	fn     func(c *Client) (any, error)
	check  func(t *testing.T, got any)
}

func runCalls(t *testing.T, calls []apiCall) {
	t.Helper()
	for _, tc := range calls {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			status := tc.status
			if status == 0 {
				status = http.StatusOK
			}
			ts := newTestServer(t, status, tc.body)
			got, err := tc.fn(ts.client)
			if err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			ts.assertRequest(tc.method, tc.path)
			if ts.query != tc.query {
				t.Errorf("query = %q, want %q", ts.query, tc.query)
			}
			if tc.check != nil {
				tc.check(t, got)
			}
		})
	}
}

func TestCustomersFinancialSummary(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":{"customer_id":"cus_1","currencies":[{"currency":"USD","outstanding":5000,"past_due":2000,"past_due_count":1,"billed":90000,"paid":85000}]}}`)
	fs, err := ts.client.Customers.FinancialSummary(context.Background(), "cus_1")
	if err != nil {
		t.Fatalf("FinancialSummary: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/customers/cus_1/financial-summary")
	if fs.CustomerID != "cus_1" || len(fs.Currencies) != 1 || fs.Currencies[0].PastDue != 2000 || fs.Currencies[0].Paid != 85000 {
		t.Errorf("summary = %+v", fs)
	}
}

func TestCouponsGet(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":{"id":"cpn_1","code":"SAVE10","discount_type":"percent","discount_value":10,"duration":"forever"}}`)
	c, err := ts.client.Coupons.Get(context.Background(), "cpn_1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/coupons/cpn_1")
	if c.Code != "SAVE10" || c.DiscountValue != 10 {
		t.Errorf("coupon = %+v", c)
	}
}

func TestDisputesGet(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":{"id":"dsp_1","invoice_id":"inv_1","reason":"double charge","status":"open","note":null}}`)
	d, err := ts.client.Disputes.Get(context.Background(), "dsp_1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/disputes/dsp_1")
	if d.Reason != "double charge" || d.Status != "open" {
		t.Errorf("dispute = %+v", d)
	}
}

func TestDeveloperRevokeKey(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"status":"revoked"}`)
	res, err := ts.client.Developer.RevokeKey(context.Background(), "key_1")
	if err != nil {
		t.Fatalf("RevokeKey: %v", err)
	}
	ts.assertRequest(http.MethodDelete, "/developer/keys/key_1")
	if res.Status != "revoked" {
		t.Errorf("status = %q", res.Status)
	}
}

func TestCreditNotesLifecycle(t *testing.T) {
	cn := `{"data":{"id":"cn_1","customer_id":"cus_1","amount":5000,"balance":5000,"currency":"USD","status":"%s"}}`
	check := func(status string) func(t *testing.T, got any) {
		return func(t *testing.T, got any) {
			n := got.(*CreditNote)
			if n.ID != "cn_1" || n.Status != status || n.Amount != 5000 {
				t.Errorf("credit note = %+v", n)
			}
		}
	}
	runCalls(t, []apiCall{
		{name: "approve", body: fmt.Sprintf(cn, "issued"), method: http.MethodPost, path: "/credit-notes/cn_1/approve",
			fn: func(c *Client) (any, error) { return c.CreditNotes.Approve(context.Background(), "cn_1") }, check: check("issued")},
		{name: "reject", body: fmt.Sprintf(cn, "rejected"), method: http.MethodPost, path: "/credit-notes/cn_1/reject",
			fn: func(c *Client) (any, error) { return c.CreditNotes.Reject(context.Background(), "cn_1") }, check: check("rejected")},
		{name: "void", body: fmt.Sprintf(cn, "void"), method: http.MethodPost, path: "/credit-notes/cn_1/void",
			fn: func(c *Client) (any, error) { return c.CreditNotes.Void(context.Background(), "cn_1") }, check: check("void")},
		{name: "journal-entries", body: `{"data":{"credit_note_id":"cn_1","entries":[{"transaction_id":"tx_1","code":5,"debit_account_code":4000,"debit_account_name":"Revenue","credit_account_code":1200,"credit_account_name":"AR","amount":5000,"accounting_version":2}]}}`,
			method: http.MethodGet, path: "/credit-notes/cn_1/journal-entries",
			fn: func(c *Client) (any, error) { return c.CreditNotes.JournalEntries(context.Background(), "cn_1") },
			check: func(t *testing.T, got any) {
				je := got.(*CreditNoteJournalEntries)
				if je.CreditNoteID != "cn_1" || len(je.Entries) != 1 || je.Entries[0].DebitAccountCode != 4000 || je.Entries[0].Amount != 5000 {
					t.Errorf("entries = %+v", je)
				}
			}},
	})
}

func TestCreditNotesDownloadPDF(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `<html>credit note</html>`)
	doc, err := ts.client.CreditNotes.DownloadPDF(context.Background(), "cn_1")
	if err != nil {
		t.Fatalf("DownloadPDF: %v", err)
	}
	if ts.method != http.MethodGet || ts.path != "/credit-notes/cn_1/pdf" || ts.accept != "text/html" {
		t.Errorf("request = %s %s accept=%s", ts.method, ts.path, ts.accept)
	}
	if string(doc) != "<html>credit note</html>" {
		t.Errorf("doc = %q", doc)
	}
}

func TestInvoicesDrilldowns(t *testing.T) {
	runCalls(t, []apiCall{
		{name: "journal-entries", body: `{"data":{"invoice_id":"inv_1","entries":[{"transaction_id":"tx_1","debit_account_code":1200,"credit_account_code":4000,"amount":2900}]}}`,
			method: http.MethodGet, path: "/invoices/inv_1/journal-entries",
			fn: func(c *Client) (any, error) { return c.Invoices.JournalEntries(context.Background(), "inv_1") },
			check: func(t *testing.T, got any) {
				je := got.(*InvoiceJournalEntries)
				if je.InvoiceID != "inv_1" || len(je.Entries) != 1 || je.Entries[0].CreditAccountCode != 4000 {
					t.Errorf("entries = %+v", je)
				}
			}},
		{name: "payment-attempts", body: `{"data":{"invoice_id":"inv_1","attempts":[{"id":"pa_1","gateway":"stripe","status":"failed","failure_code":"card_declined","amount":2900,"settled_at":null}]}}`,
			method: http.MethodGet, path: "/invoices/inv_1/payment-attempts",
			fn: func(c *Client) (any, error) { return c.Invoices.PaymentAttempts(context.Background(), "inv_1") },
			check: func(t *testing.T, got any) {
				pa := got.(*InvoicePaymentAttempts)
				if len(pa.Attempts) != 1 || pa.Attempts[0].FailureCode != "card_declined" || pa.Attempts[0].SettledAt != nil {
					t.Errorf("attempts = %+v", pa)
				}
			}},
		{name: "status-history", body: `{"data":{"invoice_id":"inv_1","history":[{"id":"h_1","from_status":null,"to_status":"open","changed_at":"2026-08-01T00:00:00Z"},{"id":"h_2","from_status":"open","to_status":"paid","changed_at":"2026-08-02T00:00:00Z"}]}}`,
			method: http.MethodGet, path: "/invoices/inv_1/status-history",
			fn: func(c *Client) (any, error) { return c.Invoices.StatusHistory(context.Background(), "inv_1") },
			check: func(t *testing.T, got any) {
				h := got.(*InvoiceStatusHistory)
				if len(h.History) != 2 || h.History[0].FromStatus != nil || *h.History[1].FromStatus != "open" || h.History[1].ToStatus != "paid" {
					t.Errorf("history = %+v", h)
				}
			}},
		{name: "send", body: `{"message":"Invoice emailed"}`, method: http.MethodPost, path: "/invoices/inv_1/send",
			fn: func(c *Client) (any, error) { return c.Invoices.Send(context.Background(), "inv_1") },
			check: func(t *testing.T, got any) {
				if got.(*MessageResponse).Message != "Invoice emailed" {
					t.Errorf("message = %+v", got)
				}
			}},
		{name: "eu-einvoice", body: `{"data":{"id":"eu_1","invoice_id":"inv_1","syntax":"ubl","status":"sent","document":"<Invoice/>","retry_count":0}}`,
			method: http.MethodGet, path: "/invoices/inv_1/eu-einvoice",
			fn: func(c *Client) (any, error) { return c.Invoices.EUEInvoice(context.Background(), "inv_1") },
			check: func(t *testing.T, got any) {
				e := got.(*EUEInvoice)
				if e.Status != "sent" || e.Document != "<Invoice/>" {
					t.Errorf("eu einvoice = %+v", e)
				}
			}},
		{name: "eu-einvoice-nil", body: `{"data":null}`, method: http.MethodGet, path: "/invoices/inv_1/eu-einvoice",
			fn: func(c *Client) (any, error) { return c.Invoices.EUEInvoice(context.Background(), "inv_1") },
			check: func(t *testing.T, got any) {
				if got.(*EUEInvoice) != nil {
					t.Errorf("want nil, got %+v", got)
				}
			}},
		{name: "eu-einvoice-retry", body: `{"data":{"id":"eu_1","status":"pending","retry_count":1},"message":"re-queued"}`,
			method: http.MethodPost, path: "/invoices/inv_1/eu-einvoice/retry",
			fn: func(c *Client) (any, error) { return c.Invoices.RetryEUEInvoice(context.Background(), "inv_1") },
			check: func(t *testing.T, got any) {
				r := got.(*EUEInvoiceRetryResult)
				if r.Message != "re-queued" || r.EUEInvoice == nil || r.EUEInvoice.RetryCount != 1 {
					t.Errorf("retry = %+v", r)
				}
			}},
		{name: "payment-wall", body: `{"invoice_id":"inv_1","payment_wall_active":true}`, method: http.MethodGet, path: "/invoices/inv_1/payment-wall",
			fn: func(c *Client) (any, error) { return c.Invoices.PaymentWall(context.Background(), "inv_1") },
			check: func(t *testing.T, got any) {
				if !got.(*PaymentWallStatus).PaymentWallActive {
					t.Errorf("wall = %+v", got)
				}
			}},
	})
}

func TestInvoicesRawDocuments(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `<html>invoice</html>`)
	doc, err := ts.client.Invoices.DownloadPDF(context.Background(), "inv_1")
	if err != nil {
		t.Fatalf("DownloadPDF: %v", err)
	}
	if ts.method != http.MethodGet || ts.path != "/invoices/inv_1/pdf" || ts.accept != "text/html" || string(doc) != "<html>invoice</html>" {
		t.Errorf("request = %s %s accept=%s doc=%q", ts.method, ts.path, ts.accept, doc)
	}

	ts = newTestServer(t, http.StatusOK, `<html>preview</html>`)
	doc, err = ts.client.Invoices.PreviewHTML(context.Background(), "inv_1")
	if err != nil {
		t.Fatalf("PreviewHTML: %v", err)
	}
	if ts.path != "/invoices/inv_1/preview" || string(doc) != "<html>preview</html>" {
		t.Errorf("path = %s doc=%q", ts.path, doc)
	}
}

func TestRawRequestError(t *testing.T) {
	ts := newTestServer(t, http.StatusNotFound, `{"error":{"code":"not_found","message":"invoice not found"}}`)
	_, err := ts.client.Invoices.DownloadPDF(context.Background(), "missing")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 404 || apiErr.Code != "not_found" {
		t.Fatalf("err = %v", err)
	}
}

func TestSubscriptionsFinancials(t *testing.T) {
	runCalls(t, []apiCall{
		{name: "financial-summary", body: `{"data":{"subscription_id":"sub_1","status":"active","currency":"USD","mrr":2900,"recurring_amount":2900,"interval_unit":"month","interval_count":1,"next_invoice_date":"2026-10-01T00:00:00Z","next_invoice_base_amount":2900,"coupon_id":null,"discount_active":false,"outstanding":[{"currency":"USD","outstanding":0,"billed":5800,"paid":5800}]}}`,
			method: http.MethodGet, path: "/subscriptions/sub_1/financial-summary",
			fn: func(c *Client) (any, error) { return c.Subscriptions.FinancialSummary(context.Background(), "sub_1") },
			check: func(t *testing.T, got any) {
				fs := got.(*SubscriptionFinancialSummary)
				if fs.MRR != 2900 || fs.NextInvoiceDate == nil || fs.CouponID != nil || len(fs.Outstanding) != 1 || fs.Outstanding[0].Billed != 5800 {
					t.Errorf("summary = %+v", fs)
				}
			}},
		{name: "cancel-preview", body: `{"data":{"subscription_id":"sub_1","immediately":true,"resulting_status":"canceled","currency":"USD","deferred_revenue_forfeited":1450,"recognized_as_breakage":1450,"avoided_future_recurring":2900,"flat_fee_refund":0}}`,
			method: http.MethodGet, path: "/subscriptions/sub_1/cancel-preview", query: "immediately=true",
			fn: func(c *Client) (any, error) {
				return c.Subscriptions.CancelPreview(context.Background(), "sub_1", true)
			},
			check: func(t *testing.T, got any) {
				p := got.(*CancelPreview)
				if !p.Immediately || p.DeferredRevenueForfeited != 1450 || p.AvoidedFutureRecurring != 2900 {
					t.Errorf("preview = %+v", p)
				}
			}},
		{name: "cancel-preview-period-end", body: `{"data":{"subscription_id":"sub_1","immediately":false}}`,
			method: http.MethodGet, path: "/subscriptions/sub_1/cancel-preview",
			fn: func(c *Client) (any, error) {
				return c.Subscriptions.CancelPreview(context.Background(), "sub_1", false)
			}},
		{name: "history", body: `{"data":{"subscription_id":"sub_1","history":[{"id":"h_1","change_type":"status","from_value":null,"to_value":"active","changed_at":"2026-08-01T00:00:00Z"},{"id":"h_2","change_type":"plan","from_value":"plan_1","to_value":"plan_2","changed_at":"2026-08-15T00:00:00Z"}]}}`,
			method: http.MethodGet, path: "/subscriptions/sub_1/history",
			fn: func(c *Client) (any, error) { return c.Subscriptions.History(context.Background(), "sub_1") },
			check: func(t *testing.T, got any) {
				h := got.(*SubscriptionHistory)
				if len(h.History) != 2 || h.History[0].FromValue != nil || *h.History[1].FromValue != "plan_1" || h.History[1].ChangeType != "plan" {
					t.Errorf("history = %+v", h)
				}
			}},
		{name: "bill-usage", status: http.StatusCreated, body: `{"id":"inv_9","subscription_id":"sub_1","billing_reason":"usage_interim","total":1234,"status":"open"}`,
			method: http.MethodPost, path: "/subscriptions/sub_1/bill-usage",
			fn: func(c *Client) (any, error) { return c.Subscriptions.BillUsage(context.Background(), "sub_1") },
			check: func(t *testing.T, got any) {
				inv := got.(*Invoice)
				if inv.ID != "inv_9" || inv.Total != 1234 {
					t.Errorf("invoice = %+v", inv)
				}
			}},
		{name: "consent", body: `{"id":"con_1","customer_id":"cus_1","subscription_id":"sub_1","consent_type":"recurring_billing","granted":true,"version":"v1"}`,
			method: http.MethodGet, path: "/subscriptions/sub_1/consent",
			fn: func(c *Client) (any, error) { return c.Subscriptions.Consent(context.Background(), "sub_1") },
			check: func(t *testing.T, got any) {
				con := got.(*Consent)
				if con.ID != "con_1" || con.ConsentType != "recurring_billing" || !con.Granted {
					t.Errorf("consent = %+v", con)
				}
			}},
		{name: "cancellation-reasons", body: `{"object":"list","data":[{"id":"too_expensive","label":"Too expensive","allows_feedback":true},{"id":"other","label":"Other","allows_feedback":true}]}`,
			method: http.MethodGet, path: "/cancellation-reasons",
			fn: func(c *Client) (any, error) { return c.Subscriptions.CancellationReasons(context.Background()) },
			check: func(t *testing.T, got any) {
				rs := got.([]CancellationReason)
				if len(rs) != 2 || rs[0].ID != "too_expensive" || !rs[0].AllowsFeedback {
					t.Errorf("reasons = %+v", rs)
				}
			}},
	})
}

func TestConsents(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"id":"con_1","customer_id":"cus_1","consent_type":"recurring_billing","granted":true}`)
	con, err := ts.client.Consents.Record(context.Background(), &ConsentRecordParams{CustomerID: "cus_1", ConsentType: "recurring_billing", Granted: true, ConsentText: "I agree"})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/consents")
	if body := ts.bodyMap(); body["consent_type"] != "recurring_billing" || body["granted"] != true || body["consent_text"] != "I agree" {
		t.Errorf("body = %v", body)
	}
	if con.ID != "con_1" || !con.Granted {
		t.Errorf("consent = %+v", con)
	}

	ts = newTestServer(t, http.StatusOK, `{"message":"consent revoked"}`)
	res, err := ts.client.Consents.Revoke(context.Background(), "con_1")
	if err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/consents/revoke")
	if ts.bodyMap()["consent_id"] != "con_1" || res.Message != "consent revoked" {
		t.Errorf("body = %v res = %+v", ts.bodyMap(), res)
	}
}

func TestAccountingConnect(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"auth_url":"https://appcenter.intuit.com/connect?state=abc"}`)
	res, err := ts.client.Accounting.Connect(context.Background(), "quickbooks")
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/accounting/connect/quickbooks")
	if res.AuthURL != "https://appcenter.intuit.com/connect?state=abc" {
		t.Errorf("auth_url = %q", res.AuthURL)
	}
	want := ts.server.URL + "/accounting/callback/quickbooks?code=c1&realmId=r1&state=s1"
	if got := ts.client.Accounting.CallbackURL("quickbooks", "c1", "s1", "r1"); got != want {
		t.Errorf("CallbackURL = %q, want %q", got, want)
	}
}

func TestAnalyticsReports(t *testing.T) {
	runCalls(t, []apiCall{
		{name: "ask", body: `{"data":[{"plan":"Pro","mrr":2900}],"query":"SELECT plan, mrr FROM ..."}`, method: http.MethodPost, path: "/analytics/ask",
			fn: func(c *Client) (any, error) { return c.Analytics.Ask(context.Background(), "MRR by plan?") },
			check: func(t *testing.T, got any) {
				a := got.(*AnalyticsAnswer)
				var rows []map[string]any
				if err := json.Unmarshal(a.Data, &rows); err != nil || len(rows) != 1 || rows[0]["plan"] != "Pro" || !strings.HasPrefix(a.Query, "SELECT") {
					t.Errorf("answer = %+v (%v)", a, err)
				}
			}},
		{name: "dunning-overview", body: `{"total_retries":40,"total_successes":10,"success_rate":0.25}`, method: http.MethodGet, path: "/analytics/dunning/overview",
			fn: func(c *Client) (any, error) { return c.Analytics.DunningOverview(context.Background()) },
			check: func(t *testing.T, got any) {
				o := got.(*DunningOverview)
				if o.TotalRetries != 40 || o.SuccessRate != 0.25 {
					t.Errorf("overview = %+v", o)
				}
			}},
		{name: "dunning-weights", body: `{"data":[{"context_key":"card_declined","action_id":"retry_1d","average_reward":0.4,"sample_count":12}]}`, method: http.MethodGet, path: "/analytics/dunning/weights",
			fn: func(c *Client) (any, error) { return c.Analytics.DunningWeights(context.Background()) },
			check: func(t *testing.T, got any) {
				w := got.([]DunningWeight)
				if len(w) != 1 || w[0].ActionID != "retry_1d" || w[0].SampleCount != 12 {
					t.Errorf("weights = %+v", w)
				}
			}},
		{name: "dunning-history", body: `{"data":[{"id":"d_1","invoice_id":"inv_1","action_id":"retry_1d","retry_interval":86400,"outcome":"success","reward":1}]}`, method: http.MethodGet, path: "/analytics/dunning/history", query: "limit=25",
			fn: func(c *Client) (any, error) { return c.Analytics.DunningHistory(context.Background(), 25) },
			check: func(t *testing.T, got any) {
				h := got.([]DunningAttempt)
				if len(h) != 1 || h[0].Outcome != "success" || h[0].RetryInterval != 86400 {
					t.Errorf("history = %+v", h)
				}
			}},
		{name: "dunning-recovered", body: `{"recovered_amount_total":{"USD":12000},"recovered_count":3,"avg_attempts":1.5,"avg_days_to_recover":2.2,"monthly":[{"month":"2026-08","currency":"USD","amount":12000,"count":3}]}`, method: http.MethodGet, path: "/analytics/dunning/recovered",
			fn: func(c *Client) (any, error) { return c.Analytics.DunningRecovered(context.Background()) },
			check: func(t *testing.T, got any) {
				r := got.(*DunningRecovered)
				if r.RecoveredAmountTotal["USD"] != 12000 || r.RecoveredCount != 3 || len(r.Monthly) != 1 || r.Monthly[0].Amount != 12000 {
					t.Errorf("recovered = %+v", r)
				}
			}},
		{name: "mrr-waterfall", body: `{"data":{"start_date":"2026-07-01T00:00:00Z","end_date":"2026-08-01T00:00:00Z","starting_mrr":10000,"new":2000,"expansion":500,"contraction":300,"churned":700,"reactivation":0,"ending_mrr":11500,"reporting_currency":"USD","net_dollar_retention":0.95,"gross_dollar_retention":0.9,"has_start_history":true}}`,
			method: http.MethodGet, path: "/analytics/mrr/waterfall", query: "end=2026-08-01&entity_id=ent_1&start=2026-07-01",
			fn: func(c *Client) (any, error) {
				return c.Analytics.MRRWaterfall(context.Background(), &MRRWaterfallParams{Start: "2026-07-01", End: "2026-08-01", EntityID: "ent_1"})
			},
			check: func(t *testing.T, got any) {
				w := got.(*MRRWaterfall)
				if w.StartingMRR != 10000 || w.EndingMRR != 11500 || w.Churned != 700 || !w.HasStartHistory {
					t.Errorf("waterfall = %+v", w)
				}
			}},
		{name: "mrr-waterfall-default", body: `{"data":{"starting_mrr":1}}`, method: http.MethodGet, path: "/analytics/mrr/waterfall",
			fn: func(c *Client) (any, error) { return c.Analytics.MRRWaterfall(context.Background(), nil) }},
		{name: "revenue-by-plan", body: `{"data":{"reporting_currency":"USD","total_mrr":11500,"segments":[{"key":"plan_1","label":"Pro","mrr":9000,"subscriptions":3,"share_pct":78.3}]}}`, method: http.MethodGet, path: "/analytics/revenue-by-plan",
			fn: func(c *Client) (any, error) { return c.Analytics.RevenueByPlan(context.Background()) },
			check: func(t *testing.T, got any) {
				r := got.(*RevenueBreakdown)
				if r.TotalMRR != 11500 || len(r.Segments) != 1 || r.Segments[0].Label != "Pro" || r.Segments[0].SharePct != 78.3 {
					t.Errorf("by plan = %+v", r)
				}
			}},
		{name: "revenue-by-geography", body: `{"data":{"reporting_currency":"USD","total_mrr":11500,"segments":[{"key":"IN","label":"India","mrr":2500,"subscriptions":5,"share_pct":21.7}]}}`, method: http.MethodGet, path: "/analytics/revenue-by-geography",
			fn: func(c *Client) (any, error) { return c.Analytics.RevenueByGeography(context.Background()) },
			check: func(t *testing.T, got any) {
				r := got.(*RevenueBreakdown)
				if len(r.Segments) != 1 || r.Segments[0].Key != "IN" || r.Segments[0].MRR != 2500 {
					t.Errorf("by geo = %+v", r)
				}
			}},
		{name: "unit-economics", body: `{"data":{"reporting_currency":"USD","mrr":11500,"active_customers":10,"active_subscriptions":12,"arpa":1150,"arpu":958,"monthly_churn_rate":2.5,"ltv":46000,"has_ltv":true}}`, method: http.MethodGet, path: "/analytics/unit-economics",
			fn: func(c *Client) (any, error) { return c.Analytics.UnitEconomics(context.Background()) },
			check: func(t *testing.T, got any) {
				u := got.(*UnitEconomics)
				if u.ARPA != 1150 || u.LTV != 46000 || !u.HasLTV || u.MonthlyChurnRate != 2.5 {
					t.Errorf("unit economics = %+v", u)
				}
			}},
		{name: "usage-stats", body: `{"data":[{"dimension":"api_calls","total_quantity":120000}],"customers_metered":7}`, method: http.MethodGet, path: "/analytics/usage",
			fn: func(c *Client) (any, error) { return c.Analytics.UsageStats(context.Background()) },
			check: func(t *testing.T, got any) {
				u := got.(*UsageStats)
				if u.CustomersMetered != 7 || len(u.Dimensions) != 1 || u.Dimensions[0].TotalQuantity != 120000 {
					t.Errorf("usage = %+v", u)
				}
			}},
	})
}

func TestLedgerReports(t *testing.T) {
	runCalls(t, []apiCall{
		{name: "transaction", body: `{"data":{"transaction_id":"tx_1","code":1,"debit_account_id":"acc_1","debit_account_code":1200,"debit_account_name":"AR","credit_account_id":"acc_2","credit_account_code":4000,"credit_account_name":"Revenue","amount":2900,"reference_id":"inv_1","description":"Invoice INV-1","accounting_version":2,"entity_id":"ent_1","entity_name":"Recurso Inc"}}`,
			method: http.MethodGet, path: "/ledger/transactions/tx_1",
			fn: func(c *Client) (any, error) { return c.Ledger.Transaction(context.Background(), "tx_1") },
			check: func(t *testing.T, got any) {
				je := got.(*JournalEntry)
				if je.TransactionID != "tx_1" || je.Amount != 2900 || je.CreditAccountName != "Revenue" || je.EntityID == nil || *je.EntityID != "ent_1" {
					t.Errorf("entry = %+v", je)
				}
			}},
		{name: "trial-balance", body: `{"data":{"tenant_id":"t_1","lines":[{"account_id":"acc_1","code":1200,"name":"AR","type":1,"debits":5000,"credits":2000,"balance":3000,"abnormal":false}],"total_debits":5000,"total_credits":5000,"balanced":true,"as_of":"2026-08-31T00:00:00Z","reporting_currency":"USD"}}`,
			method: http.MethodGet, path: "/ledger/trial-balance", query: "consolidated=true",
			fn: func(c *Client) (any, error) {
				return c.Ledger.TrialBalance(context.Background(), &TrialBalanceParams{Consolidated: true})
			},
			check: func(t *testing.T, got any) {
				tb := got.(*TrialBalance)
				if !tb.Balanced || tb.TotalDebits != 5000 || len(tb.Lines) != 1 || tb.Lines[0].Balance != 3000 || tb.Lines[0].Code != 1200 {
					t.Errorf("trial balance = %+v", tb)
				}
			}},
		{name: "trial-balance-entity", body: `{"data":{"balanced":true}}`, method: http.MethodGet, path: "/ledger/trial-balance", query: "entity_id=ent_1",
			fn: func(c *Client) (any, error) {
				return c.Ledger.TrialBalance(context.Background(), &TrialBalanceParams{EntityID: "ent_1"})
			}},
		{name: "deferred-rollforward", body: `{"data":{"tenant_id":"t_1","period_start":"2026-08-01T00:00:00Z","period_end":"2026-09-01T00:00:00Z","opening":10000,"added":5000,"released":4000,"closing":11000,"reporting_currency":"USD"}}`,
			method: http.MethodGet, path: "/ledger/deferred-rollforward", query: "month=8&year=2026",
			fn: func(c *Client) (any, error) { return c.Ledger.DeferredRollforward(context.Background(), 8, 2026) },
			check: func(t *testing.T, got any) {
				rf := got.(*DeferredRollforward)
				if rf.Opening != 10000 || rf.Closing != 11000 || rf.Opening+rf.Added-rf.Released != rf.Closing {
					t.Errorf("rollforward = %+v", rf)
				}
			}},
	})
}

func TestLedgerExport(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, "date,account,debit,credit\n2026-08-01,1200,2900,0\n")
	csv, err := ts.client.Ledger.Export(context.Background(), &LedgerExportParams{Month: 8, Year: 2026, EntityID: "ent_1"})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if ts.method != http.MethodGet || ts.path != "/ledger/export" || ts.query != "entity_id=ent_1&month=8&year=2026" || ts.accept != "text/csv" {
		t.Errorf("request = %s %s?%s accept=%s", ts.method, ts.path, ts.query, ts.accept)
	}
	if !strings.HasPrefix(string(csv), "date,account") {
		t.Errorf("csv = %q", csv)
	}
}

func TestMeteringChargesAndSimulation(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[{"charge_id":"chg_1","plan_id":"plan_1","plan_name":"Pro","plan_code":"PRO","plan_active":true,"charge_model":"per_unit","pay_in_advance":false}]}`)
	charges, err := ts.client.BillableMetrics.Charges(context.Background(), "bm_1")
	if err != nil {
		t.Fatalf("Charges: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/billable-metrics/bm_1/charges")
	if len(charges) != 1 || charges[0].PlanCode != "PRO" || charges[0].ChargeModel != "per_unit" {
		t.Errorf("charges = %+v", charges)
	}

	ts = newTestServer(t, http.StatusOK, `{"data":{"plan_id":"plan_1","currency":"USD","charges":[{"metric_id":"bm_1","metric_code":"api_calls","charge_model":"per_unit","quantity":1000,"amount":3500}],"subtotal":3500,"gl_preview":[{"account_code":1200,"account_name":"AR","debit":3500,"credit":0},{"account_code":4000,"account_name":"Revenue","debit":0,"credit":3500}],"balanced":true,"note":"read-only"}}`)
	sim, err := ts.client.Plans.SimulateCharges(context.Background(), "plan_1", &SimulateChargesParams{
		Currency: "USD",
		Charges:  []ChargeParams{{MetricID: "bm_1", ChargeModel: "per_unit", Amounts: map[string]ChargeAmounts{"USD": {UnitAmount: "0.0035"}}, FilterKey: "region", Filters: []ChargeFilter{{Value: "eu", Amounts: map[string]ChargeAmounts{"USD": {UnitAmount: "0.004"}}}}}},
		Usage:    []SimulateUsage{{MetricID: "bm_1", Quantity: 1000}},
	})
	if err != nil {
		t.Fatalf("SimulateCharges: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/plans/plan_1/simulate-charges")
	body := ts.bodyMap()
	if body["currency"] != "USD" || len(body["charges"].([]any)) != 1 || len(body["usage"].([]any)) != 1 {
		t.Errorf("body = %v", body)
	}
	if chg := body["charges"].([]any)[0].(map[string]any); chg["filter_key"] != "region" || len(chg["filters"].([]any)) != 1 {
		t.Errorf("charge body = %v", chg)
	}
	if sim.Subtotal != 3500 || !sim.Balanced || len(sim.Charges) != 1 || sim.Charges[0].Amount != 3500 || len(sim.GLPreview) != 2 || sim.GLPreview[1].Credit != 3500 {
		t.Errorf("simulation = %+v", sim)
	}
}

func TestPaymentAttempts(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[{"id":"pa_1","invoice_id":"inv_1","invoice_number":"INV-1","currency":"USD","gateway":"stripe","method":"card","status":"failed","failure_code":"card_declined","amount":2900,"settled_at":null}],"pagination":{"page":2,"per_page":25,"total":51,"total_pages":3}}`)
	page, err := ts.client.PaymentAttempts.List(context.Background(), &PaymentAttemptListParams{Status: "failed", Page: 2, PerPage: 25})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/payment-attempts")
	if ts.query != "page=2&per_page=25&status=failed" {
		t.Errorf("query = %q", ts.query)
	}
	if len(page.Data) != 1 || page.Data[0].FailureCode != "card_declined" || page.Pagination.TotalPages != 3 || page.Pagination.Total != 51 {
		t.Errorf("page = %+v", page)
	}

	ts = newTestServer(t, http.StatusOK, `{"data":[]}`)
	if _, err := ts.client.PaymentAttempts.List(context.Background(), &PaymentAttemptListParams{Q: "INV-1"}); err != nil {
		t.Fatalf("List: %v", err)
	}
	if ts.query != "q=INV-1" {
		t.Errorf("query = %q", ts.query)
	}

	ts = newTestServer(t, http.StatusOK, `{"data":{"id":"pa_1","invoice_id":"inv_1","invoice_number":"INV-1","customer_id":"cus_1","subscription_id":"sub_1","gateway":"stripe","status":"succeeded","amount":2900,"settled_at":"2026-08-02T10:00:00Z"}}`)
	pa, err := ts.client.PaymentAttempts.Get(context.Background(), "pa_1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/payment-attempts/pa_1")
	if pa.CustomerID != "cus_1" || pa.SubscriptionID == nil || *pa.SubscriptionID != "sub_1" || pa.SettledAt == nil || pa.Status != "succeeded" {
		t.Errorf("attempt = %+v", pa)
	}
}

func TestFinance(t *testing.T) {
	recon := `{"data":{"tenant_id":"t_1","invoices_checked":120,"paid_invoices_checked":80,"total_discrepancies":1,"discrepancies":[{"type":"missing_ar_posting","invoice_id":"inv_7","expected_amount":2900,"found_amount":0}],"truncated":false,"tb_compared":true,"reporting_currency":"USD"}}`
	checkRecon := func(t *testing.T, got any) {
		r := got.(*ReconciliationReport)
		if r.InvoicesChecked != 120 || r.TotalDiscrepancies != 1 || len(r.Discrepancies) != 1 || *r.Discrepancies[0].InvoiceID != "inv_7" || r.Discrepancies[0].ExpectedAmount != 2900 || !r.TBCompared {
			t.Errorf("report = %+v", r)
		}
	}
	runCalls(t, []apiCall{
		{name: "reconcile", body: recon, method: http.MethodGet, path: "/finance/reconciliation",
			fn: func(c *Client) (any, error) { return c.Finance.Reconcile(context.Background()) }, check: checkRecon},
		{name: "record-reconciliation", body: recon, method: http.MethodPost, path: "/finance/reconciliation/runs",
			fn: func(c *Client) (any, error) { return c.Finance.RecordReconciliation(context.Background()) }, check: checkRecon},
		{name: "runs", body: `{"data":[{"id":"run_1","run_by":null,"run_at":"2026-08-31T00:00:00Z","invoices_checked":120,"total_discrepancies":0,"tb_compared":true,"tb_accounts_checked":9,"tb_transfers_checked":300}]}`,
			method: http.MethodGet, path: "/finance/reconciliation/runs", query: "limit=10",
			fn: func(c *Client) (any, error) { return c.Finance.ReconciliationRuns(context.Background(), 10) },
			check: func(t *testing.T, got any) {
				runs := got.([]ReconciliationRun)
				if len(runs) != 1 || runs[0].ID != "run_1" || runs[0].RunBy != nil || runs[0].TBTransfersChecked != 300 {
					t.Errorf("runs = %+v", runs)
				}
			}},
		{name: "run", body: `{"data":{"id":"run_1","run_by":"user_1","invoices_checked":120,"total_discrepancies":1,"discrepancies_truncated":false,"discrepancies":[{"type":"abnormal_account_balance","account_code":2100,"expected_amount":0,"found_amount":-50}]}}`,
			method: http.MethodGet, path: "/finance/reconciliation/runs/run_1",
			fn: func(c *Client) (any, error) { return c.Finance.ReconciliationRun(context.Background(), "run_1") },
			check: func(t *testing.T, got any) {
				run := got.(*ReconciliationRun)
				if run.RunBy == nil || *run.RunBy != "user_1" || len(run.Discrepancies) != 1 || run.Discrepancies[0].AccountCode != 2100 || run.Discrepancies[0].FoundAmount != -50 {
					t.Errorf("run = %+v", run)
				}
			}},
		{name: "close-pack", body: `{"data":{"tenant_id":"t_1","period":{"month":8,"year":2026,"start":"2026-08-01T00:00:00Z","end":"2026-09-01T00:00:00Z"},"ready_to_close":false,"blockers":["1 reconciliation discrepancy"],"trial_balance":{"balanced":true,"total_debits":5000,"total_credits":5000},"reconciliation":{"total_discrepancies":1},"deferred_revenue":{"rollforward":{"opening":10000,"added":5000,"released":4000,"closing":11000},"recognition":{"month":8,"year":2026,"recognized_amount":4000,"deferred_balance":11000},"awaiting_payment":0,"unexplained_delta":0,"ties":true},"general_ledger":{"format":"csv","export_url":"/v1/ledger/export?month=8&year=2026"},"reporting_currency":"USD"}}`,
			method: http.MethodGet, path: "/finance/close-pack", query: "month=8&year=2026",
			fn: func(c *Client) (any, error) { return c.Finance.ClosePack(context.Background(), 8, 2026) },
			check: func(t *testing.T, got any) {
				p := got.(*ClosePack)
				if p.ReadyToClose || len(p.Blockers) != 1 || p.Period.Month != 8 || p.TrialBalance == nil || !p.TrialBalance.Balanced ||
					p.Reconciliation == nil || p.Reconciliation.TotalDiscrepancies != 1 || p.Deferred.Rollforward.Closing != 11000 ||
					p.Deferred.Recognition == nil || p.Deferred.Recognition.RecognizedAmount != 4000 || !p.Deferred.Ties || p.GeneralLedger.Format != "csv" {
					t.Errorf("close pack = %+v", p)
				}
			}},
		{name: "revrec-report", body: `{"data":{"month":8,"year":2026,"recognized_amount":4000,"deferred_balance":11000,"upcoming":[{"month":9,"year":2026,"amount":6000},{"month":10,"year":2026,"amount":5000}],"by_currency":[{"currency":"USD","deferred":11000}]}}`,
			method: http.MethodGet, path: "/finance/revrec/report", query: "month=8&year=2026",
			fn: func(c *Client) (any, error) { return c.Finance.RevRecReport(context.Background(), 8, 2026) },
			check: func(t *testing.T, got any) {
				r := got.(*RevRecReport)
				if r.RecognizedAmount != 4000 || len(r.Upcoming) != 2 || r.Upcoming[1].Amount != 5000 || r.ByCurrency[0].Deferred != 11000 {
					t.Errorf("report = %+v", r)
				}
			}},
		{name: "revrec-waterfall", body: `{"data":{"tenant_id":"t_1","buckets":[{"year":2026,"month":8,"recognized":4000,"scheduled":0},{"year":2026,"month":9,"recognized":0,"scheduled":6000}],"total_recognized":4000,"total_scheduled":6000,"reporting_currency":"USD"}}`,
			method: http.MethodGet, path: "/finance/revrec/waterfall",
			fn: func(c *Client) (any, error) { return c.Finance.RevRecWaterfall(context.Background()) },
			check: func(t *testing.T, got any) {
				w := got.(*RevenueWaterfall)
				if len(w.Buckets) != 2 || w.Buckets[1].Scheduled != 6000 || w.TotalRecognized != 4000 {
					t.Errorf("waterfall = %+v", w)
				}
			}},
	})
}

func TestSettings(t *testing.T) {
	bg := context.Background()
	runCalls(t, []apiCall{
		{name: "gst", body: `{"data":{"gstin":"27AAPFU0939F1ZV","state_code":"27","state_name":"Maharashtra","sac_code":"998314","gst_rate":18,"pan":"AAPFU0939F","legal_name":"Recurso Pvt Ltd","has_lut":true}}`,
			method: http.MethodGet, path: "/settings/gst",
			fn: func(c *Client) (any, error) { return c.Settings.GST(bg) },
			check: func(t *testing.T, got any) {
				g := got.(*GSTConfig)
				if g.GSTIN != "27AAPFU0939F1ZV" || g.GSTRate != 18 || !g.HasLUT {
					t.Errorf("gst = %+v", g)
				}
			}},
		{name: "update-gst", body: `{"data":{"gstin":"27AAPFU0939F1ZV","gst_rate":18},"message":"GST configuration updated"}`,
			method: http.MethodPut, path: "/settings/gst",
			fn: func(c *Client) (any, error) {
				return c.Settings.UpdateGST(bg, &GSTConfig{GSTIN: "27AAPFU0939F1ZV", GSTRate: 18})
			},
			check: func(t *testing.T, got any) {
				if got.(*GSTConfig).GSTIN != "27AAPFU0939F1ZV" {
					t.Errorf("gst = %+v", got)
				}
			}},
		{name: "validate-gstin", body: `{"valid":true,"state_code":"27","state_name":"Maharashtra","pan":"AAPFU0939F","message":"valid"}`,
			method: http.MethodPost, path: "/settings/gst/validate",
			fn: func(c *Client) (any, error) { return c.Settings.ValidateGSTIN(bg, "27AAPFU0939F1ZV") },
			check: func(t *testing.T, got any) {
				v := got.(*GSTINValidation)
				if !v.Valid || v.StateCode != "27" || v.PAN != "AAPFU0939F" {
					t.Errorf("validation = %+v", v)
				}
			}},
		{name: "tax-registrations", body: `{"data":[{"state_code":"CA","registration_number":"123","status":"registered","registered_at":"2026-01-01"},{"state_code":"NY","status":"pending","registered_at":null}]}`,
			method: http.MethodGet, path: "/settings/tax/registrations",
			fn: func(c *Client) (any, error) { return c.Settings.TaxRegistrations(bg) },
			check: func(t *testing.T, got any) {
				rs := got.([]TaxRegistration)
				if len(rs) != 2 || rs[0].Status != "registered" || *rs[0].RegisteredAt != "2026-01-01" || rs[1].RegisteredAt != nil {
					t.Errorf("registrations = %+v", rs)
				}
			}},
		{name: "set-tax-registrations", body: `{"data":[{"state_code":"CA","status":"registered"}]}`,
			method: http.MethodPut, path: "/settings/tax/registrations",
			fn: func(c *Client) (any, error) {
				return c.Settings.SetTaxRegistrations(bg, &TaxRegistrationsParams{Registrations: []TaxRegistration{{StateCode: "CA", Status: "registered"}}})
			},
			check: func(t *testing.T, got any) {
				if rs := got.([]TaxRegistration); len(rs) != 1 || rs[0].StateCode != "CA" {
					t.Errorf("registrations = %+v", rs)
				}
			}},
		{name: "tax-nexus", body: `{"data":[{"state_code":"CA","nexus_type":"physical","established_at":"2025-06-01T00:00:00Z","created_at":"2025-06-01T00:00:00Z"}]}`,
			method: http.MethodGet, path: "/settings/tax/nexus", query: "entity_id=ent_1",
			fn: func(c *Client) (any, error) { return c.Settings.TaxNexus(bg, "ent_1") },
			check: func(t *testing.T, got any) {
				ns := got.([]TaxNexusState)
				if len(ns) != 1 || ns[0].NexusType != "physical" || ns[0].EstablishedAt == nil {
					t.Errorf("nexus = %+v", ns)
				}
			}},
		{name: "set-tax-nexus", body: `{"data":[{"state_code":"TX","nexus_type":"economic"}]}`,
			method: http.MethodPut, path: "/settings/tax/nexus",
			fn: func(c *Client) (any, error) {
				return c.Settings.SetTaxNexus(bg, "", &TaxNexusParams{States: []TaxNexusState{{StateCode: "TX", NexusType: "economic"}}})
			},
			check: func(t *testing.T, got any) {
				if ns := got.([]TaxNexusState); len(ns) != 1 || ns[0].StateCode != "TX" {
					t.Errorf("nexus = %+v", ns)
				}
			}},
		{name: "tax-liability", body: `{"data":{"from_date":"2026-01-01","to_date":"2026-12-31","currency":"USD","total_gross_sales":500000,"total_tax_collected":40000,"states":[{"state_code":"CA","gross_sales":300000,"taxable_sales":250000,"exempt_sales":50000,"tax_collected":25000,"invoice_count":30,"has_nexus":true,"nexus_type":"physical"}]}}`,
			method: http.MethodGet, path: "/settings/tax/liability", query: "year=2026",
			fn: func(c *Client) (any, error) { return c.Settings.TaxLiability(bg, &TaxLiabilityParams{Year: 2026}) },
			check: func(t *testing.T, got any) {
				r := got.(*TaxLiabilityReport)
				if r.TotalTaxCollected != 40000 || len(r.States) != 1 || r.States[0].TaxCollected != 25000 || !r.States[0].HasNexus {
					t.Errorf("liability = %+v", r)
				}
			}},
		{name: "tax-liability-range", body: `{"data":{}}`, method: http.MethodGet, path: "/settings/tax/liability", query: "from=2026-01-01&to=2026-03-31",
			fn: func(c *Client) (any, error) {
				return c.Settings.TaxLiability(bg, &TaxLiabilityParams{From: "2026-01-01", To: "2026-03-31"})
			}},
		{name: "tax-nexus-status", body: `{"data":{"year":2026,"dataset_certified":false,"states":[{"state_code":"TX","taxable_sales":450000,"txn_count":120,"threshold":{"state_code":"TX","sales_threshold":500000,"txn_threshold":0,"combinator":"or","measurement_period":"calendar_year","certified":false},"proximity_pct":90,"crossed":false}]}}`,
			method: http.MethodGet, path: "/settings/tax/nexus/status", query: "year=2026",
			fn: func(c *Client) (any, error) { return c.Settings.TaxNexusStatus(bg, 2026) },
			check: func(t *testing.T, got any) {
				r := got.(*NexusStatusReport)
				if r.Year != 2026 || len(r.States) != 1 || r.States[0].ProximityPct != 90 || r.States[0].Threshold.SalesThreshold != 500000 || r.States[0].Crossed {
					t.Errorf("nexus status = %+v", r)
				}
			}},
		{name: "irp", body: `{"data":{"environment":"sandbox","client_id":"cid","username":"u","gstin":"27AAPFU0939F1ZV","is_enabled":true}}`,
			method: http.MethodGet, path: "/settings/irp",
			fn: func(c *Client) (any, error) { return c.Settings.IRP(bg) },
			check: func(t *testing.T, got any) {
				if cfg := got.(*IRPConfig); cfg.Environment != "sandbox" || !cfg.IsEnabled {
					t.Errorf("irp = %+v", cfg)
				}
			}},
		{name: "update-irp", body: `{"data":{"environment":"production","is_enabled":true},"message":"saved"}`,
			method: http.MethodPut, path: "/settings/irp",
			fn: func(c *Client) (any, error) {
				return c.Settings.UpdateIRP(bg, &IRPConfig{Environment: "production", ClientID: "cid", ClientSecret: "sec", IsEnabled: true})
			},
			check: func(t *testing.T, got any) {
				if got.(*IRPConfig).Environment != "production" {
					t.Errorf("irp = %+v", got)
				}
			}},
		{name: "test-irp", body: `{"success":true,"message":"authenticated"}`, method: http.MethodPost, path: "/settings/irp/test",
			fn: func(c *Client) (any, error) { return c.Settings.TestIRP(bg) },
			check: func(t *testing.T, got any) {
				if r := got.(*IRPTestResult); !r.Success || r.Message != "authenticated" {
					t.Errorf("irp test = %+v", r)
				}
			}},
		{name: "eu-einvoice", body: `{"data":{"enabled":true,"legal_name":"Recurso GmbH","vat_number":"DE123","country_code":"DE","city":"Berlin"}}`,
			method: http.MethodGet, path: "/settings/eu-einvoice",
			fn: func(c *Client) (any, error) { return c.Settings.EUEInvoice(bg) },
			check: func(t *testing.T, got any) {
				if cfg := got.(*EUEInvoiceConfig); !cfg.Enabled || cfg.VATNumber != "DE123" {
					t.Errorf("eu config = %+v", cfg)
				}
			}},
		{name: "update-eu-einvoice", body: `{"data":{"enabled":true,"country_code":"DE"}}`, method: http.MethodPut, path: "/settings/eu-einvoice",
			fn: func(c *Client) (any, error) {
				return c.Settings.UpdateEUEInvoice(bg, &EUEInvoiceConfig{Enabled: true, CountryCode: "DE"})
			},
			check: func(t *testing.T, got any) {
				if got.(*EUEInvoiceConfig).CountryCode != "DE" {
					t.Errorf("eu config = %+v", got)
				}
			}},
		{name: "us-tax", body: `{"data":{"legal_name":"Recurso Inc","ein":"12-3456789","address":"1 Main St"}}`, method: http.MethodGet, path: "/settings/tax/us",
			fn: func(c *Client) (any, error) { return c.Settings.USTax(bg) },
			check: func(t *testing.T, got any) {
				if got.(*USTaxConfig).EIN != "12-3456789" {
					t.Errorf("us tax = %+v", got)
				}
			}},
		{name: "update-us-tax", body: `{"data":{"legal_name":"Recurso Inc","ein":"12-3456789"}}`, method: http.MethodPut, path: "/settings/tax/us",
			fn: func(c *Client) (any, error) {
				return c.Settings.UpdateUSTax(bg, &USTaxConfig{LegalName: "Recurso Inc", EIN: "12-3456789"})
			},
			check: func(t *testing.T, got any) {
				if got.(*USTaxConfig).LegalName != "Recurso Inc" {
					t.Errorf("us tax = %+v", got)
				}
			}},
		{name: "invoice-branding", body: `{"data":{"company_name":"Recurso","signatory_name":"Jane","terms":"Net 30"}}`, method: http.MethodGet, path: "/settings/invoice-branding",
			fn: func(c *Client) (any, error) { return c.Settings.InvoiceBranding(bg) },
			check: func(t *testing.T, got any) {
				if b := got.(*InvoiceBranding); b.CompanyName != "Recurso" || b.Terms != "Net 30" {
					t.Errorf("branding = %+v", b)
				}
			}},
		{name: "update-invoice-branding", body: `{"data":{"company_name":"Recurso","terms":"Net 15"}}`, method: http.MethodPut, path: "/settings/invoice-branding",
			fn: func(c *Client) (any, error) {
				return c.Settings.UpdateInvoiceBranding(bg, &InvoiceBranding{CompanyName: "Recurso", Terms: "Net 15"})
			},
			check: func(t *testing.T, got any) {
				if got.(*InvoiceBranding).Terms != "Net 15" {
					t.Errorf("branding = %+v", got)
				}
			}},
		{name: "mcp", body: `{"data":{"tier3_enabled":false}}`, method: http.MethodGet, path: "/settings/mcp",
			fn: func(c *Client) (any, error) { return c.Settings.MCP(bg) },
			check: func(t *testing.T, got any) {
				if got.(*MCPSettings).Tier3Enabled {
					t.Errorf("mcp = %+v", got)
				}
			}},
		{name: "update-mcp", body: `{"data":{"tier3_enabled":true}}`, method: http.MethodPut, path: "/settings/mcp",
			fn: func(c *Client) (any, error) { return c.Settings.UpdateMCP(bg, &MCPSettings{Tier3Enabled: true}) },
			check: func(t *testing.T, got any) {
				if !got.(*MCPSettings).Tier3Enabled {
					t.Errorf("mcp = %+v", got)
				}
			}},
	})
}

func TestSettingsUpdateBodies(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":{"tier3_enabled":true}}`)
	if _, err := ts.client.Settings.UpdateMCP(context.Background(), &MCPSettings{Tier3Enabled: true}); err != nil {
		t.Fatalf("UpdateMCP: %v", err)
	}
	if ts.ctype != "application/json" || ts.bodyMap()["tier3_enabled"] != true {
		t.Errorf("body = %v ctype=%s", ts.bodyMap(), ts.ctype)
	}
	ts = newTestServer(t, http.StatusOK, `{"valid":false}`)
	if _, err := ts.client.Settings.ValidateGSTIN(context.Background(), "bad"); err != nil {
		t.Fatalf("ValidateGSTIN: %v", err)
	}
	if ts.bodyMap()["gstin"] != "bad" {
		t.Errorf("body = %v", ts.bodyMap())
	}
}

func TestImports(t *testing.T) {
	bg := context.Background()
	stripe := &StripeExport{Customers: []map[string]any{{"id": "cus_stripe_1", "email": "a@example.com"}}, Products: []map[string]any{}, Prices: []map[string]any{}, Subscriptions: []map[string]any{}}
	rc := &RevenueCatExport{Subscribers: []map[string]any{{"app_user_id": "u1"}}, Products: []map[string]any{}}
	cb := &ChargebeeExport{Customers: []map[string]any{{"id": "cb_1"}}, Plans: []map[string]any{}, Subscriptions: []map[string]any{}}
	preview := `{"items":[{"kind":"customer","stripe_id":"cus_stripe_1","label":"a@example.com","action":"create","detail":""}],"summary":{"create":1},"warnings":["no payment methods"]}`
	checkPreview := func(t *testing.T, got any) {
		p := got.(*ImportPreview)
		if len(p.Items) != 1 || p.Items[0].Action != "create" || p.Summary["create"] != 1 || len(p.Warnings) != 1 {
			t.Errorf("preview = %+v", p)
		}
	}
	commit := `{"plan":{"customers":1},"created":{"customers":1,"plans":0},"failures":[{"kind":"subscription","stripe_id":"sub_x","error":"unsupported interval"}]}`
	checkCommit := func(t *testing.T, got any) {
		r := got.(*ImportResult)
		if r.Created["customers"] != 1 || len(r.Failures) != 1 || r.Failures[0].Error != "unsupported interval" || len(r.Plan) == 0 {
			t.Errorf("result = %+v", r)
		}
	}
	compare := `{"source":"stripe","customers":{"source":10,"matched":10,"missing":0},"plans":{"source":2,"matched":2,"missing":0},"subscriptions":{"source":8,"matched":7,"missing":1},"issues":[{"kind":"subscription","external_id":"sub_1","field":"status","source":"active","recurso":"missing"}],"ready":false,"generated_at":"2026-08-31T00:00:00Z"}`
	checkCompare := func(t *testing.T, got any) {
		r := got.(*CompareReport)
		if r.Ready || r.Customers.Matched != 10 || r.Subscriptions.Missing != 1 || len(r.Issues) != 1 || r.Issues[0].Field != "status" {
			t.Errorf("compare = %+v", r)
		}
	}
	runCalls(t, []apiCall{
		{name: "stripe-preview", body: preview, method: http.MethodPost, path: "/import/stripe/preview", fn: func(c *Client) (any, error) { return c.Imports.PreviewStripe(bg, stripe) }, check: checkPreview},
		{name: "stripe-commit", body: commit, method: http.MethodPost, path: "/import/stripe/commit", fn: func(c *Client) (any, error) { return c.Imports.CommitStripe(bg, stripe) }, check: checkCommit},
		{name: "stripe-compare", body: compare, method: http.MethodPost, path: "/import/stripe/compare", fn: func(c *Client) (any, error) { return c.Imports.CompareStripe(bg, stripe) }, check: checkCompare},
		{name: "revenuecat-preview", body: preview, method: http.MethodPost, path: "/import/revenuecat/preview", fn: func(c *Client) (any, error) { return c.Imports.PreviewRevenueCat(bg, rc) }, check: checkPreview},
		{name: "revenuecat-commit", body: commit, method: http.MethodPost, path: "/import/revenuecat/commit", fn: func(c *Client) (any, error) { return c.Imports.CommitRevenueCat(bg, rc) }, check: checkCommit},
		{name: "revenuecat-compare", body: compare, method: http.MethodPost, path: "/import/revenuecat/compare", fn: func(c *Client) (any, error) { return c.Imports.CompareRevenueCat(bg, rc) }, check: checkCompare},
		{name: "chargebee-preview", body: `{"items":[{"kind":"customer","chargebee_id":"cb_1","label":"cb_1","action":"link_existing"}],"summary":{"link_existing":1},"warnings":[]}`, method: http.MethodPost, path: "/import/chargebee/preview",
			fn: func(c *Client) (any, error) { return c.Imports.PreviewChargebee(bg, cb) },
			check: func(t *testing.T, got any) {
				if p := got.(*ImportPreview); len(p.Items) != 1 || p.Items[0].ChargebeeID != "cb_1" || p.Items[0].Action != "link_existing" {
					t.Errorf("preview = %+v", p)
				}
			}},
		{name: "chargebee-commit", body: commit, method: http.MethodPost, path: "/import/chargebee/commit", fn: func(c *Client) (any, error) { return c.Imports.CommitChargebee(bg, cb) }, check: checkCommit},
		{name: "chargebee-compare", body: compare, method: http.MethodPost, path: "/import/chargebee/compare", fn: func(c *Client) (any, error) { return c.Imports.CompareChargebee(bg, cb) }, check: checkCompare},
		{name: "compare-reports", body: `{"data":[{"id":"rep_1","source":"stripe","ready":true,"generated_at":"2026-08-31T00:00:00Z"}]}`, method: http.MethodGet, path: "/import/compare-reports", query: "limit=5",
			fn: func(c *Client) (any, error) { return c.Imports.CompareReports(bg, 5) },
			check: func(t *testing.T, got any) {
				if rs := got.([]StoredCompareReport); len(rs) != 1 || rs[0].ID != "rep_1" || !rs[0].Ready {
					t.Errorf("reports = %+v", rs)
				}
			}},
		{name: "compare-report", body: `{"data":{"id":"rep_1","source":"stripe","ready":true,"report":{"source":"stripe","ready":true},"generated_at":"2026-08-31T00:00:00Z"}}`, method: http.MethodGet, path: "/import/compare-reports/rep_1",
			fn: func(c *Client) (any, error) { return c.Imports.CompareReport(bg, "rep_1") },
			check: func(t *testing.T, got any) {
				r := got.(*StoredCompareReport)
				var full CompareReport
				if err := json.Unmarshal(r.Report, &full); err != nil || r.ID != "rep_1" || !full.Ready || full.Source != "stripe" {
					t.Errorf("report = %+v (%v)", r, err)
				}
			}},
	})
}

func TestImportsRequestBody(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"items":[],"summary":{},"warnings":[]}`)
	_, err := ts.client.Imports.PreviewStripe(context.Background(), &StripeExport{Customers: []map[string]any{{"id": "cus_1"}}, Products: []map[string]any{}, Prices: []map[string]any{}, Subscriptions: []map[string]any{}})
	if err != nil {
		t.Fatalf("PreviewStripe: %v", err)
	}
	body := ts.bodyMap()
	if len(body["customers"].([]any)) != 1 || body["customers"].([]any)[0].(map[string]any)["id"] != "cus_1" {
		t.Errorf("body = %v", body)
	}
	if _, ok := body["payment_methods"]; ok {
		t.Errorf("empty payment_methods should be omitted: %v", body)
	}
}

func TestImportsCompareReportDocument(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `<html>receipt</html>`)
	doc, err := ts.client.Imports.CompareReportDocument(context.Background(), "rep_1")
	if err != nil {
		t.Fatalf("CompareReportDocument: %v", err)
	}
	if ts.method != http.MethodGet || ts.path != "/import/compare-reports/rep_1/document" || ts.accept != "text/html" || string(doc) != "<html>receipt</html>" {
		t.Errorf("request = %s %s accept=%s doc=%q", ts.method, ts.path, ts.accept, doc)
	}
}

func TestUsers(t *testing.T) {
	bg := context.Background()
	user := `{"data":{"id":"usr_1","email":"jane@example.com","name":"Jane","role":"%s"}}`
	checkRole := func(role string) func(t *testing.T, got any) {
		return func(t *testing.T, got any) {
			if u := got.(*User); u.ID != "usr_1" || u.Role != role {
				t.Errorf("user = %+v", u)
			}
		}
	}
	runCalls(t, []apiCall{
		{name: "list", body: `{"data":[{"id":"usr_1","email":"jane@example.com","role":"owner"},{"id":"usr_2","email":"bob@example.com","role":"member"}]}`, method: http.MethodGet, path: "/users",
			fn: func(c *Client) (any, error) { return c.Users.List(bg) },
			check: func(t *testing.T, got any) {
				if us := got.([]User); len(us) != 2 || us[1].Role != "member" {
					t.Errorf("users = %+v", us)
				}
			}},
		{name: "create", status: http.StatusCreated, body: fmt.Sprintf(user, "admin"), method: http.MethodPost, path: "/users",
			fn: func(c *Client) (any, error) {
				return c.Users.Create(bg, &UserCreateParams{Email: "jane@example.com", Name: "Jane", Role: "admin", Password: "hunter22"})
			}, check: checkRole("admin")},
		{name: "invite", status: http.StatusCreated, body: fmt.Sprintf(user, "member"), method: http.MethodPost, path: "/users/invite",
			fn: func(c *Client) (any, error) {
				return c.Users.Invite(bg, &UserInviteParams{Email: "jane@example.com", Role: "member"})
			}, check: checkRole("member")},
		{name: "update-role", body: fmt.Sprintf(user, "admin"), method: http.MethodPatch, path: "/users/usr_1",
			fn: func(c *Client) (any, error) { return c.Users.UpdateRole(bg, "usr_1", "admin") }, check: checkRole("admin")},
		{name: "delete", body: `{"status":"deleted"}`, method: http.MethodDelete, path: "/users/usr_1",
			fn: func(c *Client) (any, error) { return c.Users.Delete(bg, "usr_1") },
			check: func(t *testing.T, got any) {
				if got.(*StatusResponse).Status != "deleted" {
					t.Errorf("status = %+v", got)
				}
			}},
	})
}

func TestUsersUpdateRoleBody(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":{"id":"usr_1","role":"admin"}}`)
	if _, err := ts.client.Users.UpdateRole(context.Background(), "usr_1", "admin"); err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}
	if ts.bodyMap()["role"] != "admin" {
		t.Errorf("body = %v", ts.bodyMap())
	}
}

func TestGatewayConnections(t *testing.T) {
	bg := context.Background()
	runCalls(t, []apiCall{
		{name: "list", body: `{"data":{"connections":[{"id":"gc_1","provider":"stripe","mode":"live","public_key":"pk_live_x","has_webhook_secret":true,"webhook_path":"/webhooks/stripe/gc_1"}],"vault_ready":true}}`, method: http.MethodGet, path: "/gateway-connections",
			fn: func(c *Client) (any, error) { return c.GatewayConnections.List(bg) },
			check: func(t *testing.T, got any) {
				l := got.(*GatewayConnectionList)
				if !l.VaultReady || len(l.Connections) != 1 || l.Connections[0].Provider != "stripe" || !l.Connections[0].HasWebhookSecret {
					t.Errorf("list = %+v", l)
				}
			}},
		{name: "create", status: http.StatusCreated, body: `{"data":{"id":"gc_2","provider":"razorpay","mode":"test","public_key":"rzp_test_x","has_webhook_secret":false}}`, method: http.MethodPost, path: "/gateway-connections",
			fn: func(c *Client) (any, error) {
				return c.GatewayConnections.Create(bg, &GatewayConnectionCreateParams{Provider: "razorpay", Mode: "test", PublicKey: "rzp_test_x", SecretKey: "s"})
			},
			check: func(t *testing.T, got any) {
				if gc := got.(*GatewayConnection); gc.ID != "gc_2" || gc.Mode != "test" {
					t.Errorf("connection = %+v", gc)
				}
			}},
		{name: "delete", body: ``, method: http.MethodDelete, path: "/gateway-connections/razorpay",
			fn: func(c *Client) (any, error) { return nil, c.GatewayConnections.Delete(bg, "razorpay") }},
		{name: "set-webhook-secret", body: ``, method: http.MethodPut, path: "/gateway-connections/stripe/webhook-secret",
			fn: func(c *Client) (any, error) {
				return nil, c.GatewayConnections.SetWebhookSecret(bg, "stripe", "whsec_x")
			}},
	})
}

func TestGatewayConnectionsWebhookSecretBody(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, ``)
	if err := ts.client.GatewayConnections.SetWebhookSecret(context.Background(), "stripe", "whsec_x"); err != nil {
		t.Fatalf("SetWebhookSecret: %v", err)
	}
	if ts.bodyMap()["webhook_secret"] != "whsec_x" {
		t.Errorf("body = %v", ts.bodyMap())
	}
}

func TestIntegrationConnections(t *testing.T) {
	bg := context.Background()
	runCalls(t, []apiCall{
		{name: "list", body: `{"data":{"connections":[{"id":"ic_1","category":"crm","provider":"hubspot","config":{"portal_id":"123"},"has_secrets":true}],"vault_ready":true}}`, method: http.MethodGet, path: "/integration-connections",
			fn: func(c *Client) (any, error) { return c.IntegrationConnections.List(bg) },
			check: func(t *testing.T, got any) {
				l := got.(*IntegrationConnectionList)
				if len(l.Connections) != 1 || l.Connections[0].Category != "crm" || l.Connections[0].Config["portal_id"] != "123" {
					t.Errorf("list = %+v", l)
				}
			}},
		{name: "create", status: http.StatusCreated, body: `{"data":{"id":"ic_2","category":"tax","provider":"avalara","config":{},"has_secrets":true}}`, method: http.MethodPost, path: "/integration-connections",
			fn: func(c *Client) (any, error) {
				return c.IntegrationConnections.Create(bg, &IntegrationConnectionCreateParams{Category: "tax", Provider: "avalara", Config: map[string]string{"api_key": "k"}})
			},
			check: func(t *testing.T, got any) {
				if ic := got.(*IntegrationConnection); ic.ID != "ic_2" || ic.Provider != "avalara" {
					t.Errorf("connection = %+v", ic)
				}
			}},
		{name: "delete", body: ``, method: http.MethodDelete, path: "/integration-connections/tax/avalara",
			fn: func(c *Client) (any, error) { return nil, c.IntegrationConnections.Delete(bg, "tax", "avalara") }},
		{name: "sync-crm", body: `{"data":{"contacts_synced":40,"contacts_remaining":0}}`, method: http.MethodPost, path: "/crm/sync",
			fn: func(c *Client) (any, error) { return c.IntegrationConnections.SyncCRM(bg) },
			check: func(t *testing.T, got any) {
				if r := got.(*CRMSyncResult); r.ContactsSynced != 40 {
					t.Errorf("sync = %+v", r)
				}
			}},
	})
}

func TestIndiaGSTReturns(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":{"tenant_id":"t_1","month":8,"year":2026,"b2b":[{"gstin":"29AAAAA0000A1Z5","invoices":[{"invoice_number":"INV-1","date":"2026-08-05T00:00:00Z","place_of_supply":"29","taxable_value":100000,"igst":18000,"cgst":0,"sgst":0,"rate":18}]}],"b2cs":[],"cdnr":[],"hsn_summary":[{"hsn_code":"998314","taxable_value":100000,"igst":18000,"invoice_count":1}],"total_taxable_value":100000,"total_igst":18000,"invoice_count":1},"gov_schema":{"gstin":"27AAPFU0939F1ZV","fp":"082026"}}`)
	r1, err := ts.client.India.GSTR1(context.Background(), &GSTReturnParams{Month: 8, Year: 2026, EntityID: "ent_1"})
	if err != nil {
		t.Fatalf("GSTR1: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/india/gstr1")
	if ts.query != "entity_id=ent_1&month=8&year=2026" {
		t.Errorf("query = %q", ts.query)
	}
	if r1.Data.TotalIGST != 18000 || len(r1.Data.B2B) != 1 || r1.Data.B2B[0].Invoices[0].Rate != 18 || r1.Data.HSNSummary[0].HSNCode != "998314" {
		t.Errorf("gstr1 = %+v", r1.Data)
	}
	var gov map[string]any
	if err := json.Unmarshal(r1.GovSchema, &gov); err != nil || gov["fp"] != "082026" {
		t.Errorf("gov_schema = %s (%v)", r1.GovSchema, err)
	}

	ts = newTestServer(t, http.StatusOK, `{"data":{"month":8,"year":2026,"outward_taxable":{"taxable_value":100000,"igst":18000,"cgst":0,"sgst":0},"zero_rated":{},"nil_exempt":{},"inward_reverse_charge":{},"non_gst":{},"inter_state_unregistered":[{"place_of_supply":"07","taxable_value":5000,"igst":900}],"invoice_count":1,"credit_note_count":0},"gov_schema":{}}`)
	r3, err := ts.client.India.GSTR3B(context.Background(), &GSTReturnParams{Month: 8, Year: 2026})
	if err != nil {
		t.Fatalf("GSTR3B: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/india/gstr3b")
	if ts.query != "month=8&year=2026" {
		t.Errorf("query = %q", ts.query)
	}
	if r3.Data.OutwardTaxable.IGST != 18000 || len(r3.Data.InterStateUnregistered) != 1 || r3.Data.InterStateUnregistered[0].IGST != 900 {
		t.Errorf("gstr3b = %+v", r3.Data)
	}
}

func TestAuthMFAAndSessions(t *testing.T) {
	bg := context.Background()
	runCalls(t, []apiCall{
		{name: "mfa-setup", body: `{"secret":"JBSWY3DPEHPK3PXP","otpauth_url":"otpauth://totp/Recurso:jane?secret=JBSWY3DPEHPK3PXP"}`, method: http.MethodPost, path: "/auth/mfa/setup",
			fn: func(c *Client) (any, error) { return c.Auth.MFASetup(bg) },
			check: func(t *testing.T, got any) {
				if s := got.(*MFASetup); s.Secret != "JBSWY3DPEHPK3PXP" || !strings.HasPrefix(s.OTPAuthURL, "otpauth://") {
					t.Errorf("setup = %+v", s)
				}
			}},
		{name: "mfa-verify", body: `{"mfa_enabled":true,"backup_codes":["aaaa-bbbb","cccc-dddd"]}`, method: http.MethodPost, path: "/auth/mfa/verify",
			fn: func(c *Client) (any, error) { return c.Auth.MFAVerify(bg, "123456") },
			check: func(t *testing.T, got any) {
				if s := got.(*MFAStatus); !s.MFAEnabled || len(s.BackupCodes) != 2 {
					t.Errorf("status = %+v", s)
				}
			}},
		{name: "mfa-disable", body: `{"mfa_enabled":false}`, method: http.MethodPost, path: "/auth/mfa/disable",
			fn: func(c *Client) (any, error) { return c.Auth.MFADisable(bg, "123456") },
			check: func(t *testing.T, got any) {
				if got.(*MFAStatus).MFAEnabled {
					t.Errorf("status = %+v", got)
				}
			}},
		{name: "sessions", body: `{"data":[{"id":"ses_1","user_agent":"Mozilla/5.0","created_at":"2026-08-01T00:00:00Z","expires_at":"2026-09-01T00:00:00Z","current":true}]}`, method: http.MethodGet, path: "/auth/sessions",
			fn: func(c *Client) (any, error) { return c.Auth.Sessions(bg) },
			check: func(t *testing.T, got any) {
				if ss := got.([]Session); len(ss) != 1 || !ss[0].Current || ss[0].UserAgent != "Mozilla/5.0" {
					t.Errorf("sessions = %+v", ss)
				}
			}},
		{name: "revoke-other-sessions", body: `{"message":"2 sessions revoked"}`, method: http.MethodDelete, path: "/auth/sessions",
			fn: func(c *Client) (any, error) { return c.Auth.RevokeOtherSessions(bg) },
			check: func(t *testing.T, got any) {
				if got.(*MessageResponse).Message != "2 sessions revoked" {
					t.Errorf("message = %+v", got)
				}
			}},
		{name: "revoke-session", body: `{"message":"session revoked"}`, method: http.MethodDelete, path: "/auth/sessions/ses_2",
			fn: func(c *Client) (any, error) { return c.Auth.RevokeSession(bg, "ses_2") },
			check: func(t *testing.T, got any) {
				if got.(*MessageResponse).Message != "session revoked" {
					t.Errorf("message = %+v", got)
				}
			}},
	})
}

func TestAuthMFAVerifyBody(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"mfa_enabled":true}`)
	if _, err := ts.client.Auth.MFAVerify(context.Background(), "654321"); err != nil {
		t.Fatalf("MFAVerify: %v", err)
	}
	if ts.bodyMap()["code"] != "654321" {
		t.Errorf("body = %v", ts.bodyMap())
	}
}

func TestSSOConnection(t *testing.T) {
	bg := context.Background()
	conn := `{"data":{"tenant_id":"t_1","idp_entity_id":"https://idp.example.com","idp_sso_url":"https://idp.example.com/sso","enabled":%s,"configured":true,"sp_metadata_url":"https://api.recurso.dev/auth/saml/t_1/metadata","sp_acs_url":"https://api.recurso.dev/auth/saml/t_1/acs"}}`
	runCalls(t, []apiCall{
		{name: "get", body: fmt.Sprintf(conn, "true"), method: http.MethodGet, path: "/sso/connection",
			fn: func(c *Client) (any, error) { return c.SSO.Get(bg) },
			check: func(t *testing.T, got any) {
				if s := got.(*SSOConnection); !s.Enabled || !s.Configured || s.IdPEntityID != "https://idp.example.com" || s.SPACSURL == "" {
					t.Errorf("sso = %+v", s)
				}
			}},
		{name: "upsert", body: fmt.Sprintf(conn, "false"), method: http.MethodPut, path: "/sso/connection",
			fn: func(c *Client) (any, error) {
				return c.SSO.Upsert(bg, &SSOConnectionParams{IdPMetadataXML: "<EntityDescriptor/>", Enabled: false})
			},
			check: func(t *testing.T, got any) {
				if s := got.(*SSOConnection); s.Enabled || !s.Configured {
					t.Errorf("sso = %+v", s)
				}
			}},
		{name: "delete", body: `{"message":"SSO connection deleted"}`, method: http.MethodDelete, path: "/sso/connection",
			fn: func(c *Client) (any, error) { return c.SSO.Delete(bg) },
			check: func(t *testing.T, got any) {
				if got.(*MessageResponse).Message != "SSO connection deleted" {
					t.Errorf("message = %+v", got)
				}
			}},
	})
}

func TestBilling(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"billing_status":"trialing","plan_tier":"starter","trial_ends_at":"2026-09-15T00:00:00Z","trial_days_left":12,"trial_expired":false}`)
	st, err := ts.client.Billing.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/billing/status")
	if st.BillingStatus != "trialing" || st.TrialDaysLeft != 12 || st.TrialExpired {
		t.Errorf("status = %+v", st)
	}

	ts = newTestServer(t, http.StatusOK, `{"plans":[{"tier":"starter","name":"Starter","price":"$49","period":"month","features":["Unlimited customers"],"cta":"Start trial","recommended":false},{"tier":"growth","name":"Growth","price":"$199","period":"month","recommended":true}]}`)
	plans, err := ts.client.Billing.Plans(context.Background())
	if err != nil {
		t.Fatalf("Plans: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/billing/plans")
	if len(plans) != 2 || plans[1].Tier != "growth" || !plans[1].Recommended || len(plans[0].Features) != 1 {
		t.Errorf("plans = %+v", plans)
	}
}

func TestSystemVersion(t *testing.T) {
	// The base URL carries /v1; /version lives beside it at the root.
	ts := newTestServer(t, http.StatusOK, `{"version":"1.42.0","gateway_mode":"live"}`)
	ts.client = NewClient("test_key", WithBaseURL(ts.server.URL+"/v1"))
	v, err := ts.client.System.Version(context.Background())
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/version")
	if v.Version != "1.42.0" || v.GatewayMode != "live" {
		t.Errorf("version = %+v", v)
	}
	// Other resources still go under /v1.
	ts2 := newTestServer(t, http.StatusOK, `{"data":[]}`)
	ts2.client = NewClient("test_key", WithBaseURL(ts2.server.URL+"/v1"))
	if _, err := ts2.client.Plans.List(context.Background(), nil); err != nil {
		t.Fatalf("List: %v", err)
	}
	ts2.assertRequest(http.MethodGet, "/v1/plans")
}
