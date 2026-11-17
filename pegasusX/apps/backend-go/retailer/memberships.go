package retailer

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// RetailerMembership is one active (or historical) org link for a person (UserId).
// Wave C1.1: dual-written with RetailerUsers; C1.2 login uses multi-membership list.
type RetailerMembership struct {
	UserID       string   `json:"user_id"`
	RetailerID   string   `json:"retailer_id"`
	RetailerRole string   `json:"retailer_role"`
	IsActive     bool     `json:"is_active"`
	LocationIDs  []string `json:"location_ids,omitempty"`
	// Display fields (from RetailerUsers join / dual-read)
	Phone     string `json:"phone,omitempty"`
	Name      string `json:"name,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// UpsertMembership writes a membership row (memory or Spanner). Idempotent.
func (s *Service) UpsertMembership(ctx context.Context, m RetailerMembership) error {
	if s == nil {
		return nil
	}
	m.UserID = strings.TrimSpace(m.UserID)
	m.RetailerID = strings.TrimSpace(m.RetailerID)
	m.RetailerRole = strings.TrimSpace(m.RetailerRole)
	if m.UserID == "" || m.RetailerID == "" {
		return nil
	}
	if m.RetailerRole == "" {
		m.RetailerRole = "CASHIER"
	}
	locJSON := ""
	if len(m.LocationIDs) > 0 {
		raw, _ := json.Marshal(m.LocationIDs)
		locJSON = string(raw)
	}

	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.membershipsByUser == nil {
			s.membershipsByUser = map[string]map[string]RetailerMembership{}
		}
		if s.membershipsByUser[m.UserID] == nil {
			s.membershipsByUser[m.UserID] = map[string]RetailerMembership{}
		}
		now := s.now().UTC().Format(time.RFC3339Nano)
		if cur, ok := s.membershipsByUser[m.UserID][m.RetailerID]; ok && cur.CreatedAt != "" {
			m.CreatedAt = cur.CreatedAt
		} else if m.CreatedAt == "" {
			m.CreatedAt = now
		}
		m.UpdatedAt = now
		s.membershipsByUser[m.UserID][m.RetailerID] = m
		return nil
	}

	_, err := s.spannerClient.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("RetailerUserMemberships", map[string]any{
			"UserId":          m.UserID,
			"RetailerId":      m.RetailerID,
			"RetailerRole":    m.RetailerRole,
			"IsActive":        m.IsActive,
			"LocationIdsJson": nullableStr(locJSON),
			"CreatedAt":       spanner.CommitTimestamp,
			"UpdatedAt":       spanner.CommitTimestamp,
		}),
	})
	// Pre-migration: table missing — ignore so user create still works.
	if err != nil && (strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "Table not found") ||
		strings.Contains(err.Error(), "RetailerUserMemberships")) {
		return nil
	}
	return err
}

// UpsertMembershipFromUser dual-writes membership from a RetailerUsers row.
func (s *Service) UpsertMembershipFromUser(ctx context.Context, u RetailerUser) error {
	return s.UpsertMembership(ctx, RetailerMembership{
		UserID:       u.UserID,
		RetailerID:   u.RetailerID,
		RetailerRole: u.RetailerRole,
		IsActive:     u.IsActive,
		Phone:        u.Phone,
		Name:         u.Name,
	})
}

// ListMembershipsByUser returns active memberships for a global person id.
// Dual-read: if any membership row exists (active or not), return only active ones
// (empty when all deactivated). If no membership rows exist, synthesize from
// RetailerUsers / memory owner+staff (pre-backfill).
func (s *Service) ListMembershipsByUser(ctx context.Context, userID string) ([]RetailerMembership, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, nil
	}
	all, err := s.listMembershipsByUserSpannerOrMem(ctx, userID, false)
	if err != nil {
		return nil, err
	}
	if len(all) > 0 {
		var active []RetailerMembership
		for _, m := range all {
			if m.IsActive {
				active = append(active, m)
			}
		}
		return active, nil
	}
	// Dual-read legacy: single RetailerUsers row for this UserId
	if u, ok, err := s.findRetailerUserByID(ctx, userID); err != nil {
		return nil, err
	} else if ok {
		if !u.IsActive {
			return nil, nil
		}
		m := RetailerMembership{
			UserID: u.UserID, RetailerID: u.RetailerID, RetailerRole: u.RetailerRole,
			IsActive: u.IsActive, Phone: u.Phone, Name: u.Name,
			CreatedAt: formatTimeOpt(u.CreatedAt), UpdatedAt: formatTimeOpt(u.UpdatedAt),
		}
		return []RetailerMembership{m}, nil
	}
	// Memory owner/staff scan
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.ownerByRetailer {
		if u.UserID == userID && u.IsActive {
			return []RetailerMembership{{
				UserID: u.UserID, RetailerID: u.RetailerID, RetailerRole: u.RetailerRole,
				IsActive: u.IsActive, Phone: u.Phone, Name: u.Name,
			}}, nil
		}
	}
	for _, list := range s.staffByRetailer {
		for _, u := range list {
			if u.UserID == userID && u.IsActive {
				return []RetailerMembership{{
					UserID: u.UserID, RetailerID: u.RetailerID, RetailerRole: u.RetailerRole,
					IsActive: u.IsActive, Phone: u.Phone, Name: u.Name,
				}}, nil
			}
		}
	}
	return nil, nil
}

// ListMembershipsByPhone returns all org memberships for a phone (multi-org ready).
// Dual-read:
//  1. Memberships joined via users with that phone
//  2. Fallback: all RetailerUsers rows with that phone (not LIMIT 1)
func (s *Service) ListMembershipsByPhone(ctx context.Context, phone string) ([]RetailerMembership, error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return nil, nil
	}

	// Path A: users with phone → memberships for each user id (union)
	users, err := s.listRetailerUsersByPhone(ctx, phone)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{} // retailerID key (active membership preferred)
	var out []RetailerMembership
	for _, u := range users {
		ms, err := s.listMembershipsByUserSpannerOrMem(ctx, u.UserID, true)
		if err != nil {
			return nil, err
		}
		if len(ms) == 0 {
			// Dual-read: synthesize membership from user row
			ms = []RetailerMembership{{
				UserID: u.UserID, RetailerID: u.RetailerID, RetailerRole: u.RetailerRole,
				IsActive: u.IsActive, Phone: u.Phone, Name: u.Name,
				CreatedAt: formatTimeOpt(u.CreatedAt), UpdatedAt: formatTimeOpt(u.UpdatedAt),
			}}
		}
		for _, m := range ms {
			if !m.IsActive {
				continue
			}
			// Enrich display
			if m.Phone == "" {
				m.Phone = u.Phone
			}
			if m.Name == "" {
				m.Name = u.Name
			}
			key := m.RetailerID + "|" + m.UserID
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, m)
		}
	}
	return out, nil
}

// listRetailerUsersByPhone returns ALL active users with phone (not LIMIT 1).
func (s *Service) listRetailerUsersByPhone(ctx context.Context, phone string) ([]RetailerUser, error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return nil, nil
	}
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		var out []RetailerUser
		for _, u := range s.ownerByRetailer {
			if u.Phone == phone && u.IsActive {
				out = append(out, u)
			}
		}
		for _, list := range s.staffByRetailer {
			for _, u := range list {
				if u.Phone == phone && u.IsActive {
					out = append(out, u)
				}
			}
		}
		return out, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT UserId, RetailerId, Phone, Name, IFNULL(PasswordHash, ''), IFNULL(FirebaseUid, ''),
			RetailerRole, IsOwner, IsActive, CreatedAt, UpdatedAt
			FROM RetailerUsers@{FORCE_INDEX=Idx_RetailerUsers_ByPhone}
			WHERE Phone = @phone AND IsActive = TRUE`,
		Params: map[string]any{"phone": phone},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []RetailerUser
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		u, err := decodeRetailerUserRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, nil
}

func (s *Service) listMembershipsByUserSpannerOrMem(ctx context.Context, userID string, activeOnly bool) ([]RetailerMembership, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		if s.membershipsByUser == nil {
			return nil, nil
		}
		var out []RetailerMembership
		for _, m := range s.membershipsByUser[userID] {
			if activeOnly && !m.IsActive {
				continue
			}
			out = append(out, m)
		}
		return out, nil
	}
	sql := `SELECT UserId, RetailerId, RetailerRole, IsActive, COALESCE(LocationIdsJson, ''),
		CAST(CreatedAt AS STRING), CAST(UpdatedAt AS STRING)
		FROM RetailerUserMemberships WHERE UserId = @uid`
	if activeOnly {
		sql += ` AND IsActive = true`
	}
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL:    sql,
		Params: map[string]any{"uid": userID},
	})
	defer iter.Stop()
	var out []RetailerMembership
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// Table missing pre-migration
			if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "RetailerUserMemberships") {
				return nil, nil
			}
			return nil, err
		}
		var m RetailerMembership
		var locJSON, created, updated string
		if err := row.Columns(&m.UserID, &m.RetailerID, &m.RetailerRole, &m.IsActive, &locJSON, &created, &updated); err != nil {
			return nil, err
		}
		m.CreatedAt = created
		m.UpdatedAt = updated
		if locJSON != "" {
			_ = json.Unmarshal([]byte(locJSON), &m.LocationIDs)
		}
		out = append(out, m)
	}
	return out, nil
}

// BackfillMembershipsFromUsers copies every RetailerUsers row into memberships.
// Safe to re-run (InsertOrUpdate). Returns number of rows attempted.
func (s *Service) BackfillMembershipsFromUsers(ctx context.Context) (int, error) {
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		n := 0
		if s.membershipsByUser == nil {
			s.membershipsByUser = map[string]map[string]RetailerMembership{}
		}
		write := func(u RetailerUser) {
			if s.membershipsByUser[u.UserID] == nil {
				s.membershipsByUser[u.UserID] = map[string]RetailerMembership{}
			}
			s.membershipsByUser[u.UserID][u.RetailerID] = RetailerMembership{
				UserID: u.UserID, RetailerID: u.RetailerID, RetailerRole: u.RetailerRole,
				IsActive: u.IsActive, Phone: u.Phone, Name: u.Name,
			}
			n++
		}
		for _, u := range s.ownerByRetailer {
			write(u)
		}
		for _, list := range s.staffByRetailer {
			for _, u := range list {
				write(u)
			}
		}
		return n, nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT UserId, RetailerId, Phone, Name, IFNULL(PasswordHash, ''), IFNULL(FirebaseUid, ''),
			RetailerRole, IsOwner, IsActive, CreatedAt, UpdatedAt
			FROM RetailerUsers
			WHERE RetailerId IS NOT NULL`,
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	n := 0
	var muts []*spanner.Mutation
	flush := func() error {
		if len(muts) == 0 {
			return nil
		}
		_, err := s.spannerClient.Apply(ctx, muts)
		muts = muts[:0]
		return err
	}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return n, err
		}
		u, err := decodeRetailerUserRow(row)
		if err != nil {
			return n, err
		}
		muts = append(muts, spanner.InsertOrUpdateMap("RetailerUserMemberships", map[string]any{
			"UserId":       u.UserID,
			"RetailerId":   u.RetailerID,
			"RetailerRole": u.RetailerRole,
			"IsActive":     u.IsActive,
			"CreatedAt":    spanner.CommitTimestamp,
			"UpdatedAt":    spanner.CommitTimestamp,
		}))
		n++
		if len(muts) >= 100 {
			if err := flush(); err != nil {
				return n, err
			}
		}
	}
	if err := flush(); err != nil {
		return n, err
	}
	return n, nil
}

// MembershipCountForUser is a test/ops helper.
func (s *Service) MembershipCountForUser(ctx context.Context, userID string) int {
	ms, _ := s.ListMembershipsByUser(ctx, userID)
	return len(ms)
}
