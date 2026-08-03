package retailer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"golang.org/x/crypto/bcrypt"
)

// FamilyMigrateResult is the wire shape for family → Team conversion.
type FamilyMigrateResult struct {
	RetailerID   string                 `json:"retailer_id"`
	Migrated     []FamilyMigrateItem    `json:"migrated"`
	Skipped      []FamilyMigrateSkipped `json:"skipped"`
	Remaining    int                    `json:"family_remaining"`
	FamilyWrites string                 `json:"family_writes"` // "gone" after migrate policy
}

// FamilyMigrateItem is one successful conversion.
type FamilyMigrateItem struct {
	MemberID     string `json:"member_id"`
	UserID       string `json:"user_id"`
	Phone        string `json:"phone"`
	Name         string `json:"name"`
	RetailerRole string `json:"retailer_role"`
	TempPassword string `json:"temp_password,omitempty"` // returned once for owner handoff
}

// FamilyMigrateSkipped is a family row that could not become staff.
type FamilyMigrateSkipped struct {
	MemberID string `json:"member_id"`
	Phone    string `json:"phone,omitempty"`
	Reason   string `json:"reason"`
}

// HandleFamilyMigrateToTeam serves POST /v1/retailer/family-members/migrate-to-team
// Converts RAM family contacts into TEAM staff (RECEIVER by default).
func (s *Service) HandleFamilyMigrateToTeam(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermStaffManage) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermStaffManage})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	var req struct {
		Role            string `json:"retailer_role"`
		DeactivateLogin bool   `json:"deactivate_login"` // if true, create inactive until owner activates
		KeepFamilyRows  bool   `json:"keep_family_rows"`  // default false: remove migrated from family list
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	role := strings.ToUpper(strings.TrimSpace(req.Role))
	if role == "" {
		role = "RECEIVER"
	}
	if !allowedAssignableRoles[role] {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "invalid_role"})
		return
	}

	result, err := s.migrateFamilyToTeam(r.Context(), orgID, auth.ResolveRetailerUserID(claims), role, !req.DeactivateLogin, !req.KeepFamilyRows)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "migrate_failed", "detail": err.Error()})
		return
	}
	result.FamilyWrites = "gone"
	writeJSON(w, http.StatusOK, result)
}

func (s *Service) migrateFamilyToTeam(ctx context.Context, orgID, actorID, role string, active, removeMigrated bool) (FamilyMigrateResult, error) {
	// Auto-enable TEAM.
	enabled, _ := s.LoadEnabledPacks(ctx, orgID)
	if !enabled.Has(PackTEAM) {
		_ = s.SetPackEnabled(ctx, orgID, PackTEAM, actorID, true, map[string]any{})
	}

	s.mu.Lock()
	members := append([]FamilyMember(nil), s.familyByRetailer[orgID]...)
	s.mu.Unlock()

	out := FamilyMigrateResult{RetailerID: orgID, Migrated: []FamilyMigrateItem{}, Skipped: []FamilyMigrateSkipped{}}
	var removeIDs []string

	for _, m := range members {
		phone := strings.TrimSpace(m.Phone)
		name := strings.TrimSpace(m.Name)
		if phone == "" {
			out.Skipped = append(out.Skipped, FamilyMigrateSkipped{MemberID: m.MemberID, Reason: "phone_required"})
			continue
		}
		if len(name) < 2 {
			name = "Team member"
		}
		tempPass, err := randomTempPassword(10)
		if err != nil {
			return out, err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(tempPass), bcrypt.DefaultCost)
		if err != nil {
			return out, err
		}
		user := RetailerUser{
			UserID:       s.newID(),
			RetailerID:   orgID,
			Phone:        phone,
			Name:         name,
			PasswordHash: string(hash),
			RetailerRole: role,
			IsOwner:      false,
			IsActive:     active,
			CreatedAt:    s.now(),
			UpdatedAt:    s.now(),
		}
		if err := s.createOrgMember(ctx, user); err != nil {
			if errors.Is(err, errRetailerMemberPhoneExists) {
				out.Skipped = append(out.Skipped, FamilyMigrateSkipped{
					MemberID: m.MemberID, Phone: phone, Reason: "phone_already_staff",
				})
				// Still drop family row if keep not requested — already on Team.
				if removeMigrated {
					removeIDs = append(removeIDs, m.MemberID)
				}
				continue
			}
			out.Skipped = append(out.Skipped, FamilyMigrateSkipped{
				MemberID: m.MemberID, Phone: phone, Reason: "persist_failed",
			})
			continue
		}
		item := FamilyMigrateItem{
			MemberID: m.MemberID, UserID: user.UserID, Phone: phone, Name: name,
			RetailerRole: role, TempPassword: tempPass,
		}
		out.Migrated = append(out.Migrated, item)
		if removeMigrated {
			removeIDs = append(removeIDs, m.MemberID)
		}
	}

	if len(removeIDs) > 0 {
		rm := map[string]bool{}
		for _, id := range removeIDs {
			rm[id] = true
		}
		s.mu.Lock()
		cur := s.familyByRetailer[orgID]
		kept := cur[:0]
		for _, m := range cur {
			if !rm[m.MemberID] {
				kept = append(kept, m)
			}
		}
		s.familyByRetailer[orgID] = kept
		s.mu.Unlock()
	}

	s.mu.RLock()
	out.Remaining = len(s.familyByRetailer[orgID])
	s.mu.RUnlock()

	// Mark policy: family POST goes to 410 after first successful migrate attempt (even if empty).
	s.markFamilyWritesGone(ctx, orgID)

	_ = s.emitPosEvent(ctx, orgID, events.EventRetailerStaffCreated, map[string]any{
		"source":         "family_migrate",
		"migrated_count": len(out.Migrated),
		"skipped_count":  len(out.Skipped),
	})
	return out, nil
}

const familyWritesGoneFlag = "family_writes_gone"

// markFamilyWritesGone persists migrate policy (POST family → 410) in memory + Spanner.
func (s *Service) markFamilyWritesGone(ctx context.Context, orgID string) {
	s.mu.Lock()
	if s.familyWritesGone == nil {
		s.familyWritesGone = map[string]bool{}
	}
	s.familyWritesGone[orgID] = true
	s.mu.Unlock()
	s.persistOrgFlag(ctx, orgID, familyWritesGoneFlag, "true")
}

func (s *Service) isFamilyWritesGone(ctx context.Context, orgID string) bool {
	s.mu.RLock()
	if s.familyWritesGone[orgID] {
		s.mu.RUnlock()
		return true
	}
	s.mu.RUnlock()
	if s.loadOrgFlag(ctx, orgID, familyWritesGoneFlag) == "true" {
		s.mu.Lock()
		if s.familyWritesGone == nil {
			s.familyWritesGone = map[string]bool{}
		}
		s.familyWritesGone[orgID] = true
		s.mu.Unlock()
		return true
	}
	return false
}

func (s *Service) persistOrgFlag(ctx context.Context, orgID, key, value string) {
	if s.spannerClient == nil || strings.TrimSpace(orgID) == "" || strings.TrimSpace(key) == "" {
		return
	}
	_, _ = s.spannerClient.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("RetailerOrgFlags", map[string]any{
			"RetailerId": orgID,
			"FlagKey":    key,
			"FlagValue":  value,
			"UpdatedAt":  spanner.CommitTimestamp,
		}),
	})
}

func (s *Service) loadOrgFlag(ctx context.Context, orgID, key string) string {
	if s.spannerClient == nil {
		return ""
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerOrgFlags",
		spanner.Key{orgID, key}, []string{"FlagValue"})
	if err != nil {
		return ""
	}
	var v string
	if err := row.Columns(&v); err != nil {
		return ""
	}
	return strings.TrimSpace(v)
}

func randomTempPassword(n int) (string, error) {
	if n < 8 {
		n = 8
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// readable hex + prefix
	return "Tmp!" + hex.EncodeToString(b)[:n], nil
}

// patchFamilyHandlers: POST blocked with 410 when migrate policy applied.
func (s *Service) familyPostBlocked(w http.ResponseWriter, r *http.Request, orgID string) bool {
	if s.isFamilyWritesGone(r.Context(), orgID) {
		writeJSON(w, http.StatusGone, map[string]any{
			"error":   "family_writes_gone",
			"message": "Use Team (POST /v1/retailer/org/members). Run migrate-to-team for legacy list.",
			"migrate": "/v1/retailer/family-members/migrate-to-team",
		})
		return true
	}
	return false
}

