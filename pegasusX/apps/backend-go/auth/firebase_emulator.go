package auth

import (
	"os"
	"strings"
)

// FirebaseVerifierOptionsForProject returns verifier options, routing cert fetches to the
// Auth emulator when FIREBASE_AUTH_EMULATOR_HOST is set (local portal OTP dev).
func FirebaseVerifierOptionsForProject(certsURL string) FirebaseTokenVerifierOptions {
	opts := FirebaseTokenVerifierOptions{CertsURL: certsURL}
	host := strings.TrimSpace(os.Getenv("FIREBASE_AUTH_EMULATOR_HOST"))
	if host == "" {
		return opts
	}
	host = strings.TrimPrefix(strings.TrimPrefix(host, "http://"), "https://")
	opts.CertsURL = "http://" + host + "/www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com"
	return opts
}
