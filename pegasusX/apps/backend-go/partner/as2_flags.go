package partner

import (
	"os"
	"strings"
)

// PartnerAS2Enabled gates AS2 receive/send (default off).
func PartnerAS2Enabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("PARTNER_AS2_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// PartnerAS2InsecurePlain allows unsigned/unencrypted application/edifact bodies
// for local SSMR. Never enable in production overlays.
func PartnerAS2InsecurePlain() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("PARTNER_AS2_INSECURE_PLAIN")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}
