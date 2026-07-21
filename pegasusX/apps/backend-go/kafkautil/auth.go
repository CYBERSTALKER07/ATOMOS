package kafkautil

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Auth modes for cloud vs local Docker Kafka.
const (
	AuthModePlaintext     = ""                 // local SSMR
	AuthModeNone          = "NONE"             // alias
	AuthModeGCPManaged    = "GCP_MANAGED_OAUTH" // GCP Managed Kafka (SASL_SSL + access token as PLAIN)
	AuthModeGCPManagedAlt = "GCP_MANAGED"
)

// ClientAuth configures TLS/SASL for producers and consumers.
type ClientAuth struct {
	// Mode: empty/NONE = plaintext; GCP_MANAGED_OAUTH = GCP Managed Kafka.
	Mode string
	// Username: service account email for SASL PLAIN (required for GCP Managed).
	// Defaults to KAFKA_SASL_USERNAME or GOOGLE_SERVICE_ACCOUNT env.
	Username string
}

// Dialer builds a kafka-go Dialer with optional GCP Managed Kafka auth.
func Dialer(auth ClientAuth) (*kafka.Dialer, error) {
	d := &kafka.Dialer{
		Timeout:   15 * time.Second,
		DualStack: true,
	}
	mech, tlsCfg, err := mechanismAndTLS(auth)
	if err != nil {
		return nil, err
	}
	if mech != nil {
		d.SASLMechanism = mech
	}
	if tlsCfg != nil {
		d.TLS = tlsCfg
	}
	return d, nil
}

// Transport builds a shared kafka-go Transport for Writers.
func Transport(auth ClientAuth) (*kafka.Transport, error) {
	mech, tlsCfg, err := mechanismAndTLS(auth)
	if err != nil {
		return nil, err
	}
	t := &kafka.Transport{}
	if mech != nil {
		t.SASL = mech
	}
	if tlsCfg != nil {
		t.TLS = tlsCfg
	}
	return t, nil
}

func mechanismAndTLS(auth ClientAuth) (sasl.Mechanism, *tls.Config, error) {
	mode := strings.ToUpper(strings.TrimSpace(auth.Mode))
	switch mode {
	case AuthModePlaintext, AuthModeNone, "PLAINTEXT":
		return nil, nil, nil
	case AuthModeGCPManaged, AuthModeGCPManagedAlt, "SASL_SSL_GCP", "GCP":
		user := strings.TrimSpace(auth.Username)
		if user == "" {
			user = strings.TrimSpace(os.Getenv("KAFKA_SASL_USERNAME"))
		}
		if user == "" {
			user = strings.TrimSpace(os.Getenv("GOOGLE_SERVICE_ACCOUNT"))
		}
		if user == "" {
			return nil, nil, fmt.Errorf("kafka GCP auth: set ClientAuth.Username or KAFKA_SASL_USERNAME to the service account email")
		}
		mech, err := newGCPAccessTokenPLAIN(user)
		if err != nil {
			return nil, nil, err
		}
		return mech, &tls.Config{MinVersion: tls.VersionTLS12}, nil
	default:
		return nil, nil, fmt.Errorf("kafka auth: unknown mode %q", auth.Mode)
	}
}

// gcpAccessTokenPLAIN implements SASL PLAIN where password is a Google OAuth access token.
// Documented by Google for Managed Service for Apache Kafka:
// username = principal email, password = access token.
type gcpAccessTokenPLAIN struct {
	username string
	ts       oauth2.TokenSource
}

func newGCPAccessTokenPLAIN(username string) (sasl.Mechanism, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, fmt.Errorf("kafka GCP auth: find default credentials: %w", err)
	}
	if creds.TokenSource == nil {
		return nil, fmt.Errorf("kafka GCP auth: nil token source")
	}
	return &gcpAccessTokenPLAIN{
		username: username,
		ts:       oauth2.ReuseTokenSource(nil, creds.TokenSource),
	}, nil
}

func (m *gcpAccessTokenPLAIN) Name() string { return "PLAIN" }

func (m *gcpAccessTokenPLAIN) Start(ctx context.Context) (session sasl.StateMachine, ir []byte, err error) {
	tok, err := m.ts.Token()
	if err != nil {
		return nil, nil, fmt.Errorf("kafka GCP auth: token: %w", err)
	}
	if tok == nil || tok.AccessToken == "" {
		return nil, nil, fmt.Errorf("kafka GCP auth: empty access token")
	}
	// PLAIN: \0username\0password
	ir = []byte(fmt.Sprintf("\x00%s\x00%s", m.username, tok.AccessToken))
	return m, ir, nil
}

func (m *gcpAccessTokenPLAIN) Next(ctx context.Context, challenge []byte) (done bool, response []byte, err error) {
	return true, nil, nil
}

// SplitBrokers parses a comma-separated broker list.
func SplitBrokers(brokersCSV string) []string {
	parts := strings.Split(brokersCSV, ",")
	brokers := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			brokers = append(brokers, trimmed)
		}
	}
	return brokers
}
