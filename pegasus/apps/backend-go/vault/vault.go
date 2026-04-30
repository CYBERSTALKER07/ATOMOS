// Package vault provides AES-256-GCM encryption for supplier payment credentials.
// Plaintext secret keys NEVER leave this package or the backend process.
package vault

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"backend-go/cache"
	"backend-go/hotspot"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// masterKey is loaded once at init from VAULT_MASTER_KEY (hex-encoded 32-byte AES key).
// masterKeyPrev is the previous key for key rotation (VAULT_MASTER_KEY_PREV).
var (
	masterKey     []byte
	masterKeyPrev []byte // Previous key for rotation — Decrypt tries current, then prev
	keyVersion    byte   = 0x01
)

func init() {
	keyHex := os.Getenv("VAULT_MASTER_KEY")
	if keyHex == "" {
		log.Println("[VAULT] WARNING: VAULT_MASTER_KEY not set — credential vault will fail at runtime")
		return
	}
	decoded, err := hex.DecodeString(keyHex)
	if err != nil || len(decoded) != 32 {
		log.Printf("[VAULT] FATAL: VAULT_MASTER_KEY must be 64 hex chars (32 bytes). Got %d bytes, err=%v", len(decoded), err)
		return
	}
	masterKey = decoded

	// Optional previous key for rotation
	prevKeyHex := os.Getenv("VAULT_MASTER_KEY_PREV")
	if prevKeyHex != "" {
		prevDecoded, err := hex.DecodeString(prevKeyHex)
		if err == nil && len(prevDecoded) == 32 {
			masterKeyPrev = prevDecoded
			keyVersion = 0x02
			log.Println("[VAULT] Key rotation mode — current key v2, previous key v1 loaded")
		}
	}
	log.Printf("[VAULT] Master key loaded (version %d) — AES-256-GCM credential vault armed", keyVersion)
}

// Encrypt encrypts plaintext using AES-256-GCM with the master key.
// Returns version_byte || nonce || ciphertext.
func Encrypt(plaintext []byte) ([]byte, error) {
	if len(masterKey) == 0 {
		return nil, fmt.Errorf("vault master key not configured")
	}
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, fmt.Errorf("aes cipher init: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm init: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("nonce generation: %w", err)
	}
	sealed := gcm.Seal(nonce, nonce, plaintext, nil)
	// Prepend version byte
	result := make([]byte, 1+len(sealed))
	result[0] = keyVersion
	copy(result[1:], sealed)
	return result, nil
}

// Decrypt decrypts AES-256-GCM ciphertext with version-aware key selection.
// Supports: version-prefixed format (v1/v2) and legacy format (no version byte).
func Decrypt(ciphertext []byte) ([]byte, error) {
	if len(masterKey) == 0 {
		return nil, fmt.Errorf("vault master key not configured")
	}
	if len(ciphertext) == 0 {
		return nil, fmt.Errorf("empty ciphertext")
	}

	// Check for version byte prefix
	version := ciphertext[0]
	if version == 0x01 || version == 0x02 {
		// Versioned format: version_byte || nonce || ciphertext
		payload := ciphertext[1:]
		key := masterKey
		if version == 0x01 && masterKeyPrev != nil {
			key = masterKeyPrev
		}
		return decryptWithKey(key, payload)
	}

	// Legacy format (no version byte): nonce || ciphertext
	// Try current key first, then previous
	result, err := decryptWithKey(masterKey, ciphertext)
	if err == nil {
		return result, nil
	}
	if masterKeyPrev != nil {
		return decryptWithKey(masterKeyPrev, ciphertext)
	}
	return nil, err
}

func decryptWithKey(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher init: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm init: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ct, nil)
}

// GatewayConfig represents a supplier's payment gateway credentials.
type GatewayConfig struct {
	ConfigID    string    `json:"config_id"`
	SupplierId  string    `json:"supplier_id"`
	GatewayName string    `json:"gateway_name"`
	MerchantId  string    `json:"merchant_id"`
	ServiceId   string    `json:"service_id,omitempty"`   // Cash only
	SecretKey   string    `json:"-"`                      // NEVER serialized to JSON
	RecipientId string    `json:"recipient_id,omitempty"` // Global Pay split-payment recipient
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// GatewayConfigSummary is the safe version returned to frontends — no secret key.
type GatewayConfigSummary struct {
	ConfigID    string    `json:"config_id"`
	GatewayName string    `json:"gateway_name"`
	MerchantId  string    `json:"merchant_id"`
	ServiceId   string    `json:"service_id,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	HasSecret   bool      `json:"has_secret"`
}

// Service handles vault CRUD for SupplierPaymentConfigs.
type Service struct {
	Spanner *spanner.Client
}

// UpsertConfig creates or updates a supplier's gateway config with encrypted secret.
func (s *Service) UpsertConfig(ctx context.Context, supplierId, gatewayName, merchantId, serviceId, secretKey string) (*GatewayConfigSummary, error) {
	cap := GetProviderCapability(gatewayName)
	if cap == nil {
		return nil, fmt.Errorf("unsupported gateway: %s (supported: CASH, GLOBAL_PAY, GLOBAL_PAY)", gatewayName)
	}
	fieldValues := map[string]string{
		"merchant_id": merchantId,
		"service_id":  serviceId,
		"secret_key":  secretKey,
	}
	missing := make([]string, 0, len(cap.RequiredFields))
	for _, fieldName := range cap.RequiredFields {
		if strings.TrimSpace(fieldValues[fieldName]) == "" {
			missing = append(missing, labelForField(*cap, fieldName))
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required fields for %s: %s", gatewayName, strings.Join(missing, ", "))
	}

	encryptedKey, err := Encrypt([]byte(secretKey))
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	var configId string

	_, err = s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `SELECT ConfigId FROM SupplierPaymentConfigs
			      WHERE SupplierId = @sid AND GatewayName = @gw LIMIT 1`,
			Params: map[string]interface{}{"sid": supplierId, "gw": gatewayName},
		}
		iter := txn.Query(ctx, stmt)
		row, iterErr := iter.Next()
		iter.Stop()

		if iterErr == nil {
			// Update existing
			if colErr := row.Columns(&configId); colErr != nil {
				return colErr
			}
			cols := []string{"ConfigId", "MerchantId", "SecretKey", "IsActive", "UpdatedAt"}
			vals := []interface{}{configId, merchantId, encryptedKey, true, spanner.CommitTimestamp}
			if serviceId != "" {
				cols = append(cols, "ServiceId")
				vals = append(vals, serviceId)
			}
			return txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("SupplierPaymentConfigs", cols, vals),
			})
		}

		// Insert new
		configId = hotspot.NewOrderID()
		cols := []string{"ConfigId", "SupplierId", "GatewayName", "MerchantId", "SecretKey", "IsActive", "CreatedAt", "UpdatedAt"}
		vals := []interface{}{configId, supplierId, gatewayName, merchantId, encryptedKey, true, spanner.CommitTimestamp, spanner.CommitTimestamp}
		if serviceId != "" {
			cols = append(cols, "ServiceId")
			vals = append(vals, serviceId)
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Insert("SupplierPaymentConfigs", cols, vals),
		})
	})

	if err != nil {
		return nil, fmt.Errorf("upsert config failed: %w", err)
	}

	cache.Invalidate(ctx, cache.SupplierProfile(supplierId))

	return &GatewayConfigSummary{
		ConfigID:    configId,
		GatewayName: gatewayName,
		MerchantId:  merchantId,
		ServiceId:   serviceId,
		IsActive:    true,
		HasSecret:   true,
	}, nil
}

// SetRecipientId stores the Global Pay split-payment recipient ID on an existing
// supplier gateway config. This is a distinct lifecycle event from credential upsert.
func (s *Service) SetRecipientId(ctx context.Context, supplierId, gatewayName, recipientId string) error {
	if strings.TrimSpace(recipientId) == "" {
		return fmt.Errorf("recipient_id is required")
	}
	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `SELECT ConfigId FROM SupplierPaymentConfigs
			      WHERE SupplierId = @sid AND GatewayName = @gw AND IsActive = true LIMIT 1`,
			Params: map[string]interface{}{"sid": supplierId, "gw": gatewayName},
		}
		iter := txn.Query(ctx, stmt)
		row, iterErr := iter.Next()
		iter.Stop()
		if iterErr != nil {
			return fmt.Errorf("no active %s config for supplier %s", gatewayName, supplierId)
		}
		var configId string
		if err := row.Columns(&configId); err != nil {
			return err
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("SupplierPaymentConfigs",
				[]string{"ConfigId", "RecipientId", "UpdatedAt"},
				[]interface{}{configId, recipientId, spanner.CommitTimestamp}),
		})
	})
	if err != nil {
		return fmt.Errorf("set recipient id: %w", err)
	}

	cache.Invalidate(ctx, cache.SupplierProfile(supplierId))
	return nil
}

// ListConfigs returns all gateway configs for a supplier (secrets masked).
func (s *Service) ListConfigs(ctx context.Context, supplierId string) ([]GatewayConfigSummary, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ConfigId, GatewayName, MerchantId, ServiceId, IsActive, CreatedAt
		      FROM SupplierPaymentConfigs
		      WHERE SupplierId = @sid ORDER BY GatewayName`,
		Params: map[string]interface{}{"sid": supplierId},
	}
	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var configs []GatewayConfigSummary
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var c GatewayConfigSummary
		var serviceId spanner.NullString
		if err := row.Columns(&c.ConfigID, &c.GatewayName, &c.MerchantId, &serviceId, &c.IsActive, &c.CreatedAt); err != nil {
			return nil, err
		}
		if serviceId.Valid {
			c.ServiceId = serviceId.StringVal
		}
		c.HasSecret = true
		configs = append(configs, c)
	}
	return configs, nil
}

// ListActiveGatewayNames returns the configured active gateways for a supplier
// in provider display order.
func (s *Service) ListActiveGatewayNames(ctx context.Context, supplierId string) ([]string, error) {
	stmt := spanner.Statement{
		SQL: `SELECT GatewayName
		      FROM SupplierPaymentConfigs
		      WHERE SupplierId = @sid AND IsActive = TRUE`,
		Params: map[string]interface{}{"sid": supplierId},
	}
	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	active := make(map[string]struct{}, len(providerOrder))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var gatewayName string
		if err := row.Columns(&gatewayName); err != nil {
			return nil, err
		}
		active[gatewayName] = struct{}{}
	}

	gateways := make([]string, 0, len(active))
	for _, gatewayName := range providerOrder {
		if _, ok := active[gatewayName]; ok {
			gateways = append(gateways, gatewayName)
		}
	}
	return gateways, nil
}

// DeactivateConfig sets IsActive=false for a config.
func (s *Service) DeactivateConfig(ctx context.Context, supplierId, configId string) error {
	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "SupplierPaymentConfigs", spanner.Key{configId},
			[]string{"SupplierId", "IsActive"})
		if readErr != nil {
			return fmt.Errorf("config not found: %w", readErr)
		}
		var owner string
		var active bool
		if colErr := row.Columns(&owner, &active); colErr != nil {
			return colErr
		}
		if owner != supplierId {
			return fmt.Errorf("config does not belong to supplier")
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("SupplierPaymentConfigs",
				[]string{"ConfigId", "IsActive", "UpdatedAt"},
				[]interface{}{configId, false, spanner.CommitTimestamp}),
		})
	})
	if err != nil {
		return err
	}

	cache.Invalidate(ctx, cache.SupplierProfile(supplierId))
	return nil
}

// GetDecryptedConfig returns the full decrypted credentials for a supplier+gateway.
// INTERNAL ONLY — never expose the result to HTTP responses.
func (s *Service) GetDecryptedConfig(ctx context.Context, supplierId, gatewayName string) (*GatewayConfig, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ConfigId, MerchantId, ServiceId, SecretKey, IsActive, CreatedAt, RecipientId
		      FROM SupplierPaymentConfigs
		      WHERE SupplierId = @sid AND GatewayName = @gw AND IsActive = TRUE
		      LIMIT 1`,
		Params: map[string]interface{}{"sid": supplierId, "gw": gatewayName},
	}
	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("no active %s config for supplier %s", gatewayName, supplierId)
	}
	if err != nil {
		return nil, err
	}

	var cfg GatewayConfig
	var encryptedKey []byte
	var serviceId spanner.NullString
	var recipientId spanner.NullString
	if err := row.Columns(&cfg.ConfigID, &cfg.MerchantId, &serviceId, &encryptedKey, &cfg.IsActive, &cfg.CreatedAt, &recipientId); err != nil {
		return nil, err
	}
	if serviceId.Valid {
		cfg.ServiceId = serviceId.StringVal
	}
	if recipientId.Valid {
		cfg.RecipientId = recipientId.StringVal
	}

	decrypted, err := Decrypt(encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("vault decryption failed: %w", err)
	}

	// Audit log: track credential access for PCI compliance
	go logDecryptAccess(s.Spanner, supplierId, gatewayName, cfg.ConfigID)

	cfg.SupplierId = supplierId
	cfg.GatewayName = gatewayName
	cfg.SecretKey = string(decrypted)
	return &cfg, nil
}

// logDecryptAccess writes a decrypt audit event to the AuditLog table.
func logDecryptAccess(client *spanner.Client, supplierID, gatewayName, configID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	logID := "AUDIT-VAULT-" + time.Now().Format("20060102150405") + "-" + configID[:8]
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Insert("AuditLog",
				[]string{"LogId", "ActorId", "ActorRole", "Action", "ResourceType", "ResourceId", "Metadata", "CreatedAt"},
				[]interface{}{logID, "SYSTEM", "BACKEND", "DECRYPT", "PAYMENT_CONFIG", configID,
					`{"supplier_id":"` + supplierID + `","gateway":"` + gatewayName + `"}`,
					spanner.CommitTimestamp,
				},
			),
		})
	})
	if err != nil {
		log.Printf("[VAULT] Audit log write failed: %v", err)
	}
}

// GetDecryptedConfigByOrder resolves supplier credentials from an order ID.
// Path: OrderId -> Orders.SupplierId -> SupplierPaymentConfigs
func (s *Service) GetDecryptedConfigByOrder(ctx context.Context, orderId, gatewayName string) (*GatewayConfig, error) {
	row, err := s.Spanner.Single().ReadRow(ctx, "Orders", spanner.Key{orderId}, []string{"SupplierId"})
	if err != nil {
		return nil, fmt.Errorf("order %s not found: %w", orderId, err)
	}
	var supplierId spanner.NullString
	if err := row.Columns(&supplierId); err != nil {
		return nil, err
	}
	if !supplierId.Valid || supplierId.StringVal == "" {
		return nil, fmt.Errorf("order %s has no supplier assignment", orderId)
	}
	return s.GetDecryptedConfig(ctx, supplierId.StringVal, gatewayName)
}
