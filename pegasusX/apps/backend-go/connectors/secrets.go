package connectors

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// ValidateSecretRefForTenant ensures secretRef points only to a CONNECTOR_SECRET_* env
// variable, a safe logical slug under CONNECTOR_SECRET_*, or a scoped connector GCP secret.
func ValidateSecretRefForTenant(tenantID, secretRef string) error {
	ref := strings.TrimSpace(secretRef)
	if ref == "" {
		return fmt.Errorf("%w: empty secret_ref", ErrInvalidInput)
	}
	if strings.ContainsRune(ref, 0) || strings.Contains(ref, "..") || strings.Contains(ref, "\\") {
		return fmt.Errorf("%w: invalid secret_ref characters", ErrInvalidInput)
	}

	if strings.HasPrefix(ref, "env://") {
		envName := strings.TrimSpace(strings.TrimPrefix(ref, "env://"))
		if !isEnvIdentifier(envName) || !strings.HasPrefix(envName, "CONNECTOR_SECRET_") {
			return fmt.Errorf("%w: env:// secret_ref must use CONNECTOR_SECRET_* prefix", ErrInvalidInput)
		}
		return nil
	}

	if strings.HasPrefix(ref, "gcp-secret://") || strings.HasPrefix(ref, "projects/") {
		resource := strings.TrimSpace(strings.TrimPrefix(ref, "gcp-secret://"))
		if _, err := parseAndValidateGCPSecretName(tenantID, resource); err != nil {
			return err
		}
		return nil
	}

	if strings.ContainsAny(ref, ":/ \t\r\n") {
		return fmt.Errorf("%w: unsupported secret_ref scheme", ErrInvalidInput)
	}
	if sanitizeEnvKey(ref) == "" {
		return fmt.Errorf("%w: invalid secret_ref slug", ErrInvalidInput)
	}
	return nil
}

func sanitizeGCPSlug(raw string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(raw)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func parseAndValidateGCPSecretName(tenantID, resource string) (string, error) {
	parts := strings.Split(strings.TrimSpace(resource), "/")
	// Accepted shapes: projects/<proj>/secrets/<name> or projects/<proj>/secrets/<name>/versions/<ver>
	validLen := len(parts) == 4 || len(parts) == 6
	if !validLen || parts[0] != "projects" || parts[2] != "secrets" || parts[1] == "" || parts[3] == "" {
		return "", fmt.Errorf("%w: invalid gcp secret_ref resource path", ErrInvalidInput)
	}
	if len(parts) == 6 && (parts[4] != "versions" || parts[5] == "") {
		return "", fmt.Errorf("%w: invalid gcp secret_ref version path", ErrInvalidInput)
	}
	for _, seg := range parts {
		for _, r := range seg {
			ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.'
			if !ok {
				return "", fmt.Errorf("%w: invalid gcp secret_ref segment", ErrInvalidInput)
			}
		}
	}
	secretName := strings.ToLower(parts[3])
	if !strings.HasPrefix(secretName, "connector-") {
		return "", fmt.Errorf("%w: gcp secret name must begin with connector-", ErrInvalidInput)
	}
	tid := sanitizeGCPSlug(tenantID)
	if tid != "" && !strings.HasPrefix(secretName, "connector-"+tid+"-") {
		return "", fmt.Errorf("%w: gcp secret_ref is outside tenant scope", ErrInvalidInput)
	}
	return resource, nil
}

// GCPSecretFetcher resolves a GCP Secret Manager resource name or short secret reference.
type GCPSecretFetcher func(ctx context.Context, secretName string) (string, error)

// SecretResolver resolves SecretRef handles into runtime credentials without storing raw secrets.
type SecretResolver interface {
	Resolve(ctx context.Context, secretRef string) (string, error)
}

// EnvSecretResolver resolves env://..., CONNECTOR_SECRET_*, and GCP Secret Manager handles.
// It fails closed when a secret cannot be resolved, especially in PEGASUSX_ENV=production.
type EnvSecretResolver struct {
	Getenv       func(string) string
	GCPFetcher   GCPSecretFetcher
	IsProduction func() bool
}

// NewDefaultSecretResolver constructs the production-safe SecretResolver backed by os.Getenv
// and optional GCP Secret Manager hook.
func NewDefaultSecretResolver(gcpFetcher GCPSecretFetcher) *EnvSecretResolver {
	return &EnvSecretResolver{
		Getenv:       os.Getenv,
		GCPFetcher:   gcpFetcher,
		IsProduction: auth.IsProduction,
	}
}

func (r *EnvSecretResolver) getenv(key string) string {
	if r != nil && r.Getenv != nil {
		return strings.TrimSpace(r.Getenv(key))
	}
	return strings.TrimSpace(os.Getenv(key))
}

func (r *EnvSecretResolver) isProd() bool {
	if r != nil && r.IsProduction != nil {
		return r.IsProduction()
	}
	return auth.IsProduction()
}

// Resolve resolves secretRef to its credential material, failing closed on missing or unsafe refs.
func (r *EnvSecretResolver) Resolve(ctx context.Context, secretRef string) (string, error) {
	return r.ResolveForTenant(ctx, "", secretRef)
}

// ResolveForTenant resolves secretRef with tenant scoping enforced on GCP Secret Manager references.
func (r *EnvSecretResolver) ResolveForTenant(ctx context.Context, tenantID, secretRef string) (string, error) {
	ref := strings.TrimSpace(secretRef)
	if ref == "" {
		return "", fmt.Errorf("%w: empty secret_ref", ErrConnectorNotConfigured)
	}
	if err := ValidateSecretRefForTenant(tenantID, ref); err != nil {
		return "", fmt.Errorf("%w: %v", ErrConnectorNotConfigured, err)
	}

	if strings.HasPrefix(ref, "env://") {
		envName := strings.TrimSpace(strings.TrimPrefix(ref, "env://"))
		val := r.getenv(envName)
		if val == "" {
			return "", fmt.Errorf("%w: secret %s is not set", ErrConnectorNotConfigured, MaskSecretRef(ref))
		}
		return val, nil
	}

	if strings.HasPrefix(ref, "gcp-secret://") || strings.HasPrefix(ref, "projects/") {
		secretName := strings.TrimSpace(strings.TrimPrefix(ref, "gcp-secret://"))
		if r != nil && r.GCPFetcher != nil {
			val, err := r.GCPFetcher(ctx, secretName)
			if err != nil {
				return "", fmt.Errorf("%w: gcp secret %s failed: %v", ErrConnectorNotConfigured, MaskSecretRef(ref), err)
			}
			val = strings.TrimSpace(val)
			if val == "" {
				return "", fmt.Errorf("%w: gcp secret %s resolved empty", ErrConnectorNotConfigured, MaskSecretRef(ref))
			}
			return val, nil
		}
		if !r.isProd() {
			if val := r.getenv("CONNECTOR_SECRET_" + sanitizeEnvKey(secretName)); val != "" {
				return val, nil
			}
		}
		return "", fmt.Errorf("%w: gcp secret manager hook not configured for %s", ErrConnectorNotConfigured, MaskSecretRef(ref))
	}

	if strings.HasPrefix(strings.ToUpper(ref), "CONNECTOR_SECRET_") {
		if val := r.getenv(ref); val != "" {
			return val, nil
		}
		if val := r.getenv(strings.ToUpper(ref)); val != "" {
			return val, nil
		}
		return "", fmt.Errorf("%w: secret %s is not set", ErrConnectorNotConfigured, MaskSecretRef(ref))
	}

	if val := r.getenv("CONNECTOR_SECRET_" + sanitizeEnvKey(ref)); val != "" {
		return val, nil
	}
	return "", fmt.Errorf("%w: secret %s could not be resolved", ErrConnectorNotConfigured, MaskSecretRef(ref))
}

// MaskSecretRef redacts a secret reference or credential handle for safe logging and API responses.
func MaskSecretRef(secretRef string) string {
	ref := strings.TrimSpace(secretRef)
	if ref == "" {
		return ""
	}
	for _, scheme := range []string{"env://", "gcp-secret://"} {
		if strings.HasPrefix(ref, scheme) {
			tail := strings.TrimPrefix(ref, scheme)
			return scheme + maskTail(tail)
		}
	}
	return maskTail(ref)
}

func maskTail(val string) string {
	val = strings.TrimSpace(val)
	if val == "" {
		return ""
	}
	if len(val) <= 4 {
		return "****"
	}
	return "****" + val[len(val)-4:]
}

func sanitizeEnvKey(raw string) string {
	var b strings.Builder
	b.Grow(len(raw))
	lastUnderscore := false
	for _, r := range strings.ToUpper(strings.TrimSpace(raw)) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(b.String(), "_")
}

func isEnvIdentifier(raw string) bool {
	if raw == "" {
		return false
	}
	for _, r := range raw {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return true
}

// scrubSecrets removes resolved secrets from error strings before persistence or logging.
func scrubSecrets(msg string, secrets ...string) string {
	out := msg
	for _, sec := range secrets {
		s := strings.TrimSpace(sec)
		if s != "" {
			out = strings.ReplaceAll(out, s, "[REDACTED]")
		}
	}
	return out
}
