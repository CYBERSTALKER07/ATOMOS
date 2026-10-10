package partner

import (
	"net"
	"testing"
)

func TestValidateWebhookURL_HTTPSRequired(t *testing.T) {
	err := ValidateWebhookURL("http://hooks.example.com/x", WebhookURLPolicy{
		LookupIP: publicLookup,
	})
	if err == nil || err.Error() != "webhook_url_https_required" {
		t.Fatalf("got %v", err)
	}
	if err := ValidateWebhookURL("https://hooks.example.com/x", WebhookURLPolicy{
		LookupIP: publicLookup,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestValidateWebhookURL_BlocksPrivateAndMetadata(t *testing.T) {
	cases := []string{
		"https://127.0.0.1/hook",
		"https://10.0.0.5/hook",
		"https://169.254.169.254/latest",
		"https://localhost/hook",
		"https://metadata.google.internal/",
	}
	for _, u := range cases {
		if err := ValidateWebhookURL(u, WebhookURLPolicy{}); err == nil {
			t.Fatalf("expected reject for %s", u)
		}
	}
}

func TestValidateWebhookURL_DNSPrivateRejected(t *testing.T) {
	err := ValidateWebhookURL("https://evil.example/hook", WebhookURLPolicy{
		LookupIP: func(string) ([]net.IP, error) {
			return []net.IP{net.ParseIP("10.1.2.3")}, nil
		},
	})
	if err == nil || err.Error() != "webhook_url_private_ip_forbidden" {
		t.Fatalf("got %v", err)
	}
}

func TestValidateWebhookURL_AllowlistBypassesDNS(t *testing.T) {
	err := ValidateWebhookURL("https://hooks.partner.test/x", WebhookURLPolicy{
		HostAllowlist: []string{"partner.test"},
		LookupIP: func(string) ([]net.IP, error) {
			t.Fatal("lookup should not run when allowlisted")
			return nil, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateWebhookURL_AllowHTTPPrivateForSSMR(t *testing.T) {
	err := ValidateWebhookURL("http://host.docker.internal:9999/hook", WebhookURLPolicy{
		AllowHTTP:         true,
		AllowPrivateHosts: true,
		LookupIP: func(string) ([]net.IP, error) {
			return []net.IP{net.ParseIP("127.0.0.1")}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func publicLookup(string) ([]net.IP, error) {
	return []net.IP{net.ParseIP("93.184.216.34")}, nil // example.com public
}

func TestIsPartnerWebhookable_PingAndOrder(t *testing.T) {
	if !IsPartnerWebhookable(EventPartnerWebhookPing) {
		t.Fatal("ping must be subscribeable")
	}
	if !IsPartnerWebhookable("ORDER_CREATED") {
		t.Fatal("ORDER_CREATED must stay webhookable")
	}
	if IsPartnerWebhookable("NOT_A_REAL_EVENT") {
		t.Fatal("unknown events stay closed")
	}
}
