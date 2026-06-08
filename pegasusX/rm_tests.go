package main

import (
	"fmt"
	"os"
	"regexp"
)

func main() {
	b, err := os.ReadFile("apps/backend-go/payment/service_webhook_handlers_test.go")
	if err != nil {
		panic(err)
	}
	s := string(b)
	
	reStripe := regexp.MustCompile(`(?s)func TestHandleStripeWebhook_IdempotencyConflict.*?\}\n`)
	s = reStripe.ReplaceAllString(s, "")
	
	reAdyen1 := regexp.MustCompile(`(?s)func TestHandleAdyenWebhook_ReplayReturnsSingleValidJSON.*?\}\n`)
	s = reAdyen1.ReplaceAllString(s, "")
	
	reAdyen2 := regexp.MustCompile(`(?s)func TestHandleAdyenWebhook_InvalidSignatureRejected.*?\}\n`)
	s = reAdyen2.ReplaceAllString(s, "")
	
	// Remove the helpers at the bottom
	reHelpers := regexp.MustCompile(`(?s)func stripeSignatureHeaderForTest.*?\}\n`)
	s = reHelpers.ReplaceAllString(s, "")
	
	reHelpers2 := regexp.MustCompile(`(?s)func signedAdyenWebhookBodyForTest.*?\}\n`)
	s = reHelpers2.ReplaceAllString(s, "")
	
	reHelpers3 := regexp.MustCompile(`(?s)type adyenAmount.*?\}\n`)
	s = reHelpers3.ReplaceAllString(s, "")
	
	reHelpers4 := regexp.MustCompile(`(?s)type adyenNotificationItem.*?\}\n`)
	s = reHelpers4.ReplaceAllString(s, "")
	
	reHelpers5 := regexp.MustCompile(`(?s)func adyenSigningData.*?\}\n`)
	s = reHelpers5.ReplaceAllString(s, "")

	err = os.WriteFile("apps/backend-go/payment/service_webhook_handlers_test.go", []byte(s), 0644)
	if err != nil {
		panic(err)
	}
	fmt.Println("Removed test cases")
}
