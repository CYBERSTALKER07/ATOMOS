package fiscal

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Signer selection for FISCAL_PROVIDER=MY_SOLIQ.
//
//	FISCAL_MY_SOLIQ_SIGNER=dev-hmac   HMAC-SHA256 stand-in for EDS, NON-PROD ONLY
//	                                  (contract tests + SSMR; key: FISCAL_MY_SOLIQ_SIGN_KEY)
//	FISCAL_MY_SOLIQ_SIGNER=pkcs12     real E-IMZO PKCS#12 key — owner task, blocked
//	                                  until Soliq/EDS procurement lands (hard error today)
//
// Fail-closed: any other value, or dev-hmac in production, is a construction error —
// a misconfigured fiscal path must never silently issue unsigned receipts.

// DevHMACSigner signs canonical payloads with HMAC-SHA256. It is NOT a legal EDS
// signature; it exists so the full sign→submit→verify chain is exercisable in
// tests and SSMR before E-IMZO credentials arrive.
type DevHMACSigner struct {
	key []byte
}

func NewDevHMACSigner(key []byte) (*DevHMACSigner, error) {
	if len(key) < 16 {
		return nil, fmt.Errorf("dev-hmac signer key too short (need >= 16 bytes)")
	}
	return &DevHMACSigner{key: key}, nil
}

func (s *DevHMACSigner) Sign(_ context.Context, payload []byte) ([]byte, error) {
	mac := hmac.New(sha256.New, s.key)
	mac.Write(payload)
	return []byte("DEVHMAC." + base64.StdEncoding.EncodeToString(mac.Sum(nil))), nil
}

// Verify recomputes the HMAC — used by contract tests to prove the signed
// payload round-trips.
func (s *DevHMACSigner) Verify(payload []byte, signature string) bool {
	signed, err := s.Sign(context.Background(), payload)
	if err != nil {
		return false
	}
	return string(signed) == signature
}

// SignerFromEnv builds the EDS signer for the MY_SOLIQ provider.
// envName is the deployment environment (PEGASUSX_ENV); production and sandbox
// reject the dev-hmac signer.
func SignerFromEnv(envName string) (EDSSigner, error) {
	kind := strings.ToLower(strings.TrimSpace(os.Getenv("FISCAL_MY_SOLIQ_SIGNER")))
	switch kind {
	case "dev-hmac":
		switch auth.EnvClassFrom(envName) {
		case auth.EnvClassProduction, auth.EnvClassSandbox:
			return nil, fmt.Errorf("FISCAL_MY_SOLIQ_SIGNER=dev-hmac is forbidden in %s", envName)
		}
		key := strings.TrimSpace(os.Getenv("FISCAL_MY_SOLIQ_SIGN_KEY"))
		if key == "" {
			return nil, fmt.Errorf("FISCAL_MY_SOLIQ_SIGN_KEY required for FISCAL_MY_SOLIQ_SIGNER=dev-hmac")
		}
		return NewDevHMACSigner([]byte(key))
	case "pkcs12", "eimzo", "e-imzo":
		file := strings.TrimSpace(os.Getenv("FISCAL_MY_SOLIQ_PKCS12_FILE"))
		password := os.Getenv("FISCAL_MY_SOLIQ_PKCS12_PASSWORD")
		if file == "" {
			return nil, fmt.Errorf("FISCAL_MY_SOLIQ_PKCS12_FILE required for FISCAL_MY_SOLIQ_SIGNER=%s (path to the E-IMZO .p12 container — EDS key procurement owner task)", kind)
		}
		return NewPKCS12SignerFromFile(file, password)
	case "":
		return nil, fmt.Errorf("FISCAL_MY_SOLIQ_SIGNER required when FISCAL_PROVIDER=MY_SOLIQ (dev-hmac for non-prod, pkcs12 once EDS key is procured)")
	default:
		return nil, fmt.Errorf("unknown FISCAL_MY_SOLIQ_SIGNER %q", kind)
	}
}
