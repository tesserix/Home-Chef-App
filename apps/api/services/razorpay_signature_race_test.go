package services

// razorpay_signature_race_test.go — #395·5. VerifyPaymentSignature and
// VerifyWebhookSignature read the package-global razorpayClient; GetRazorpay (cache
// refresh after TTL), InvalidateRazorpay, and the test SetRazorpayClient all WRITE
// it. The signature readers must snapshot the pointer under razorpayMu, or a payment
// verify racing a credential refresh is a data race. This test trips `go test -race`
// unless the readers lock. (NOTE: the HomeChef API CI runs `go test` WITHOUT -race,
// so this guard only bites under a local `go test -race ./services/...`.)

import (
	"sync"
	"testing"
)

func TestSignatureReaders_NoRaceWithClientSwap(t *testing.T) {
	t.Cleanup(func() { SetRazorpayClient(nil) })

	var readers sync.WaitGroup
	var writer sync.WaitGroup
	stop := make(chan struct{})

	// Writer: continuously swap the global client (models GetRazorpay's post-TTL
	// refresh / InvalidateRazorpay) until the readers finish.
	writer.Add(1)
	go func() {
		defer writer.Done()
		for {
			select {
			case <-stop:
				return
			default:
				SetRazorpayClient(NewRazorpayTestClient("", "key", "secret", "whsec"))
			}
		}
	}()

	// Readers: verify signatures concurrently. Under -race, an unlocked read of the
	// global here vs the writer above is flagged.
	for i := 0; i < 8; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for j := 0; j < 500; j++ {
				_ = VerifyPaymentSignature("order_x", "pay_x", "sig")
				_ = VerifyWebhookSignature([]byte("payload"), "sig")
			}
		}()
	}

	readers.Wait() // readers did their fixed work with the writer swapping alongside
	close(stop)    // now let the writer exit
	writer.Wait()
}
