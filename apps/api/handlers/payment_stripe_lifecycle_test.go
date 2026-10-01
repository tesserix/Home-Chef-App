package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
	"github.com/stretchr/testify/require"
)

type stripeRefundTransport func(*http.Request) (*http.Response, error)

func (f stripeRefundTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestStripeRefundLifecycle_PartialRetryAndRemainingFull(t *testing.T) {
	for _, currency := range []string{"AUD", "NZD"} {
		t.Run(currency, func(t *testing.T) {
			db := setupPayDB(t)
			for _, column := range []string{"fulfillment_type text", "accepted_at datetime", "prepared_at datetime", "picked_up_at datetime", "delivered_at datetime", "confirmed_fulfillment_at datetime", "fulfillment_time_status text", "tax_rate real", "tax_name text", "tax_inclusive boolean", "tax_rate_food real", "tax_rate_service real", "tax_rate_delivery real", "tax_service_inclusive boolean", "delivery_address_country text"} {
				require.NoError(t, db.Exec("ALTER TABLE orders ADD COLUMN "+column).Error)
			}
			require.NoError(t, db.Exec(chefPayoutsDDL).Error)
			require.NoError(t, db.Exec(chefPenaltiesDDL).Error)
			require.NoError(t, db.Exec("ALTER TABLE order_items ADD COLUMN name text").Error)
			require.NoError(t, db.Exec("ALTER TABLE order_items ADD COLUMN price real").Error)
			country, rate, total, remainder := "AU", 10.0, 15.83, "1083"
			if currency == "NZD" {
				country, rate, total, remainder = "NZ", 15, 15.86, "1086"
			}
			pricing := models.ComputeOrderPricing(models.PricingInput{Subtotal: 15, PlatformFee: 0.75, Country: country, Rates: models.TaxRates{Name: "GST", Food: rate, FoodInclusive: true, Service: rate, Delivery: rate}})
			require.Equal(t, total, pricing.Total)

			originalConfig, originalTransport := config.AppConfig, http.DefaultTransport
			config.AppConfig = &config.Config{Environment: "test", StripeTestSecretKey: "sk_test_fixture"}
			services.InvalidateStripe()
			t.Cleanup(func() {
				config.AppConfig = originalConfig
				http.DefaultTransport = originalTransport
				services.InvalidateStripe()
			})
			var amounts, keys []string
			http.DefaultTransport = stripeRefundTransport(func(r *http.Request) (*http.Response, error) {
				if r.URL.Host != "api.stripe.com" || r.URL.Path != "/v1/refunds" {
					return nil, fmt.Errorf("unexpected provider request")
				}
				require.NoError(t, r.ParseForm())
				require.Equal(t, "true", r.Form.Get("reverse_transfer"))
				require.Equal(t, "true", r.Form.Get("refund_application_fee"))
				require.NotEmpty(t, r.Header.Get("Idempotency-Key"))
				amounts = append(amounts, r.Form.Get("amount"))
				keys = append(keys, r.Header.Get("Idempotency-Key"))
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"id":"re_%d","status":"succeeded"}`, len(amounts))))}, nil
			})
			customer := payUser(t, db, "customer")
			chefUser := payUser(t, db, "chef")
			chef := payChef(t, db, chefUser)
			order := payOrderOn(t, db, models.PaymentProviderStripe, customer, chef, "pending", total, "", "")
			require.NoError(t, db.Exec(`UPDATE chef_profiles SET payout_country=?,stripe_account_id='acct_vendor',stripe_charges_enabled=1 WHERE id=?`, country, chef).Error)
			require.NoError(t, db.Exec(`UPDATE orders SET currency=?, mode='test', fulfillment_type='pickup', subtotal=?, tax=?, tax_food=?, tax_service=?, service_fee=?, commission_rate=0.06, tax_rate=?, tax_name='GST', tax_inclusive=1, delivery_address_country=? WHERE id=?`, currency, pricing.Subtotal, pricing.Tax, pricing.TaxFood, pricing.TaxService, pricing.PlatformFee, rate, country, order).Error)
			require.NoError(t, db.Exec(`INSERT INTO order_items (id,order_id,name,price,quantity,subtotal) VALUES (?,?,?,?,?,?)`, order, order, "Test Vegetable Bowl", 15, 1, 15).Error)
			stub := &stripeGatewayStub{intent: &services.StripePaymentIntent{ID: "pi_test", Amount: services.ToMinor(total, currency), Currency: strings.ToLower(currency), Status: "requires_payment_method", ClientSecret: "fixture"}}
			create := callPay(customer, http.MethodPost, "/payments/order/"+order.String()+"/create", func(r *gin.Engine, h *PaymentHandler) { h.stripe = stub; regCreate(r, h) }, nil)
			require.Equal(t, 200, create.Code, create.Body.String())
			require.Len(t, stub.created, 1)
			require.Equal(t, services.ToMinor(total, currency), stub.created[0].Amount)
			stub.intent.Status = "succeeded"
			stub.intent.AmountReceived = stub.intent.Amount
			paid := callPay(customer, http.MethodPost, "/payments/order/"+order.String()+"/verify", func(r *gin.Engine, h *PaymentHandler) { h.stripe = stub; regVerify(r, h) }, map[string]string{"stripePaymentIntentId": "pi_test"})
			require.Equal(t, 200, paid.Code, paid.Body.String())
			require.Equal(t, "completed", paymentStatusOf(t, db, order))
			for _, status := range []string{"accepted", "preparing", "ready", "delivered"} {
				result := postStatus(t, chefUser, order, status)
				require.Equal(t, 200, result.Code, result.Body.String())
				require.Equal(t, status, orderStatus(t, db, order))
			}
			var saved models.Order
			require.NoError(t, db.Preload("Chef").First(&saved, "id = ?", order).Error)
			require.Equal(t, total, saved.ToResponse().Total)
			require.InDelta(t, total, models.RoundAmount(saved.Subtotal+saved.PlatformFee+saved.Tax), 0.001)
			var payout models.OrderChefPayout
			require.NoError(t, db.First(&payout, "order_id = ?", order).Error)
			require.Equal(t, currency, payout.Currency)
			expectedVendor := models.RoundAmount(pricing.Subtotal + pricing.TaxFood - models.RoundAmount(pricing.Subtotal*0.06))
			require.Equal(t, expectedVendor, payout.NetPayout)
			require.Equal(t, services.ToMinor(total-expectedVendor, currency), stub.created[0].ApplicationFeeCents)
			pdf, name, err := services.GenerateOrderInvoicePDF(order)
			require.NoError(t, err)
			require.True(t, strings.HasPrefix(name, "invoice-"))
			require.True(t, strings.HasPrefix(string(pdf), "%PDF"))
			if dir := os.Getenv("STRIPE_LIFECYCLE_PDF_DIR"); dir != "" {
				require.NoError(t, os.WriteFile(filepath.Join(dir, country+"-invoice.pdf"), pdf, 0600))
			}

			path := "/payments/order/" + order.String() + "/refund"
			partial := map[string]any{"amount": 5.0, "reason": "partial quality refund"}
			response := callPay(chefUser, http.MethodPost, path, regRefund, partial)
			require.Equal(t, 200, response.Code, response.Body.String())
			state := loadRefundState(t, db, order)
			require.Equal(t, 5.0, state.RefundAmount)
			require.Equal(t, "completed", state.PaymentStatus)
			require.Equal(t, "delivered", state.Status)
			require.False(t, state.RefundedAt.Valid)
			response = callPay(chefUser, http.MethodPost, path, regRefund, partial)
			require.Equal(t, 200, response.Code, response.Body.String())
			require.Len(t, amounts, 1, "retry of successful partial must not refund twice")
			response = callPay(chefUser, http.MethodPost, path, regRefund, map[string]any{"amount": 16.0, "reason": "too much"})
			require.Equal(t, 400, response.Code, response.Body.String())
			require.Len(t, amounts, 1)
			response = callPay(chefUser, http.MethodPost, path, regRefund, map[string]any{"amount": 0.0, "reason": "refund remaining"})
			require.Equal(t, 200, response.Code, response.Body.String())
			require.Equal(t, []string{"500", remainder}, amounts)
			require.NotEqual(t, keys[0], keys[1])
			state = loadRefundState(t, db, order)
			require.Equal(t, total, state.RefundAmount)
			require.Equal(t, "refunded", state.PaymentStatus)
			require.Equal(t, "refunded", state.Status)
			require.True(t, state.RefundedAt.Valid)
			require.NoError(t, db.First(&saved, "id = ?", order).Error)
			for _, displayed := range []*models.ChefPayoutResponse{services.ChefPayoutFor(db, &saved), services.ChefPayoutsFor(db, []models.Order{saved})[order]} {
				require.Zero(t, displayed.NetPayout)
				require.Equal(t, models.ChefPayoutReversed, displayed.Status)
			}

			pdf, name, err = services.GenerateOrderInvoicePDF(order)
			require.NoError(t, err)
			require.True(t, strings.HasPrefix(name, "receipt-"))
			if dir := os.Getenv("STRIPE_LIFECYCLE_PDF_DIR"); dir != "" {
				require.NoError(t, os.WriteFile(filepath.Join(dir, country+"-refund-receipt.pdf"), pdf, 0600))
			}
			response = callPay(chefUser, http.MethodPost, path, regRefund, map[string]any{"amount": 1.0, "reason": "after full"})
			require.Equal(t, 400, response.Code, response.Body.String())
			require.Len(t, amounts, 2)
		})
	}
}
