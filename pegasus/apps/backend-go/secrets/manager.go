// Package secrets provides a unified secret-retrieval interface.
// In production: reads from GCP Secret Manager.
// In development: falls back to environment variables.
//
// The package auto-detects the environment:
//   - If GOOGLE_APPLICATION_CREDENTIALS is set and GCP_PROJECT_ID is available, uses Secret Manager.
//   - Otherwise, reads from os.Getenv.
package secrets

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// Manager provides secret retrieval with automatic GCP Secret Manager / ENV fallback.
type Manager struct {
	client    *secretmanager.Client
	projectID string
	cache     map[string]cachedSecret
	mu        sync.RWMutex
	cacheTTL  time.Duration
}

type cachedSecret struct {
	value     string
	fetchedAt time.Time
}

var (
	globalManager *Manager
	initOnce      sync.Once
)

// Init boots the secret manager. Call once at startup.
// Returns a Manager that can be used directly, or use the package-level Get() function.
func Init(ctx context.Context) *Manager {
	initOnce.Do(func() {
		globalManager = &Manager{
			cache:    make(map[string]cachedSecret),
			cacheTTL: 5 * time.Minute,
		}

		projectID := os.Getenv("GCP_PROJECT_ID")
		if projectID == "" {
			projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
		}

		// Only attempt GCP Secret Manager if we have a project ID and credentials
		if projectID != "" && os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
			client, err := secretmanager.NewClient(ctx)
			if err != nil {
				log.Printf("[SECRETS] GCP Secret Manager init failed (%v) — falling back to ENV vars", err)
			} else {
				globalManager.client = client
				globalManager.projectID = projectID
				return
			}
		}
	})
	return globalManager
}

// Get retrieves a secret by name.
//   - If GCP Secret Manager is configured, fetches from there (latest version).
//   - Falls back to os.Getenv(name).
//   - Results are cached for cacheTTL to avoid repeated API calls.
func Get(name string) string {
	if globalManager == nil {
		return os.Getenv(name)
	}
	return globalManager.GetSecret(name)
}

// GetSecret retrieves a secret by name from this Manager instance.
func (m *Manager) GetSecret(name string) string {
	// Check cache first
	m.mu.RLock()
	if cached, ok := m.cache[name]; ok && time.Since(cached.fetchedAt) < m.cacheTTL {
		m.mu.RUnlock()
		return cached.value
	}
	m.mu.RUnlock()

	// Try GCP Secret Manager
	if m.client != nil && m.projectID != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		secretName := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", m.projectID, name)
		result, err := m.client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
			Name: secretName,
		})
		if err == nil {
			value := string(result.Payload.Data)
			m.cacheSet(name, value)
			return value
		}
		log.Printf("[SECRETS] GCP fetch failed for %s (%v) — falling back to ENV", name, err)
	}

	// Fallback to ENV
	value := os.Getenv(name)
	if value != "" {
		m.cacheSet(name, value)
	}
	return value
}

func (m *Manager) cacheSet(name, value string) {
	m.mu.Lock()
	m.cache[name] = cachedSecret{value: value, fetchedAt: time.Now()}
	m.mu.Unlock()
}

// Close shuts down the Secret Manager client.
func Close() {
	if globalManager != nil && globalManager.client != nil {
		globalManager.client.Close()
	}
}
