package retailer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// RetailerUser is a credentialed person under a retailer org (tenant).
type RetailerUser struct {
	UserID       string
	RetailerID   string
	Phone        string
	Name         string
	PasswordHash string
	FirebaseUID  string
	RetailerRole string
	IsOwner      bool
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// EnsureOwnerUser guarantees a durable OWNER row for the retailer shop.
// Idempotent: returns existing owner/phone match when present.
func (s *Service) EnsureOwnerUser(ctx context.Context, ret Retailer) (RetailerUser, error) {
	if s == nil {
		return RetailerUser{}, fmt.Errorf("nil service")
	}
	phone := strings.TrimSpace(ret.Phone)
	if phone == "" {
		// Allow bootstrap without phone using retailer id as pseudo-phone for tests.
		phone = strings.TrimSpace(ret.RetailerID)
	}
	if phone == "" {
		return RetailerUser{}, fmt.Errorf("retailer phone required for owner bootstrap")
	}
	if s.now == nil {
		s.now = func() time.Time { return time.Now().UTC() }
	}
	if s.newID == nil {
		s.newID = defaultRetailerID
	}

	// Prefer Spanner when available.
	if s.spannerClient != nil {
		if u, ok, err := s.findRetailerUserByRetailerPhone(ctx, ret.RetailerID, phone); err != nil {
			return RetailerUser{}, err
		} else if ok {
			return u, nil
		}
		// Also: any owner for this retailer.
		if u, ok, err := s.findOwnerUser(ctx, ret.RetailerID); err != nil {
			return RetailerUser{}, err
		} else if ok {
			return u, nil
		}
		return s.insertOwnerUser(ctx, ret)
	}

	// Memory fallback for unit tests without Spanner.
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ownerByRetailer == nil {
		s.ownerByRetailer = map[string]RetailerUser{}
	}
	if u, ok := s.ownerByRetailer[ret.RetailerID]; ok {
		if s.membershipsByUser == nil {
			s.membershipsByUser = map[string]map[string]RetailerMembership{}
		}
		if s.membershipsByUser[u.UserID] == nil {
			s.membershipsByUser[u.UserID] = map[string]RetailerMembership{}
		}
		s.membershipsByUser[u.UserID][u.RetailerID] = RetailerMembership{
			UserID: u.UserID, RetailerID: u.RetailerID, RetailerRole: u.RetailerRole,
			IsActive: u.IsActive, Phone: u.Phone, Name: u.Name,
		}
		return u, nil
	}
	u := RetailerUser{
		UserID:       s.newID(),
		RetailerID:   ret.RetailerID,
		Phone:        phone,
		Name:         coalesceRetailerName(ret.Name),
		RetailerRole: "OWNER",
		IsOwner:      true,
		IsActive:     true,
		CreatedAt:    s.now(),
		UpdatedAt:    s.now(),
	}
	s.ownerByRetailer[ret.RetailerID] = u
	// Dual-write membership (memory) — unlock held after return via unlocked path
	if s.membershipsByUser == nil {
		s.membershipsByUser = map[string]map[string]RetailerMembership{}
	}
	if s.membershipsByUser[u.UserID] == nil {
		s.membershipsByUser[u.UserID] = map[string]RetailerMembership{}
	}
	s.membershipsByUser[u.UserID][u.RetailerID] = RetailerMembership{
		UserID: u.UserID, RetailerID: u.RetailerID, RetailerRole: u.RetailerRole,
		IsActive: true, Phone: u.Phone, Name: u.Name,
	}
	return u, nil
}

func (s *Service) insertOwnerUser(ctx context.Context, ret Retailer) (RetailerUser, error) {
	u := RetailerUser{
		UserID:       s.newID(),
		RetailerID:   ret.RetailerID,
		Phone:        strings.TrimSpace(ret.Phone),
		Name:         coalesceRetailerName(ret.Name),
		RetailerRole: "OWNER",
		IsOwner:      true,
		IsActive:     true,
		CreatedAt:    s.now(),
		UpdatedAt:    s.now(),
	}
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		m := spanner.InsertMap("RetailerUsers", map[string]any{
			"UserId":       u.UserID,
			"RetailerId":   u.RetailerID,
			"Phone":        u.Phone,
			"Name":         u.Name,
			"RetailerRole": u.RetailerRole,
			"IsOwner":      true,
			"IsActive":     true,
			"CreatedAt":    spanner.CommitTimestamp,
			"UpdatedAt":    spanner.CommitTimestamp,
		})
		if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
			return err
		}
		// C1.1 dual-write membership (best-effort if table missing — Apply after txn)
		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateRetailer, u.RetailerID, events.TopicMain, events.RetailerEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventRetailerStaffCreated, Timestamp: s.now().Format(time.RFC3339Nano)},
			RetailerID: u.RetailerID,
			Phone:      u.Phone,
			Name:       u.Name,
			SupplierID: s.resolveSupplierScope(ctx),
			UserID:     u.UserID,
		}); err != nil {
			return err
		}
		return buf.Flush(txn)
	})
	if err != nil {
		// Race: another request created owner — re-read.
		if u2, ok, err2 := s.findRetailerUserByRetailerPhone(ctx, ret.RetailerID, ret.Phone); err2 == nil && ok {
			_ = s.UpsertMembershipFromUser(ctx, u2)
			return u2, nil
		}
		if u2, ok, err2 := s.findOwnerUser(ctx, ret.RetailerID); err2 == nil && ok {
			_ = s.UpsertMembershipFromUser(ctx, u2)
			return u2, nil
		}
		return RetailerUser{}, fmt.Errorf("insert owner user: %w", err)
	}
	_ = s.UpsertMembershipFromUser(ctx, u)
	return u, nil
}

func (s *Service) findRetailerUserByRetailerPhone(ctx context.Context, retailerID, phone string) (RetailerUser, bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT UserId, RetailerId, Phone, Name, IFNULL(PasswordHash, ''), IFNULL(FirebaseUid, ''),
			RetailerRole, IsOwner, IsActive, CreatedAt, UpdatedAt
			FROM RetailerUsers@{FORCE_INDEX=UQ_RetailerUsers_ByRetailerPhone}
			WHERE RetailerId = @rid AND Phone = @phone LIMIT 1`,
		Params: map[string]any{"rid": retailerID, "phone": phone},
	}
	return s.scanOneRetailerUser(ctx, stmt)
}

func (s *Service) findOwnerUser(ctx context.Context, retailerID string) (RetailerUser, bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT UserId, RetailerId, Phone, Name, IFNULL(PasswordHash, ''), IFNULL(FirebaseUid, ''),
			RetailerRole, IsOwner, IsActive, CreatedAt, UpdatedAt
			FROM RetailerUsers@{FORCE_INDEX=Idx_RetailerUsers_ByRetailer}
			WHERE RetailerId = @rid AND IsOwner = TRUE AND IsActive = TRUE LIMIT 1`,
		Params: map[string]any{"rid": retailerID},
	}
	return s.scanOneRetailerUser(ctx, stmt)
}

func (s *Service) findRetailerUserByID(ctx context.Context, userID string) (RetailerUser, bool, error) {
	if s.spannerClient == nil {
		return RetailerUser{}, false, nil
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerUsers", spanner.Key{userID},
		[]string{"UserId", "RetailerId", "Phone", "Name", "PasswordHash", "FirebaseUid", "RetailerRole", "IsOwner", "IsActive", "CreatedAt", "UpdatedAt"})
	if err != nil {
		if isNotFound(err) {
			return RetailerUser{}, false, nil
		}
		return RetailerUser{}, false, err
	}
	u, err := decodeRetailerUserRow(row)
	if err != nil {
		return RetailerUser{}, false, err
	}
	return u, true, nil
}

func (s *Service) scanOneRetailerUser(ctx context.Context, stmt spanner.Statement) (RetailerUser, bool, error) {
	if s.spannerClient == nil {
		return RetailerUser{}, false, nil
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return RetailerUser{}, false, nil
	}
	if err != nil {
		return RetailerUser{}, false, err
	}
	u, err := decodeRetailerUserRow(row)
	if err != nil {
		return RetailerUser{}, false, err
	}
	return u, true, nil
}

func decodeRetailerUserRow(row *spanner.Row) (RetailerUser, error) {
	var u RetailerUser
	var pass, fb spanner.NullString
	var created, updated time.Time
	// Query uses IFNULL so PasswordHash/FirebaseUid may be string; ReadRow may be NullString.
	// Try flexible decode via columns by index using interface.
	cols := row.Size()
	if cols >= 11 {
		var password, firebase string
		if err := row.Columns(
			&u.UserID, &u.RetailerID, &u.Phone, &u.Name,
			&password, &firebase,
			&u.RetailerRole, &u.IsOwner, &u.IsActive, &created, &updated,
		); err != nil {
			// Retry with NullString for password/firebase from ReadRow
			if err2 := row.Columns(
				&u.UserID, &u.RetailerID, &u.Phone, &u.Name,
				&pass, &fb,
				&u.RetailerRole, &u.IsOwner, &u.IsActive, &created, &updated,
			); err2 != nil {
				return RetailerUser{}, err
			}
			if pass.Valid {
				u.PasswordHash = pass.StringVal
			}
			if fb.Valid {
				u.FirebaseUID = fb.StringVal
			}
		} else {
			u.PasswordHash = password
			u.FirebaseUID = firebase
		}
	}
	u.CreatedAt = created
	u.UpdatedAt = updated
	return u, nil
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	// spanner.ErrCode not always imported; string match is acceptable here.
	msg := err.Error()
	return strings.Contains(msg, "code = NotFound") || strings.Contains(msg, "not found")
}
