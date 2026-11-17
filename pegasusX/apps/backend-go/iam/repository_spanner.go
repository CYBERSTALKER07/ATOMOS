package iam

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// SpannerRepository implements Repository against Cloud Spanner using StrongRead()
// and atomic ReadWriteTransaction mutations.
type SpannerRepository struct {
	client      *spanner.Client
	seededMu    sync.RWMutex
	seededCache map[string]bool
}

// NewSpannerRepository creates a Spanner-backed IAM repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{
		client:      client,
		seededCache: make(map[string]bool),
	}
}

// Client returns the underlying Spanner client.
func (r *SpannerRepository) Client() *spanner.Client {
	if r == nil {
		return nil
	}
	return r.client
}

// tenantOutboxBuffer wraps SpannerTxnBuffer to stamp SupplierId with the tenantID
// for IAM/SoD events whose JSON payload carries tenant_id instead of supplier_id.
type tenantOutboxBuffer struct {
	inner    *outbox.SpannerTxnBuffer
	tenantID string
}

func (b *tenantOutboxBuffer) BufferOutbox(ctx context.Context, e outbox.Event) error {
	if strings.TrimSpace(b.tenantID) != "" && (e.SupplierID == "" || e.SupplierID == outbox.PlatformSupplierID) {
		e.SupplierID = b.tenantID
	}
	return b.inner.BufferOutbox(ctx, e)
}

func (r *SpannerRepository) EnsureTenantSeeded(ctx context.Context, tenantID string) error {
	tid := strings.TrimSpace(tenantID)
	if tid == "" {
		return fmt.Errorf("%w: tenant_id required", ErrInvalidIAMInput)
	}
	if r == nil || r.client == nil {
		return ErrIAMUnavailable
	}

	r.seededMu.RLock()
	already := r.seededCache[tid]
	r.seededMu.RUnlock()
	if already {
		return nil
	}

	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		return ensureTenantSeededInTxn(txCtx, txn, tid)
	})
	if err != nil {
		return err
	}

	r.seededMu.Lock()
	r.seededCache[tid] = true
	r.seededMu.Unlock()
	return nil
}

func ensureTenantSeededInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, tid string) error {
	existingRP := make(map[string]bool)
	rpIter := txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT RoleName, PermissionKey FROM RolePermissions WHERE TenantId = @tid`,
		Params: map[string]any{"tid": tid},
	})
	for {
		row, err := rpIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			rpIter.Stop()
			return fmt.Errorf("read RolePermissions: %w", err)
		}
		var rn, pk string
		if err := row.Columns(&rn, &pk); err != nil {
			rpIter.Stop()
			return err
		}
		existingRP[rn+"|"+pk] = true
	}
	rpIter.Stop()

	var muts []*spanner.Mutation
	if len(existingRP) == 0 {
		for role, perms := range auth.SupplierRoleTemplates() {
			for _, perm := range perms {
				muts = append(muts, spanner.InsertMap("RolePermissions", map[string]any{
					"TenantId":      tid,
					"RoleName":      role,
					"PermissionKey": perm,
					"GrantedBy":     "system:seed",
					"CreatedAt":     spanner.CommitTimestamp,
				}))
			}
		}
	}

	existingPols := make(map[string]bool)
	polIter := txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT PolicyId FROM SodPolicies WHERE TenantId = @tid`,
		Params: map[string]any{"tid": tid},
	})
	for {
		row, err := polIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			polIter.Stop()
			return fmt.Errorf("read SodPolicies: %w", err)
		}
		var pid string
		if err := row.Columns(&pid); err != nil {
			polIter.Stop()
			return err
		}
		existingPols[pid] = true
	}
	polIter.Stop()

	for _, p := range DefaultSodPolicies(tid, time.Now().UTC()) {
		if existingPols[p.PolicyID] {
			continue
		}
		muts = append(muts, spanner.InsertMap("SodPolicies", map[string]any{
			"TenantId":          tid,
			"PolicyId":          p.PolicyID,
			"Name":              p.Name,
			"PermissionA":       p.PermissionA,
			"PermissionB":       p.PermissionB,
			"EnforceSameObject": p.EnforceSameObject,
			"Enabled":           p.Enabled,
			"CreatedAt":         spanner.CommitTimestamp,
		}))
	}

	if len(muts) > 0 {
		return txn.BufferWrite(muts)
	}
	return nil
}

func (r *SpannerRepository) ListRolePermissions(ctx context.Context, tenantID string) (map[string][]string, error) {
	if r == nil || r.client == nil {
		return nil, ErrIAMUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	if tid == "" {
		return auth.SupplierRoleTemplates(), nil
	}
	if err := r.EnsureTenantSeeded(ctx, tid); err != nil {
		return nil, err
	}

	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL:    `SELECT RoleName, PermissionKey FROM RolePermissions WHERE TenantId = @tid ORDER BY RoleName, PermissionKey`,
		Params: map[string]any{"tid": tid},
	})
	defer iter.Stop()

	out := make(map[string][]string)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list RolePermissions: %w", err)
		}
		var rn, pk string
		if err := row.Columns(&rn, &pk); err != nil {
			return nil, err
		}
		out[rn] = append(out[rn], pk)
	}
	return out, nil
}

func (r *SpannerRepository) ListUserRoles(ctx context.Context, tenantID, userID string) ([]UserRoleAssignment, error) {
	if r == nil || r.client == nil {
		return nil, ErrIAMUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	uid := strings.TrimSpace(userID)

	sql := `SELECT TenantId, UserId, RoleName, COALESCE(AssignedBy, ''), ExpiresAt, CreatedAt, UpdatedAt FROM SupplierUserRoles`
	params := map[string]any{}
	var conds []string
	if tid != "" {
		conds = append(conds, `TenantId = @tid`)
		params["tid"] = tid
	}
	if uid != "" {
		conds = append(conds, `UserId = @uid`)
		params["uid"] = uid
	}
	if len(conds) > 0 {
		sql += ` WHERE ` + strings.Join(conds, ` AND `)
	}
	sql += ` ORDER BY UserId, RoleName`

	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL:    sql,
		Params: params,
	})
	defer iter.Stop()

	var out []UserRoleAssignment
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list SupplierUserRoles: %w", err)
		}
		var a UserRoleAssignment
		var expAt spanner.NullTime
		if err := row.Columns(&a.TenantID, &a.UserID, &a.RoleName, &a.AssignedBy, &expAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		if expAt.Valid {
			t := expAt.Time.UTC()
			a.ExpiresAt = &t
		}
		a.CreatedAt = a.CreatedAt.UTC()
		a.UpdatedAt = a.UpdatedAt.UTC()
		out = append(out, a)
	}
	if out == nil {
		out = []UserRoleAssignment{}
	}
	return out, nil
}

func (r *SpannerRepository) ApplyUserRoleChange(
	ctx context.Context,
	tenantID, userID, roleName, action, assignedBy, _ string, traceID string,
	auditFn func(context.Context, *spanner.ReadWriteTransaction) error,
) ([]UserRoleAssignment, error) {
	if r == nil || r.client == nil {
		return nil, ErrIAMUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	uid := strings.TrimSpace(userID)
	role := auth.NormalizeSupplierRoleName(roleName)
	act := strings.ToUpper(strings.TrimSpace(action))
	if act == "" {
		act = RoleActionAssign
	}
	if tid == "" || uid == "" || role == "" {
		return nil, fmt.Errorf("%w: tenant_id, user_id, and role_name required", ErrInvalidIAMInput)
	}
	if act != RoleActionAssign && act != RoleActionReplace && act != RoleActionRevoke {
		return nil, fmt.Errorf("%w: unsupported role action %q", ErrInvalidIAMInput, action)
	}
	if err := r.EnsureTenantSeeded(ctx, tid); err != nil {
		return nil, err
	}

	var result []UserRoleAssignment
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		result = nil

		// Read RolePermissions for tenant inside txn.
		rolePerms := make(map[string]map[string]struct{})
		rpIter := txn.Query(txCtx, spanner.Statement{
			SQL:    `SELECT RoleName, PermissionKey FROM RolePermissions WHERE TenantId = @tid`,
			Params: map[string]any{"tid": tid},
		})
		for {
			row, err := rpIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				rpIter.Stop()
				return err
			}
			var rn, pk string
			if err := row.Columns(&rn, &pk); err != nil {
				rpIter.Stop()
				return err
			}
			if rolePerms[rn] == nil {
				rolePerms[rn] = make(map[string]struct{})
			}
			rolePerms[rn][pk] = struct{}{}
		}
		rpIter.Stop()

		if rolePerms[role] == nil {
			return fmt.Errorf("%w: unknown role %q", ErrInvalidIAMInput, roleName)
		}

		// Read existing SupplierUserRoles for (tid, uid).
		existing, err := readUserRolesInTxn(txCtx, txn, tid, uid)
		if err != nil {
			return err
		}

		// Read SodPolicies for tenant.
		policies, err := readSodPoliciesInTxn(txCtx, txn, tid)
		if err != nil {
			return err
		}

		now := time.Now().UTC()
		var target []UserRoleAssignment
		var muts []*spanner.Mutation

		switch act {
		case RoleActionRevoke:
			for _, a := range existing {
				if a.RoleName == role {
					muts = append(muts, spanner.Delete("SupplierUserRoles", spanner.Key{tid, uid, role}))
				} else {
					target = append(target, a)
				}
			}
		case RoleActionReplace:
			createdAt := any(spanner.CommitTimestamp)
			var preservedExp *time.Time
			for _, a := range existing {
				if a.RoleName != role {
					muts = append(muts, spanner.Delete("SupplierUserRoles", spanner.Key{tid, uid, a.RoleName}))
				} else {
					if !a.CreatedAt.IsZero() {
						createdAt = a.CreatedAt
					}
					preservedExp = a.ExpiresAt
				}
			}
			rowMap := map[string]any{
				"TenantId":   tid,
				"UserId":     uid,
				"RoleName":   role,
				"AssignedBy": assignedBy,
				"CreatedAt":  createdAt,
				"UpdatedAt":  spanner.CommitTimestamp,
			}
			if preservedExp != nil {
				rowMap["ExpiresAt"] = preservedExp.UTC()
			}
			muts = append(muts, spanner.InsertOrUpdateMap("SupplierUserRoles", rowMap))
			target = []UserRoleAssignment{{
				TenantID: tid, UserID: uid, RoleName: role, AssignedBy: assignedBy, ExpiresAt: preservedExp, CreatedAt: now, UpdatedAt: now,
			}}
		case RoleActionAssign:
			createdAt := any(spanner.CommitTimestamp)
			var preservedExp *time.Time
			alreadyHeld := false
			for _, a := range existing {
				if a.RoleName == role {
					alreadyHeld = true
					if !a.CreatedAt.IsZero() {
						createdAt = a.CreatedAt
					}
					preservedExp = a.ExpiresAt
					a.AssignedBy = assignedBy
					a.UpdatedAt = now
				}
				target = append(target, a)
			}
			if !alreadyHeld {
				target = append(target, UserRoleAssignment{
					TenantID: tid, UserID: uid, RoleName: role, AssignedBy: assignedBy, CreatedAt: now, UpdatedAt: now,
				})
			}
			rowMap := map[string]any{
				"TenantId":   tid,
				"UserId":     uid,
				"RoleName":   role,
				"AssignedBy": assignedBy,
				"CreatedAt":  createdAt,
				"UpdatedAt":  spanner.CommitTimestamp,
			}
			if preservedExp != nil {
				rowMap["ExpiresAt"] = preservedExp.UTC()
			}
			muts = append(muts, spanner.InsertOrUpdateMap("SupplierUserRoles", rowMap))
		}

		if act != RoleActionRevoke {
			roSet, roErr := readReadOnlyCustomRolesInTxn(txCtx, txn, tid)
			if roErr != nil {
				return roErr
			}
			roleNames := filterActiveRoleNames(target, now)
			if err := checkStaticRoleSoDWithMeta(roleNames, rolePerms, policies, roSet); err != nil {
				return err
			}
		}

		buf := &tenantOutboxBuffer{inner: outbox.NewSpannerTxnBuffer(txn), tenantID: tid}
		if err := outbox.EmitJSON(txCtx, buf, events.AggregateSupplierRole, tid+":"+uid+":"+role, events.TopicLedger, events.IAMRoleChangedEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventIAMRoleChanged, Timestamp: now.Format(time.RFC3339Nano)},
			TenantID:   tid,
			UserID:     uid,
			RoleName:   role,
			Action:     act,
			AssignedBy: assignedBy,
			TraceID:    traceID,
		}); err != nil {
			return fmt.Errorf("emit iam.role.changed: %w", err)
		}
		if err := buf.inner.Flush(txCtx); err != nil {
			return fmt.Errorf("flush iam.role.changed outbox: %w", err)
		}
		if len(muts) > 0 {
			if err := txn.BufferWrite(muts); err != nil {
				return fmt.Errorf("write SupplierUserRoles: %w", err)
			}
		}
		if auditFn != nil {
			if err := auditFn(txCtx, txn); err != nil {
				return fmt.Errorf("audit iam.role.changed: %w", err)
			}
		}
		sort.Slice(target, func(i, j int) bool { return target[i].RoleName < target[j].RoleName })
		result = target
		return nil
	})
	if err != nil {
		return nil, err
	}
	r.seededMu.Lock()
	r.seededCache[tid] = true
	r.seededMu.Unlock()
	return result, nil
}

func readSodPoliciesInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, tid string) ([]SodPolicy, error) {
	iter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT TenantId, PolicyId, Name, PermissionA, PermissionB, EnforceSameObject, Enabled, CreatedAt
		      FROM SodPolicies WHERE TenantId = @tid ORDER BY PolicyId`,
		Params: map[string]any{"tid": tid},
	})
	defer iter.Stop()
	var out []SodPolicy
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read SodPolicies: %w", err)
		}
		var p SodPolicy
		if err := row.Columns(&p.TenantID, &p.PolicyID, &p.Name, &p.PermissionA, &p.PermissionB, &p.EnforceSameObject, &p.Enabled, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.CreatedAt = p.CreatedAt.UTC()
		out = append(out, p)
	}
	return out, nil
}

func (r *SpannerRepository) ListSodPolicies(ctx context.Context, tenantID string) ([]SodPolicy, error) {
	if r == nil || r.client == nil {
		return nil, ErrIAMUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	if tid != "" {
		if err := r.EnsureTenantSeeded(ctx, tid); err != nil {
			return nil, err
		}
	}

	sql := `SELECT TenantId, PolicyId, Name, PermissionA, PermissionB, EnforceSameObject, Enabled, CreatedAt FROM SodPolicies`
	params := map[string]any{}
	if tid != "" {
		sql += ` WHERE TenantId = @tid`
		params["tid"] = tid
	}
	sql += ` ORDER BY TenantId, PolicyId`

	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL:    sql,
		Params: params,
	})
	defer iter.Stop()

	var out []SodPolicy
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list SodPolicies: %w", err)
		}
		var p SodPolicy
		if err := row.Columns(&p.TenantID, &p.PolicyID, &p.Name, &p.PermissionA, &p.PermissionB, &p.EnforceSameObject, &p.Enabled, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.CreatedAt = p.CreatedAt.UTC()
		out = append(out, p)
	}
	if len(out) == 0 && tid == "" {
		return DefaultSodPolicies("default", time.Now().UTC()), nil
	}
	if out == nil {
		out = []SodPolicy{}
	}
	return out, nil
}

func (r *SpannerRepository) UpdateSodPolicy(
	ctx context.Context,
	in UpdatePolicyInput,
	auditFn func(context.Context, *spanner.ReadWriteTransaction) error,
) (SodPolicy, error) {
	if r == nil || r.client == nil {
		return SodPolicy{}, ErrIAMUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	pid := strings.TrimSpace(in.PolicyID)
	if tid == "" || pid == "" {
		return SodPolicy{}, fmt.Errorf("%w: tenant_id and policy_id required", ErrInvalidIAMInput)
	}
	if err := r.EnsureTenantSeeded(ctx, tid); err != nil {
		return SodPolicy{}, err
	}

	var updated SodPolicy
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(txCtx, "SodPolicies", spanner.Key{tid, pid}, []string{
			"TenantId", "PolicyId", "Name", "PermissionA", "PermissionB", "EnforceSameObject", "Enabled", "CreatedAt",
		})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return ErrPolicyNotFound
			}
			return fmt.Errorf("read SodPolicy: %w", err)
		}
		var cur SodPolicy
		if err := row.Columns(&cur.TenantID, &cur.PolicyID, &cur.Name, &cur.PermissionA, &cur.PermissionB, &cur.EnforceSameObject, &cur.Enabled, &cur.CreatedAt); err != nil {
			return err
		}
		cur.CreatedAt = cur.CreatedAt.UTC()
		if in.Name != nil && strings.TrimSpace(*in.Name) != "" {
			cur.Name = strings.TrimSpace(*in.Name)
		}
		if in.PermissionA != nil && strings.TrimSpace(*in.PermissionA) != "" {
			cur.PermissionA = strings.ToLower(strings.TrimSpace(*in.PermissionA))
		}
		if in.PermissionB != nil && strings.TrimSpace(*in.PermissionB) != "" {
			cur.PermissionB = strings.ToLower(strings.TrimSpace(*in.PermissionB))
		}
		if in.EnforceSameObject != nil {
			cur.EnforceSameObject = *in.EnforceSameObject
		}
		if in.Enabled != nil {
			cur.Enabled = *in.Enabled
		}
		if cur.PermissionA == cur.PermissionB {
			return fmt.Errorf("%w: permission_a and permission_b must differ", ErrInvalidIAMInput)
		}

		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("SodPolicies", map[string]any{
				"TenantId":          tid,
				"PolicyId":          pid,
				"Name":              cur.Name,
				"PermissionA":       cur.PermissionA,
				"PermissionB":       cur.PermissionB,
				"EnforceSameObject": cur.EnforceSameObject,
				"Enabled":           cur.Enabled,
			}),
		}); err != nil {
			return fmt.Errorf("update SodPolicies: %w", err)
		}
		if auditFn != nil {
			if err := auditFn(txCtx, txn); err != nil {
				return fmt.Errorf("audit sod policy update: %w", err)
			}
		}
		updated = cur
		return nil
	})
	if err != nil {
		return SodPolicy{}, err
	}
	return updated, nil
}

func (r *SpannerRepository) RecordSodViolation(
	ctx context.Context,
	v SodViolation,
	auditFn func(context.Context, *spanner.ReadWriteTransaction) error,
) error {
	if r == nil || r.client == nil {
		return ErrIAMUnavailable
	}
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		row := map[string]any{
			"TenantId":    v.TenantID,
			"ViolationId": v.ViolationID,
			"ActorId":     v.ActorID,
			"Action":      v.Action,
			"RuleCode":    v.RuleCode,
			"CreatedAt":   spanner.CommitTimestamp,
		}
		if v.ObjectType != "" {
			row["ObjectType"] = v.ObjectType
		}
		if v.ObjectID != "" {
			row["ObjectId"] = v.ObjectID
		}
		if v.MakerID != "" {
			row["MakerId"] = v.MakerID
		}
		if v.RoutePath != "" {
			row["RoutePath"] = v.RoutePath
		}
		if v.TraceID != "" {
			row["TraceId"] = v.TraceID
		}

		if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertMap("SodViolations", row)}); err != nil {
			return fmt.Errorf("insert SodViolations: %w", err)
		}

		buf := &tenantOutboxBuffer{inner: outbox.NewSpannerTxnBuffer(txn), tenantID: v.TenantID}
		if err := outbox.EmitJSON(txCtx, buf, events.AggregateSodViolation, v.ViolationID, events.TopicLedger, events.SodViolationDetectedEvent{
			BaseEvent:   events.BaseEvent{Type: events.EventSodViolationDetected, Timestamp: v.CreatedAt.UTC().Format(time.RFC3339Nano)},
			TenantID:    v.TenantID,
			ViolationID: v.ViolationID,
			ActorID:     v.ActorID,
			Action:      v.Action,
			ObjectType:  v.ObjectType,
			ObjectID:    v.ObjectID,
			MakerID:     v.MakerID,
			RuleCode:    v.RuleCode,
			RoutePath:   v.RoutePath,
			TraceID:     v.TraceID,
		}); err != nil {
			return fmt.Errorf("emit sod.violation.detected: %w", err)
		}
		if err := buf.inner.Flush(txCtx); err != nil {
			return fmt.Errorf("flush sod violation outbox: %w", err)
		}
		if auditFn != nil {
			if err := auditFn(txCtx, txn); err != nil {
				return fmt.Errorf("audit sod violation: %w", err)
			}
		}
		return nil
	})
}

func (r *SpannerRepository) ListSodViolations(ctx context.Context, tenantID, actorID string, limit int) ([]SodViolation, error) {
	if r == nil || r.client == nil {
		return nil, ErrIAMUnavailable
	}
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	tid := strings.TrimSpace(tenantID)
	actor := strings.TrimSpace(actorID)

	sql := `SELECT TenantId, ViolationId, ActorId, Action, COALESCE(ObjectType, ''), COALESCE(ObjectId, ''),
	               COALESCE(MakerId, ''), RuleCode, COALESCE(RoutePath, ''), COALESCE(TraceId, ''), CreatedAt
	        FROM SodViolations`
	params := map[string]any{"lim": int64(limit)}
	var conds []string
	if tid != "" {
		conds = append(conds, `TenantId = @tid`)
		params["tid"] = tid
	}
	if actor != "" {
		conds = append(conds, `ActorId = @actor`)
		params["actor"] = actor
	}
	if len(conds) > 0 {
		sql += ` WHERE ` + strings.Join(conds, ` AND `)
	}
	sql += ` ORDER BY CreatedAt DESC LIMIT @lim`

	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL:    sql,
		Params: params,
	})
	defer iter.Stop()

	var out []SodViolation
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list SodViolations: %w", err)
		}
		var v SodViolation
		if err := row.Columns(&v.TenantID, &v.ViolationID, &v.ActorID, &v.Action, &v.ObjectType, &v.ObjectID,
			&v.MakerID, &v.RuleCode, &v.RoutePath, &v.TraceID, &v.CreatedAt); err != nil {
			return nil, err
		}
		v.CreatedAt = v.CreatedAt.UTC()
		out = append(out, v)
	}
	if out == nil {
		out = []SodViolation{}
	}
	return out, nil
}

func readUserRolesInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, tid, uid string) ([]UserRoleAssignment, error) {
	iter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT TenantId, UserId, RoleName, COALESCE(AssignedBy, ''), ExpiresAt, CreatedAt, UpdatedAt
		      FROM SupplierUserRoles WHERE TenantId = @tid AND UserId = @uid`,
		Params: map[string]any{"tid": tid, "uid": uid},
	})
	defer iter.Stop()
	var existing []UserRoleAssignment
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var a UserRoleAssignment
		var expAt spanner.NullTime
		if err := row.Columns(&a.TenantID, &a.UserID, &a.RoleName, &a.AssignedBy, &expAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		if expAt.Valid {
			t := expAt.Time.UTC()
			a.ExpiresAt = &t
		}
		existing = append(existing, a)
	}
	return existing, nil
}

func (r *SpannerRepository) GrantAuditorRole(
	ctx context.Context,
	tenantID, userID, assignedBy, _ string, reason, traceID string,
	expiresAt time.Time,
	auditFn func(context.Context, *spanner.ReadWriteTransaction) error,
) (AuditorGrant, error) {
	if r == nil || r.client == nil {
		return AuditorGrant{}, ErrIAMUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	uid := strings.TrimSpace(userID)
	if tid == "" || uid == "" || expiresAt.IsZero() {
		return AuditorGrant{}, fmt.Errorf("%w: tenant_id, user_id, and expires_at required", ErrInvalidIAMInput)
	}
	if err := r.EnsureTenantSeeded(ctx, tid); err != nil {
		return AuditorGrant{}, err
	}

	expUTC := expiresAt.UTC()
	var result AuditorGrant
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		result = AuditorGrant{}
		existing, err := readUserRolesInTxn(txCtx, txn, tid, uid)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if err := checkAuditorGrantEligibility(existing, now); err != nil {
			return err
		}

		createdAt := any(spanner.CommitTimestamp)
		grantCreated := now
		for _, a := range existing {
			if a.RoleName == auth.SupplierRoleAuditor && !a.CreatedAt.IsZero() {
				createdAt = a.CreatedAt
				grantCreated = a.CreatedAt.UTC()
				break
			}
		}
		mut := spanner.InsertOrUpdateMap("SupplierUserRoles", map[string]any{
			"TenantId":   tid,
			"UserId":     uid,
			"RoleName":   auth.SupplierRoleAuditor,
			"AssignedBy": strings.TrimSpace(assignedBy),
			"ExpiresAt":  expUTC,
			"CreatedAt":  createdAt,
			"UpdatedAt":  spanner.CommitTimestamp,
		})

		buf := &tenantOutboxBuffer{inner: outbox.NewSpannerTxnBuffer(txn), tenantID: tid}
		if err := outbox.EmitJSON(txCtx, buf, events.AggregateSupplierRole, tid+":"+uid+":"+auth.SupplierRoleAuditor, events.TopicLedger, events.AuditorAccessGrantedEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventAuditorAccessGranted, Timestamp: now.Format(time.RFC3339Nano)},
			TenantID:   tid,
			UserID:     uid,
			RoleName:   auth.SupplierRoleAuditor,
			Action:     RoleActionAssign,
			GrantedBy:  strings.TrimSpace(assignedBy),
			ExpiresAt:  expUTC.Format(time.RFC3339Nano),
			Reason:     strings.TrimSpace(reason),
			TraceID:    traceID,
			OccurredAt: now.Format(time.RFC3339Nano),
		}); err != nil {
			return fmt.Errorf("emit iam.auditor_access.granted: %w", err)
		}
		if err := buf.inner.Flush(txCtx); err != nil {
			return fmt.Errorf("flush iam.auditor_access.granted outbox: %w", err)
		}
		if err := txn.BufferWrite([]*spanner.Mutation{mut}); err != nil {
			return fmt.Errorf("write SupplierUserRoles auditor grant: %w", err)
		}
		if auditFn != nil {
			if err := auditFn(txCtx, txn); err != nil {
				return fmt.Errorf("audit iam.auditor_access.granted: %w", err)
			}
		}

		status := "ACTIVE"
		if !expUTC.After(now) {
			status = "EXPIRED"
		}
		result = AuditorGrant{
			TenantID:   tid,
			UserID:     uid,
			RoleName:   auth.SupplierRoleAuditor,
			AssignedBy: strings.TrimSpace(assignedBy),
			GrantedBy:  strings.TrimSpace(assignedBy),
			Status:     status,
			ExpiresAt:  expUTC,
			CreatedAt:  grantCreated,
			UpdatedAt:  now,
		}
		return nil
	})
	if err != nil {
		return AuditorGrant{}, err
	}
	return result, nil
}

func (r *SpannerRepository) ListAuditorGrants(ctx context.Context, tenantID string) ([]AuditorGrant, error) {
	if r == nil || r.client == nil {
		return nil, ErrIAMUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	sql := `SELECT TenantId, UserId, RoleName, COALESCE(AssignedBy, ''), ExpiresAt, CreatedAt, UpdatedAt
	        FROM SupplierUserRoles WHERE RoleName = @role`
	params := map[string]any{"role": auth.SupplierRoleAuditor}
	if tid != "" {
		sql += ` AND TenantId = @tid`
		params["tid"] = tid
	}
	sql += ` ORDER BY UpdatedAt DESC, UserId ASC`

	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL:    sql,
		Params: params,
	})
	defer iter.Stop()

	now := time.Now().UTC()
	var out []AuditorGrant
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list auditor grants: %w", err)
		}
		var g AuditorGrant
		var expAt spanner.NullTime
		if err := row.Columns(&g.TenantID, &g.UserID, &g.RoleName, &g.AssignedBy, &expAt, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		g.GrantedBy = g.AssignedBy
		g.CreatedAt = g.CreatedAt.UTC()
		g.UpdatedAt = g.UpdatedAt.UTC()
		g.Status = "ACTIVE"
		if expAt.Valid {
			g.ExpiresAt = expAt.Time.UTC()
			if !g.ExpiresAt.After(now) {
				g.Status = "EXPIRED"
			}
		}
		out = append(out, g)
	}
	if out == nil {
		out = []AuditorGrant{}
	}
	return out, nil
}

func (r *SpannerRepository) RevokeAuditorRole(
	ctx context.Context,
	tenantID, userID, revokedBy, _ string, traceID string,
	auditFn func(context.Context, *spanner.ReadWriteTransaction) error,
) (bool, error) {
	if r == nil || r.client == nil {
		return false, ErrIAMUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	uid := strings.TrimSpace(userID)
	if tid == "" || uid == "" {
		return false, fmt.Errorf("%w: tenant_id and user_id required", ErrInvalidIAMInput)
	}

	var revoked bool
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		revoked = false
		existing, err := readUserRolesInTxn(txCtx, txn, tid, uid)
		if err != nil {
			return err
		}
		hasAuditor := false
		for _, a := range existing {
			if a.RoleName == auth.SupplierRoleAuditor {
				hasAuditor = true
				break
			}
		}
		if !hasAuditor {
			return nil
		}

		now := time.Now().UTC()
		mut := spanner.Delete("SupplierUserRoles", spanner.Key{tid, uid, auth.SupplierRoleAuditor})
		if err := txn.BufferWrite([]*spanner.Mutation{mut}); err != nil {
			return fmt.Errorf("delete SupplierUserRoles auditor grant: %w", err)
		}

		buf := &tenantOutboxBuffer{inner: outbox.NewSpannerTxnBuffer(txn), tenantID: tid}
		if err := outbox.EmitJSON(txCtx, buf, events.AggregateSupplierRole, tid+":"+uid+":"+auth.SupplierRoleAuditor, events.TopicLedger, events.IAMRoleChangedEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventIAMRoleChanged, Timestamp: now.Format(time.RFC3339Nano)},
			TenantID:   tid,
			UserID:     uid,
			RoleName:   auth.SupplierRoleAuditor,
			Action:     RoleActionRevoke,
			AssignedBy: strings.TrimSpace(revokedBy),
			TraceID:    traceID,
		}); err != nil {
			return fmt.Errorf("emit iam.role.changed revoke: %w", err)
		}
		if err := buf.inner.Flush(txCtx); err != nil {
			return fmt.Errorf("flush iam.role.changed revoke outbox: %w", err)
		}
		if auditFn != nil {
			if err := auditFn(txCtx, txn); err != nil {
				return fmt.Errorf("audit iam.auditor_access.revoked: %w", err)
			}
		}
		revoked = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return revoked, nil
}

func readReadOnlyCustomRolesInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, tid string) (map[string]bool, error) {
	out := make(map[string]bool)
	iter := txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT RoleName FROM CustomRoles WHERE TenantId = @tid AND Status = 'ACTIVE' AND ReadOnly = TRUE`,
		Params: map[string]any{"tid": tid},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read read-only custom roles: %w", err)
		}
		var rn string
		if err := row.Columns(&rn); err != nil {
			return nil, err
		}
		out[strings.ToUpper(strings.TrimSpace(rn))] = true
	}
	return out, nil
}

func (r *SpannerRepository) ListCustomRoles(ctx context.Context, tenantID string) ([]CustomRole, error) {
	if r == nil || r.client == nil {
		return nil, ErrIAMUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	if tid == "" {
		return nil, fmt.Errorf("%w: tenant_id required", ErrInvalidIAMInput)
	}
	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT TenantId, RoleName, COALESCE(Description, ''), PermissionsJson, ReadOnly, CreatedBy, Status, CreatedAt, UpdatedAt
		      FROM CustomRoles
		      WHERE TenantId = @tid AND Status = 'ACTIVE'
		      ORDER BY RoleName ASC`,
		Params: map[string]any{"tid": tid},
	})
	defer iter.Stop()

	var out []CustomRole
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list CustomRoles: %w", err)
		}
		cr, err := scanCustomRoleRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, cr)
	}
	if out == nil {
		out = []CustomRole{}
	}
	return out, nil
}

func (r *SpannerRepository) GetCustomRole(ctx context.Context, tenantID, roleName string) (CustomRole, bool, error) {
	if r == nil || r.client == nil {
		return CustomRole{}, false, ErrIAMUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	rn := strings.ToUpper(strings.TrimSpace(roleName))
	if tid == "" || rn == "" {
		return CustomRole{}, false, nil
	}
	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()
	row, err := ro.ReadRow(ctx, "CustomRoles", spanner.Key{tid, rn}, []string{
		"TenantId", "RoleName", "Description", "PermissionsJson", "ReadOnly", "CreatedBy", "Status", "CreatedAt", "UpdatedAt",
	})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return CustomRole{}, false, nil
		}
		return CustomRole{}, false, fmt.Errorf("read CustomRoles: %w", err)
	}
	cr, err := scanCustomRoleRow(row)
	if err != nil {
		return CustomRole{}, false, err
	}
	return cr, true, nil
}

func (r *SpannerRepository) UpsertCustomRole(
	ctx context.Context,
	role CustomRole,
	auditFn func(context.Context, *spanner.ReadWriteTransaction) error,
) (CustomRole, error) {
	if r == nil || r.client == nil {
		return CustomRole{}, ErrIAMUnavailable
	}
	tid := strings.TrimSpace(role.TenantID)
	rn := strings.ToUpper(strings.TrimSpace(role.RoleName))
	if tid == "" || rn == "" {
		return CustomRole{}, fmt.Errorf("%w: tenant_id and role_name required", ErrInvalidIAMInput)
	}
	if err := r.EnsureTenantSeeded(ctx, tid); err != nil {
		return CustomRole{}, err
	}

	permsRaw, err := json.Marshal(role.Permissions)
	if err != nil {
		return CustomRole{}, fmt.Errorf("marshal custom role permissions: %w", err)
	}

	var saved CustomRole
	err = spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		createdAt := any(spanner.CommitTimestamp)
		prevCreated := role.CreatedAt
		if existingRow, rerr := txn.ReadRow(txCtx, "CustomRoles", spanner.Key{tid, rn}, []string{"Status", "CreatedAt"}); rerr == nil {
			var st string
			var ct time.Time
			if cerr := existingRow.Columns(&st, &ct); cerr == nil {
				if strings.EqualFold(st, "ACTIVE") {
					return fmt.Errorf("%w: custom role %q already exists", ErrCustomRoleExists, rn)
				}
				if !ct.IsZero() {
					createdAt = ct
					prevCreated = ct.UTC()
				}
			}
		} else if spanner.ErrCode(rerr) != codes.NotFound {
			return fmt.Errorf("check CustomRoles: %w", rerr)
		}

		// Sync RolePermissions: delete any stale rows for (tid, rn) and insert current permission set.
		var muts []*spanner.Mutation
		delIter := txn.Query(txCtx, spanner.Statement{
			SQL:    `SELECT PermissionKey FROM RolePermissions WHERE TenantId = @tid AND RoleName = @rn`,
			Params: map[string]any{"tid": tid, "rn": rn},
		})
		for {
			drow, derr := delIter.Next()
			if derr == iterator.Done {
				break
			}
			if derr != nil {
				delIter.Stop()
				return fmt.Errorf("read existing RolePermissions for custom role: %w", derr)
			}
			var pk string
			if cerr := drow.Columns(&pk); cerr != nil {
				delIter.Stop()
				return cerr
			}
			muts = append(muts, spanner.Delete("RolePermissions", spanner.Key{tid, rn, pk}))
		}
		delIter.Stop()

		muts = append(muts, spanner.InsertOrUpdateMap("CustomRoles", map[string]any{
			"TenantId":        tid,
			"RoleName":        rn,
			"Description":     strings.TrimSpace(role.Description),
			"PermissionsJson": string(permsRaw),
			"ReadOnly":        role.ReadOnly,
			"CreatedBy":       strings.TrimSpace(role.CreatedBy),
			"Status":          "ACTIVE",
			"CreatedAt":       createdAt,
			"UpdatedAt":       spanner.CommitTimestamp,
		}))

		for _, perm := range role.Permissions {
			muts = append(muts, spanner.InsertOrUpdateMap("RolePermissions", map[string]any{
				"TenantId":      tid,
				"RoleName":      rn,
				"PermissionKey": perm,
				"GrantedBy":     strings.TrimSpace(role.CreatedBy),
				"CreatedAt":     spanner.CommitTimestamp,
			}))
		}

		if err := txn.BufferWrite(muts); err != nil {
			return fmt.Errorf("write CustomRoles + RolePermissions: %w", err)
		}
		if auditFn != nil {
			if err := auditFn(txCtx, txn); err != nil {
				return fmt.Errorf("audit custom role upsert: %w", err)
			}
		}

		now := time.Now().UTC()
		saved = role
		saved.TenantID = tid
		saved.RoleName = rn
		saved.Status = "ACTIVE"
		if !prevCreated.IsZero() {
			saved.CreatedAt = prevCreated
		} else {
			saved.CreatedAt = now
		}
		saved.UpdatedAt = now
		return nil
	})
	if err != nil {
		return CustomRole{}, err
	}
	return saved, nil
}

func (r *SpannerRepository) DeleteCustomRole(
	ctx context.Context,
	tenantID, roleName string,
	auditFn func(context.Context, *spanner.ReadWriteTransaction) error,
) (bool, error) {
	if r == nil || r.client == nil {
		return false, ErrIAMUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	rn := strings.ToUpper(strings.TrimSpace(roleName))
	if tid == "" || rn == "" {
		return false, fmt.Errorf("%w: tenant_id and role_name required", ErrInvalidIAMInput)
	}

	var deleted bool
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		deleted = false
		row, rerr := txn.ReadRow(txCtx, "CustomRoles", spanner.Key{tid, rn}, []string{"Status"})
		if rerr != nil {
			if spanner.ErrCode(rerr) == codes.NotFound {
				return nil
			}
			return fmt.Errorf("read CustomRoles: %w", rerr)
		}
		var st string
		if cerr := row.Columns(&st); cerr != nil {
			return cerr
		}
		if !strings.EqualFold(st, "ACTIVE") {
			return nil
		}

		muts := []*spanner.Mutation{
			spanner.UpdateMap("CustomRoles", map[string]any{
				"TenantId":  tid,
				"RoleName":  rn,
				"Status":    "DELETED",
				"UpdatedAt": spanner.CommitTimestamp,
			}),
		}

		rpIter := txn.Query(txCtx, spanner.Statement{
			SQL:    `SELECT PermissionKey FROM RolePermissions WHERE TenantId = @tid AND RoleName = @rn`,
			Params: map[string]any{"tid": tid, "rn": rn},
		})
		for {
			rrow, derr := rpIter.Next()
			if derr == iterator.Done {
				break
			}
			if derr != nil {
				rpIter.Stop()
				return fmt.Errorf("read RolePermissions for custom role delete: %w", derr)
			}
			var pk string
			if cerr := rrow.Columns(&pk); cerr != nil {
				rpIter.Stop()
				return cerr
			}
			muts = append(muts, spanner.Delete("RolePermissions", spanner.Key{tid, rn, pk}))
		}
		rpIter.Stop()

		urIter := txn.Query(txCtx, spanner.Statement{
			SQL:    `SELECT UserId FROM SupplierUserRoles WHERE TenantId = @tid AND RoleName = @rn`,
			Params: map[string]any{"tid": tid, "rn": rn},
		})
		for {
			urow, derr := urIter.Next()
			if derr == iterator.Done {
				break
			}
			if derr != nil {
				urIter.Stop()
				return fmt.Errorf("read SupplierUserRoles for custom role delete: %w", derr)
			}
			var uid string
			if cerr := urow.Columns(&uid); cerr != nil {
				urIter.Stop()
				return cerr
			}
			muts = append(muts, spanner.Delete("SupplierUserRoles", spanner.Key{tid, uid, rn}))
		}
		urIter.Stop()

		if err := txn.BufferWrite(muts); err != nil {
			return fmt.Errorf("delete CustomRoles + sync RolePermissions: %w", err)
		}
		if auditFn != nil {
			if err := auditFn(txCtx, txn); err != nil {
				return fmt.Errorf("audit custom role delete: %w", err)
			}
		}
		deleted = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return deleted, nil
}

func scanCustomRoleRow(row *spanner.Row) (CustomRole, error) {
	var cr CustomRole
	var desc spanner.NullString
	var permsJSON string
	if err := row.Columns(&cr.TenantID, &cr.RoleName, &desc, &permsJSON, &cr.ReadOnly, &cr.CreatedBy, &cr.Status, &cr.CreatedAt, &cr.UpdatedAt); err != nil {
		return CustomRole{}, fmt.Errorf("scan CustomRoles row: %w", err)
	}
	cr.Description = desc.StringVal
	cr.CreatedAt = cr.CreatedAt.UTC()
	cr.UpdatedAt = cr.UpdatedAt.UTC()
	if strings.TrimSpace(permsJSON) != "" {
		_ = json.Unmarshal([]byte(permsJSON), &cr.Permissions)
	}
	if cr.Permissions == nil {
		cr.Permissions = []string{}
	}
	return cr, nil
}

func readRolePermsInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, tid string) (map[string]map[string]struct{}, error) {
	rolePerms := make(map[string]map[string]struct{})
	iter := txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT RoleName, PermissionKey FROM RolePermissions WHERE TenantId = @tid`,
		Params: map[string]any{"tid": tid},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read RolePermissions in txn: %w", err)
		}
		var rn, pk string
		if err := row.Columns(&rn, &pk); err != nil {
			return nil, err
		}
		if rolePerms[rn] == nil {
			rolePerms[rn] = make(map[string]struct{})
		}
		rolePerms[rn][pk] = struct{}{}
	}
	return rolePerms, nil
}

func (r *SpannerRepository) ListScimResources(ctx context.Context, tenantID, resourceType, filter string, startIndex, count int) ([]ScimResourceRecord, int, error) {
	if r == nil || r.client == nil {
		return nil, 0, ErrIAMUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	rt := strings.TrimSpace(resourceType)
	if tid == "" {
		return nil, 0, fmt.Errorf("%w: tenant_id required", ErrInvalidScimRequest)
	}
	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()

	sql := `SELECT TenantId, ResourceType, ResourceId, COALESCE(ExternalId, ''), COALESCE(UserName, ''),
	               COALESCE(DisplayName, ''), COALESCE(Email, ''), Active,
	               COALESCE(RolesJson, '[]'), COALESCE(MembersJson, '[]'), COALESCE(MetaJson, '{}'),
	               CreatedAt, UpdatedAt
	        FROM ScimIdentities WHERE TenantId = @tid`
	params := map[string]any{"tid": tid}
	if rt != "" {
		sql += ` AND ResourceType = @rt`
		params["rt"] = rt
	}
	sql += ` ORDER BY ResourceType ASC, ResourceId ASC`

	iter := ro.Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	var matched []ScimResourceRecord
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, 0, fmt.Errorf("list ScimIdentities: %w", err)
		}
		rec, err := scanScimIdentityRow(row)
		if err != nil {
			return nil, 0, err
		}
		if !matchesScimFilter(rec, filter) {
			continue
		}
		matched = append(matched, rec)
	}
	total := len(matched)
	if startIndex < 1 {
		startIndex = 1
	}
	startOffset := startIndex - 1
	if startOffset >= total || count == 0 {
		return []ScimResourceRecord{}, total, nil
	}
	endOffset := total
	if count > 0 && startOffset+count < endOffset {
		endOffset = startOffset + count
	}
	return matched[startOffset:endOffset], total, nil
}

func (r *SpannerRepository) GetScimResource(ctx context.Context, tenantID, resourceType, resourceID string) (ScimResourceRecord, bool, error) {
	if r == nil || r.client == nil {
		return ScimResourceRecord{}, false, ErrIAMUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	rt := strings.TrimSpace(resourceType)
	rid := strings.TrimSpace(resourceID)
	if tid == "" || rt == "" || rid == "" {
		return ScimResourceRecord{}, false, nil
	}
	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()
	row, err := ro.ReadRow(ctx, "ScimIdentities", spanner.Key{tid, rt, rid}, []string{
		"TenantId", "ResourceType", "ResourceId", "ExternalId", "UserName",
		"DisplayName", "Email", "Active", "RolesJson", "MembersJson", "MetaJson",
		"CreatedAt", "UpdatedAt",
	})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return ScimResourceRecord{}, false, nil
		}
		return ScimResourceRecord{}, false, fmt.Errorf("read ScimIdentities: %w", err)
	}
	rec, err := scanScimIdentityRow(row)
	if err != nil {
		return ScimResourceRecord{}, false, err
	}
	return rec, true, nil
}

func (r *SpannerRepository) UpsertScimResource(
	ctx context.Context,
	rec ScimResourceRecord,
	action, actorID, traceID string,
	auditFn func(context.Context, *spanner.ReadWriteTransaction) error,
) (ScimResourceRecord, []string, error) {
	if r == nil || r.client == nil {
		return ScimResourceRecord{}, nil, ErrIAMUnavailable
	}
	tid := strings.TrimSpace(rec.TenantID)
	rt := strings.TrimSpace(rec.ResourceType)
	rid := strings.TrimSpace(rec.ResourceID)
	if tid == "" || rt == "" || rid == "" {
		return ScimResourceRecord{}, nil, fmt.Errorf("%w: tenant_id, resource_type, and resource_id required", ErrInvalidScimRequest)
	}
	if err := r.EnsureTenantSeeded(ctx, tid); err != nil {
		return ScimResourceRecord{}, nil, err
	}

	var saved ScimResourceRecord
	var rolesSynced []string

	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		saved = ScimResourceRecord{}
		rolesSynced = nil
		now := time.Now().UTC()

		rolePerms, err := readRolePermsInTxn(txCtx, txn, tid)
		if err != nil {
			return err
		}
		policies, err := readSodPoliciesInTxn(txCtx, txn, tid)
		if err != nil {
			return err
		}
		roSet, err := readReadOnlyCustomRolesInTxn(txCtx, txn, tid)
		if err != nil {
			return err
		}

		var prev ScimResourceRecord
		hadPrev := false
		createdAt := any(spanner.CommitTimestamp)
		prevRow, rerr := txn.ReadRow(txCtx, "ScimIdentities", spanner.Key{tid, rt, rid}, []string{
			"TenantId", "ResourceType", "ResourceId", "ExternalId", "UserName",
			"DisplayName", "Email", "Active", "RolesJson", "MembersJson", "MetaJson",
			"CreatedAt", "UpdatedAt",
		})
		if rerr == nil {
			if pRec, perr := scanScimIdentityRow(prevRow); perr == nil {
				prev = pRec
				hadPrev = true
				if !pRec.CreatedAt.IsZero() {
					createdAt = pRec.CreatedAt
					rec.CreatedAt = pRec.CreatedAt
					rec.Meta.Created = pRec.CreatedAt.UTC().Format(time.RFC3339Nano)
				}
			}
		} else if spanner.ErrCode(rerr) != codes.NotFound {
			return fmt.Errorf("read existing ScimIdentities: %w", rerr)
		}

		var muts []*spanner.Mutation

		switch rt {
		case ScimResourceTypeUser:
			if strings.EqualFold(action, "CREATE") {
				dupIter := txn.Query(txCtx, spanner.Statement{
					SQL: `SELECT ResourceId FROM ScimIdentities
					      WHERE TenantId = @tid AND ResourceType = 'User' AND LOWER(UserName) = LOWER(@un)
					      LIMIT 1`,
					Params: map[string]any{"tid": tid, "un": rec.UserName},
				})
				dRow, dErr := dupIter.Next()
				dupIter.Stop()
				if dErr == nil {
					var existingID string
					_ = dRow.Columns(&existingID)
					if existingID != rid {
						return fmt.Errorf("%w: userName %q already exists", ErrScimConflict, rec.UserName)
					}
				} else if dErr != iterator.Done {
					return fmt.Errorf("check ScimIdentities userName uniqueness: %w", dErr)
				}
			}

			for _, rr := range rec.Roles {
				rn := auth.NormalizeSupplierRoleName(rr.Value)
				if rolePerms[rn] == nil {
					return fmt.Errorf("%w: unknown role %q", ErrInvalidScimRequest, rr.Value)
				}
				rolesSynced = append(rolesSynced, rn)
			}

			existingRoles, err := readUserRolesInTxn(txCtx, txn, tid, rid)
			if err != nil {
				return err
			}
			if !rec.Active {
				for _, a := range existingRoles {
					muts = append(muts, spanner.Delete("SupplierUserRoles", spanner.Key{tid, rid, a.RoleName}))
				}
				rolesSynced = nil
			} else {
				desiredMap := make(map[string]struct{}, len(rolesSynced))
				for _, rn := range rolesSynced {
					desiredMap[rn] = struct{}{}
				}
				var next []UserRoleAssignment
				for _, a := range existingRoles {
					if a.AssignedBy == "scim:sync" {
						if _, keep := desiredMap[a.RoleName]; !keep {
							muts = append(muts, spanner.Delete("SupplierUserRoles", spanner.Key{tid, rid, a.RoleName}))
							continue
						}
					}
					next = append(next, a)
				}
				for _, rn := range rolesSynced {
					already := false
					for _, a := range next {
						if a.RoleName == rn {
							already = true
							break
						}
					}
					if !already {
						next = append(next, UserRoleAssignment{
							TenantID: tid, UserID: rid, RoleName: rn, AssignedBy: "scim:sync", CreatedAt: now, UpdatedAt: now,
						})
					}
					muts = append(muts, spanner.InsertOrUpdateMap("SupplierUserRoles", map[string]any{
						"TenantId":   tid,
						"UserId":     rid,
						"RoleName":   rn,
						"AssignedBy": "scim:sync",
						"CreatedAt":  spanner.CommitTimestamp,
						"UpdatedAt":  spanner.CommitTimestamp,
					}))
				}
				roleNames := filterActiveRoleNames(next, now)
				if err := checkStaticRoleSoDWithMeta(roleNames, rolePerms, policies, roSet); err != nil {
					return err
				}
			}

		case ScimResourceTypeGroup:
			groupRoles, err := resolveGroupSyncRoles(rec, rolePerms)
			if err != nil {
				return err
			}
			rolesSynced = groupRoles
			groupTag := "scim:group:" + rid
			newMembers := make(map[string]struct{}, len(rec.Members))
			for _, m := range rec.Members {
				newMembers[m.Value] = struct{}{}
			}
			if hadPrev {
				for _, pm := range prev.Members {
					if _, kept := newMembers[pm.Value]; !kept {
						prevMemberRoles, err := readUserRolesInTxn(txCtx, txn, tid, pm.Value)
						if err != nil {
							return err
						}
						for _, a := range prevMemberRoles {
							if a.AssignedBy == groupTag {
								muts = append(muts, spanner.Delete("SupplierUserRoles", spanner.Key{tid, pm.Value, a.RoleName}))
							}
						}
					}
				}
			}
			if len(groupRoles) > 0 {
				desiredGroupSet := make(map[string]struct{}, len(groupRoles))
				for _, rn := range groupRoles {
					desiredGroupSet[rn] = struct{}{}
				}
				for _, m := range rec.Members {
					existingRoles, err := readUserRolesInTxn(txCtx, txn, tid, m.Value)
					if err != nil {
						return err
					}
					var next []UserRoleAssignment
					for _, a := range existingRoles {
						if a.AssignedBy == groupTag {
							if _, keep := desiredGroupSet[a.RoleName]; !keep {
								muts = append(muts, spanner.Delete("SupplierUserRoles", spanner.Key{tid, m.Value, a.RoleName}))
								continue
							}
						}
						next = append(next, a)
					}
					for _, rn := range groupRoles {
						already := false
						for _, a := range next {
							if a.RoleName == rn {
								already = true
								break
							}
						}
						if !already {
							next = append(next, UserRoleAssignment{
								TenantID: tid, UserID: m.Value, RoleName: rn, AssignedBy: groupTag, CreatedAt: now, UpdatedAt: now,
							})
						}
						muts = append(muts, spanner.InsertOrUpdateMap("SupplierUserRoles", map[string]any{
							"TenantId":   tid,
							"UserId":     m.Value,
							"RoleName":   rn,
							"AssignedBy": groupTag,
							"CreatedAt":  spanner.CommitTimestamp,
							"UpdatedAt":  spanner.CommitTimestamp,
						}))
					}
					roleNames := filterActiveRoleNames(next, now)
					if err := checkStaticRoleSoDWithMeta(roleNames, rolePerms, policies, roSet); err != nil {
						return err
					}
				}
			}
		}

		rolesJSON, _ := json.Marshal(rec.Roles)
		membersJSON, _ := json.Marshal(rec.Members)
		metaJSON, _ := json.Marshal(rec.Meta)
		muts = append(muts, spanner.InsertOrUpdateMap("ScimIdentities", map[string]any{
			"TenantId":     tid,
			"ResourceType": rt,
			"ResourceId":   rid,
			"ExternalId":   nullIfEmptyIAM(rec.ExternalID),
			"UserName":     nullIfEmptyIAM(rec.UserName),
			"DisplayName":  nullIfEmptyIAM(rec.DisplayName),
			"Email":        nullIfEmptyIAM(rec.Email),
			"Active":       rec.Active,
			"RolesJson":    string(rolesJSON),
			"MembersJson":  string(membersJSON),
			"MetaJson":     string(metaJSON),
			"CreatedAt":    createdAt,
			"UpdatedAt":    spanner.CommitTimestamp,
		}))

		if err := txn.BufferWrite(muts); err != nil {
			return fmt.Errorf("write ScimIdentities + SupplierUserRoles: %w", err)
		}

		buf := &tenantOutboxBuffer{inner: outbox.NewSpannerTxnBuffer(txn), tenantID: tid}
		if err := outbox.EmitJSON(txCtx, buf, events.AggregateScimIdentity, tid+":"+rt+":"+rid, events.TopicLedger, events.ScimIdentitySyncedEvent{
			BaseEvent:    events.BaseEvent{Type: events.EventScimIdentitySynced, Timestamp: now.Format(time.RFC3339Nano)},
			TenantID:     tid,
			ResourceType: rt,
			ResourceID:   rid,
			ExternalID:   rec.ExternalID,
			UserName:     rec.UserName,
			DisplayName:  rec.DisplayName,
			Email:        rec.Email,
			Active:       rec.Active,
			Action:       action,
			RolesSynced:  rolesSynced,
			ActorID:      actorID,
			TraceID:      traceID,
			OccurredAt:   now.Format(time.RFC3339Nano),
		}); err != nil {
			return fmt.Errorf("emit iam.scim.synced: %w", err)
		}
		if err := buf.inner.Flush(txCtx); err != nil {
			return fmt.Errorf("flush iam.scim.synced outbox: %w", err)
		}
		if auditFn != nil {
			if err := auditFn(txCtx, txn); err != nil {
				return fmt.Errorf("audit iam.scim.synced: %w", err)
			}
		}

		saved = cloneScimRecord(rec)
		if saved.CreatedAt.IsZero() {
			saved.CreatedAt = now
		}
		saved.UpdatedAt = now
		return nil
	})
	if err != nil {
		return ScimResourceRecord{}, nil, err
	}
	return saved, rolesSynced, nil
}

func (r *SpannerRepository) DeleteScimResource(
	ctx context.Context,
	tenantID, resourceType, resourceID, actorID, traceID string,
	auditFn func(context.Context, *spanner.ReadWriteTransaction) error,
) (bool, []string, error) {
	if r == nil || r.client == nil {
		return false, nil, ErrIAMUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	rt := strings.TrimSpace(resourceType)
	rid := strings.TrimSpace(resourceID)
	if tid == "" || rt == "" || rid == "" {
		return false, nil, fmt.Errorf("%w: tenant_id, resource_type, and resource_id required", ErrInvalidScimRequest)
	}

	var deleted bool
	var rolesSynced []string
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		deleted = false
		rolesSynced = nil
		row, rerr := txn.ReadRow(txCtx, "ScimIdentities", spanner.Key{tid, rt, rid}, []string{
			"TenantId", "ResourceType", "ResourceId", "ExternalId", "UserName",
			"DisplayName", "Email", "Active", "RolesJson", "MembersJson", "MetaJson",
			"CreatedAt", "UpdatedAt",
		})
		if rerr != nil {
			if spanner.ErrCode(rerr) == codes.NotFound {
				return nil
			}
			return fmt.Errorf("read ScimIdentities for delete: %w", rerr)
		}
		cur, err := scanScimIdentityRow(row)
		if err != nil {
			return err
		}

		now := time.Now().UTC()
		muts := []*spanner.Mutation{
			spanner.Delete("ScimIdentities", spanner.Key{tid, rt, rid}),
		}

		switch rt {
		case ScimResourceTypeUser:
			existing, err := readUserRolesInTxn(txCtx, txn, tid, rid)
			if err != nil {
				return err
			}
			for _, a := range existing {
				rolesSynced = append(rolesSynced, a.RoleName)
				muts = append(muts, spanner.Delete("SupplierUserRoles", spanner.Key{tid, rid, a.RoleName}))
			}
		case ScimResourceTypeGroup:
			groupTag := "scim:group:" + rid
			for _, m := range cur.Members {
				existing, err := readUserRolesInTxn(txCtx, txn, tid, m.Value)
				if err != nil {
					return err
				}
				for _, a := range existing {
					if a.AssignedBy == groupTag {
						rolesSynced = append(rolesSynced, a.RoleName)
						muts = append(muts, spanner.Delete("SupplierUserRoles", spanner.Key{tid, m.Value, a.RoleName}))
					}
				}
			}
		}

		if err := txn.BufferWrite(muts); err != nil {
			return fmt.Errorf("delete ScimIdentities + roles: %w", err)
		}
		buf := &tenantOutboxBuffer{inner: outbox.NewSpannerTxnBuffer(txn), tenantID: tid}
		if err := outbox.EmitJSON(txCtx, buf, events.AggregateScimIdentity, tid+":"+rt+":"+rid, events.TopicLedger, events.ScimIdentitySyncedEvent{
			BaseEvent:    events.BaseEvent{Type: events.EventScimIdentitySynced, Timestamp: now.Format(time.RFC3339Nano)},
			TenantID:     tid,
			ResourceType: rt,
			ResourceID:   rid,
			ExternalID:   cur.ExternalID,
			UserName:     cur.UserName,
			DisplayName:  cur.DisplayName,
			Email:        cur.Email,
			Active:       false,
			Action:       "DEPROVISION",
			RolesSynced:  rolesSynced,
			ActorID:      actorID,
			TraceID:      traceID,
			OccurredAt:   now.Format(time.RFC3339Nano),
		}); err != nil {
			return fmt.Errorf("emit iam.scim.synced delete: %w", err)
		}
		if err := buf.inner.Flush(txCtx); err != nil {
			return fmt.Errorf("flush iam.scim.synced delete outbox: %w", err)
		}
		if auditFn != nil {
			if err := auditFn(txCtx, txn); err != nil {
				return fmt.Errorf("audit iam.scim.synced delete: %w", err)
			}
		}
		deleted = true
		return nil
	})
	if err != nil {
		return false, nil, err
	}
	return deleted, rolesSynced, nil
}

func scanScimIdentityRow(row *spanner.Row) (ScimResourceRecord, error) {
	var rec ScimResourceRecord
	var extID, userName, displayName, email spanner.NullString
	var rolesJSON, membersJSON, metaJSON spanner.NullString
	if err := row.Columns(
		&rec.TenantID, &rec.ResourceType, &rec.ResourceID, &extID, &userName,
		&displayName, &email, &rec.Active, &rolesJSON, &membersJSON, &metaJSON,
		&rec.CreatedAt, &rec.UpdatedAt,
	); err != nil {
		return ScimResourceRecord{}, fmt.Errorf("scan ScimIdentities row: %w", err)
	}
	rec.ExternalID = extID.StringVal
	rec.UserName = userName.StringVal
	rec.DisplayName = displayName.StringVal
	rec.Email = email.StringVal
	rec.CreatedAt = rec.CreatedAt.UTC()
	rec.UpdatedAt = rec.UpdatedAt.UTC()
	if strings.TrimSpace(rolesJSON.StringVal) != "" {
		_ = json.Unmarshal([]byte(rolesJSON.StringVal), &rec.Roles)
	}
	if strings.TrimSpace(membersJSON.StringVal) != "" {
		_ = json.Unmarshal([]byte(membersJSON.StringVal), &rec.Members)
	}
	if strings.TrimSpace(metaJSON.StringVal) != "" {
		_ = json.Unmarshal([]byte(metaJSON.StringVal), &rec.Meta)
	}
	if rec.Roles == nil {
		rec.Roles = []ScimRoleRef{}
	}
	if rec.Members == nil {
		rec.Members = []ScimMemberRef{}
	}
	return rec, nil
}

func nullIfEmptyIAM(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}
