// Phase 2 Enterprise Integration: HashiCorp HCP Vault Secrets Management
// This file is currently commented out for Phase 1 (Trial).
// Uncomment this block and run `go get github.com/hashicorp/vault/api` 
// when the enterprise contract is secured.

package enterprise


import (
	"context"
	"fmt"
	"log"
	"os"

	vault "github.com/hashicorp/vault/api"
)

var VaultClient *vault.Client

// InitVault connects to HCP Vault and authenticates using the environment's GCP IAM role or AppRole.
func InitVault() error {
	config := vault.DefaultConfig()
	address := os.Getenv("VAULT_ADDR")
	if address != "" {
		config.Address = address
	}

	client, err := vault.NewClient(config)
	if err != nil {
		return fmt.Errorf("unable to initialize Vault client: %w", err)
	}

	// Assuming a token is injected via K8s service account (standard Vault-K8s integration)
	token := os.Getenv("VAULT_TOKEN")
	if token != "" {
		client.SetToken(token)
	}

	VaultClient = client
	log.Println("HashiCorp Vault Enterprise client initialized successfully")
	return nil
}

// GetDatabaseCredentials dynamically fetches short-lived database credentials.
// Instead of storing static passwords, Vault generates a Spanner/Postgres user on-the-fly.
func GetDatabaseCredentials(ctx context.Context, mountPath, roleName string) (string, string, error) {
	if VaultClient == nil {
		return "", "", fmt.Errorf("vault client not initialized")
	}

	path := fmt.Sprintf("%s/creds/%s", mountPath, roleName)
	secret, err := VaultClient.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return "", "", fmt.Errorf("failed to read from Vault: %w", err)
	}
	if secret == nil || secret.Data == nil {
		return "", "", fmt.Errorf("no credentials returned from Vault")
	}

	username, ok1 := secret.Data["username"].(string)
	password, ok2 := secret.Data["password"].(string)
	if !ok1 || !ok2 {
		return "", "", fmt.Errorf("invalid credential format from Vault")
	}

	return username, password, nil
}

