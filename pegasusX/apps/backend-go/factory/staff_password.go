package factory

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"golang.org/x/crypto/bcrypt"
)

const staffPasswordUnsetSentinel = "unset"

var (
	errEmptyStaffSecret    = errors.New("staff_secret_empty")
	errStaffSecretTooShort = errors.New("staff_secret_too_short")
	errStaffNotFound       = errors.New("staff_not_found")
)

func hashFactoryStaffSecret(secret string) (string, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "", errEmptyStaffSecret
	}
	h, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

func randomFactoryInviteToken() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

func resolveStaffSecret(pin, password string) (secret string, generatedInvite string, err error) {
	secret = strings.TrimSpace(pin)
	if secret == "" {
		secret = strings.TrimSpace(password)
	}
	if secret != "" {
		if len(secret) < 4 {
			return "", "", errStaffSecretTooShort
		}
		return secret, "", nil
	}
	invite, err := randomFactoryInviteToken()
	if err != nil {
		return "", "", err
	}
	return invite, invite, nil
}

// HandleStaffSetPassword serves POST /v1/factory/staff/{staffID}/set-password.
func (s *Service) HandleStaffSetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	staffID := strings.TrimSpace(chi.URLParam(r, "staffID"))
	if staffID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "staff_id_required"})
		return
	}
	body, err := readLimitedBody(r, 8*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	var req struct {
		PIN      string `json:"pin"`
		Password string `json:"password"`
	}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
	}
	secret := strings.TrimSpace(req.PIN)
	if secret == "" {
		secret = strings.TrimSpace(req.Password)
	}
	if secret == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "pin_or_password_required"})
		return
	}
	if len(secret) < 4 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "pin_or_password_too_short"})
		return
	}
	hash, err := hashFactoryStaffSecret(secret)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "staff_password_hash_failed"})
		return
	}

	supplierID := s.resolveSupplierScope(r.Context())
	factoryID := s.resolveFactoryNode(r.Context())
	if err := s.persistStaffPasswordHash(r.Context(), staffID, hash, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateFactory, staffID, events.TopicMain, events.FactoryEvent{
			BaseEvent:    events.BaseEvent{Type: events.EventFactoryStaffPasswordSet},
			FactoryID:    factoryID,
			SupplierID:   supplierID,
			UserID:       staffID,
			SupplierRole: string(auth.RoleFactory),
		})
	}); err != nil {
		if errors.Is(err, errStaffNotFound) || strings.Contains(strings.ToLower(err.Error()), "not found") {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "staff_not_found"})
			return
		}
		s.log.ErrorContext(r.Context(), "factory staff set-password failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "staff_password_update_failed"})
		return
	}
	s.invalidateFactoryKeys(r.Context(), factoryStaffListKey(supplierID))
	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
		"staff_id":           staffID,
		"must_set_password":  false,
		"password_set":       true,
	})
}

func (s *Service) persistStaffPasswordHash(ctx context.Context, staffID, hash string, emit func(outbox.TxnBuffer) error) error {
	staffID = strings.TrimSpace(staffID)
	if staffID == "" {
		return errStaffNotFound
	}
	if s.spannerClient != nil {
		_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			if _, err := txn.ReadRow(ctx, "SupplierUsers", spanner.Key{staffID}, []string{"UserId"}); err != nil {
				return err
			}
			buf := &spannerTxnBuffer{}
			if emit != nil {
				if err := emit(buf); err != nil {
					return err
				}
			}
			muts := []*spanner.Mutation{spanner.UpdateMap("SupplierUsers", map[string]any{
				"UserId":       staffID,
				"PasswordHash": hash,
				"UpdatedAt":    spanner.CommitTimestamp,
			})}
			muts = append(muts, outboxMutations(buf.events)...)
			return txn.BufferWrite(muts)
		})
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	found := false
	for i := range s.staff {
		if s.staff[i].StaffID == staffID {
			s.staff[i].PasswordHash = hash
			found = true
			break
		}
	}
	if !found {
		return errStaffNotFound
	}
	if emit != nil {
		return s.repo.RunTx(ctx, func(context.Context, FactoryTx) error { return nil }, emit)
	}
	return nil
}
