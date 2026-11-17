package bootstrap

import (
	"log/slog"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// newLoginFirebaseVerifier builds the ID-token verifier for role login handlers.
// Flag off or empty project → nil (password/PIN login still works).
func newLoginFirebaseVerifier(cfg *Config, log *slog.Logger) auth.FirebaseVerifier {
	if cfg == nil || !cfg.FirebaseAuthEnabled {
		return nil
	}
	projectID := strings.TrimSpace(cfg.FirebaseProjectID)
	if projectID == "" {
		if log != nil {
			log.Warn("firebase auth enabled but FIREBASE_PROJECT_ID is empty")
		}
		return nil
	}
	v := auth.NewFirebaseTokenVerifier(projectID, auth.FirebaseVerifierOptionsForProject(cfg.FirebaseCertsURL))
	if log != nil {
		log.Info("firebase auth verifier initialized", "project_id", projectID)
	}
	return v
}
