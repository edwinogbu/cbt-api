package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/webhook"
)

// ---- Webhook signature verification must fail closed, not forgeable ----
//
// All three gateway secrets (STRIPE_WEBHOOK_SECRET, PAYSTACK_SECRET_KEY,
// FLUTTERWAVE_SECRET_KEY) are read from the environment with no fallback
// default. Before this fix, an unset secret meant ParseWebhook still ran
// its HMAC comparison - but HMAC with an empty key is just a
// deterministic function of the payload, so anyone could compute it
// themselves and have a self-forged "payment succeeded" webhook treated
// as authentic, activating someone else's pending onboarding
// subscription for free. Each gateway now refuses to verify at all when
// its secret is unconfigured, failing closed instead of silently
// verifying against a forgeable key.

func TestPaystackParseWebhook_EmptySecretRejectsEvenASelfForgedSignature(t *testing.T) {
	g := &PaystackGateway{secretKey: ""}
	payload := []byte(`{"event":"charge.success","data":{"reference":"victim-ref","amount":100}}`)
	forgedSignature := computeHMAC512(payload, "") // exactly what an attacker could compute themselves

	_, err := g.ParseWebhook(context.Background(), payload, forgedSignature)
	if err == nil {
		t.Fatal("expected an unconfigured secret to reject the webhook, got nil error")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Fatalf(`expected a "not configured" error, got: %v`, err)
	}
}

func TestPaystackParseWebhook_ValidSignatureStillVerifiesWithRealSecret(t *testing.T) {
	g := &PaystackGateway{secretKey: "real-paystack-secret"}
	payload := []byte(`{"event":"charge.success","data":{"reference":"real-ref","amount":250000}}`)
	validSignature := computeHMAC512(payload, "real-paystack-secret")

	event, err := g.ParseWebhook(context.Background(), payload, validSignature)
	if err != nil {
		t.Fatalf("expected a validly-signed webhook to verify, got error: %v", err)
	}
	if event.Reference != "real-ref" || event.Type != "payment_success" {
		t.Fatalf("expected the real-ref payment_success event, got: %+v", event)
	}
}

func TestPaystackParseWebhook_WrongSignatureRejected(t *testing.T) {
	g := &PaystackGateway{secretKey: "real-paystack-secret"}
	payload := []byte(`{"event":"charge.success","data":{"reference":"real-ref","amount":250000}}`)

	_, err := g.ParseWebhook(context.Background(), payload, "not-the-right-signature")
	if err == nil {
		t.Fatal("expected a mismatched signature to be rejected, got nil error")
	}
}

func TestFlutterwaveParseWebhook_EmptySecretRejectsEvenASelfForgedSignature(t *testing.T) {
	g := &FlutterwaveGateway{secretKey: ""}
	payload := []byte(`{"event":"charge.completed","data":{"tx_ref":"victim-ref","amount":100,"currency":"NGN"}}`)
	forgedSignature := computeHMAC256(payload, "") // exactly what an attacker could compute themselves

	_, err := g.ParseWebhook(context.Background(), payload, forgedSignature)
	if err == nil {
		t.Fatal("expected an unconfigured secret to reject the webhook, got nil error")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Fatalf(`expected a "not configured" error, got: %v`, err)
	}
}

func TestFlutterwaveParseWebhook_ValidSignatureStillVerifiesWithRealSecret(t *testing.T) {
	g := &FlutterwaveGateway{secretKey: "real-flutterwave-secret"}
	payload := []byte(`{"event":"charge.completed","data":{"tx_ref":"real-ref","amount":5000,"currency":"NGN"}}`)
	validSignature := computeHMAC256(payload, "real-flutterwave-secret")

	event, err := g.ParseWebhook(context.Background(), payload, validSignature)
	if err != nil {
		t.Fatalf("expected a validly-signed webhook to verify, got error: %v", err)
	}
	if event.Reference != "real-ref" || event.Type != "payment_success" {
		t.Fatalf("expected the real-ref payment_success event, got: %+v", event)
	}
}

func TestStripeParseWebhook_EmptySecretRejectsEvenASelfForgedSignature(t *testing.T) {
	g := &StripeGateway{webhookSecret: ""}
	payload := []byte(`{"id":"evt_test","type":"checkout.session.completed"}`)
	ts := time.Now()
	forgedSig := webhook.ComputeSignature(ts, payload, "") // exactly what an attacker could compute themselves
	header := fmt.Sprintf("t=%d,v1=%s", ts.Unix(), hex.EncodeToString(forgedSig))

	_, err := g.ParseWebhook(context.Background(), payload, header)
	if err == nil {
		t.Fatal("expected an unconfigured secret to reject the webhook, got nil error")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Fatalf(`expected a "not configured" error, got: %v`, err)
	}
}

func TestStripeParseWebhook_ValidSignatureStillVerifiesWithRealSecret(t *testing.T) {
	secret := "whsec_real_test_secret"
	g := &StripeGateway{webhookSecret: secret}

	payload, err := json.Marshal(map[string]interface{}{
		"id":          "evt_test",
		"type":        "checkout.session.completed",
		"api_version": stripe.APIVersion,
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":     "cs_test_real_session",
				"object": "checkout.session",
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to build test payload: %v", err)
	}

	ts := time.Now()
	sig := webhook.ComputeSignature(ts, payload, secret)
	header := fmt.Sprintf("t=%d,v1=%s", ts.Unix(), hex.EncodeToString(sig))

	event, err := g.ParseWebhook(context.Background(), payload, header)
	if err != nil {
		t.Fatalf("expected a validly-signed webhook to verify, got error: %v", err)
	}
	if event.Reference != "cs_test_real_session" || event.Type != "payment_success" {
		t.Fatalf("expected the real checkout session's payment_success event, got: %+v", event)
	}
}

// sanity-check that the HMAC helpers used above actually agree with
// crypto/hmac so a future refactor of computeHMAC512/computeHMAC256
// can't silently defeat these tests.
func TestHMACHelpersMatchStdlib(t *testing.T) {
	payload := []byte("sanity-check-payload")
	secret := "sanity-check-secret"

	h512 := hmac.New(sha512.New, []byte(secret))
	h512.Write(payload)
	if got, want := computeHMAC512(payload, secret), hex.EncodeToString(h512.Sum(nil)); got != want {
		t.Fatalf("computeHMAC512 mismatch: got %s want %s", got, want)
	}

	h256 := hmac.New(sha256.New, []byte(secret))
	h256.Write(payload)
	if got, want := computeHMAC256(payload, secret), hex.EncodeToString(h256.Sum(nil)); got != want {
		t.Fatalf("computeHMAC256 mismatch: got %s want %s", got, want)
	}
}
