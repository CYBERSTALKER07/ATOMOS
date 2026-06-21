package memory

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
)

// ── Scaffold in-memory supplier profile repository ─────────────────────────
// Production swaps this for a Spanner-backed implementation that runs every
// UpdateProfile inside a ReadWriteTransaction and writes the OutboxEvents row
// atomically via the supplied TxnBuffer.

type inMemorySupplierRepo struct {
	mu                  sync.RWMutex
	profiles            map[string]supplier.Profile
	authByPhone         map[string]supplier.SupplierAuthRecord
	topologyBySupp      map[string]supplier.SupplierTopology
	orgMembersBySupp    map[string][]supplier.SupplierOrgMember
	fleetDriversBySupp  map[string][]supplier.SupplierFleetDriver
	fleetVehiclesBySupp map[string][]supplier.SupplierFleetVehicle
	pricingBySupp       map[string]supplier.SupplierPricingRule
	aiRecommendations   map[string]supplier.AIRecommendation
	outboxAppender      OutboxAppender
}

func NewSupplierRepo(outboxAppender OutboxAppender) *inMemorySupplierRepo {
	return &inMemorySupplierRepo{
		profiles:            make(map[string]supplier.Profile),
		authByPhone:         make(map[string]supplier.SupplierAuthRecord),
		topologyBySupp:      make(map[string]supplier.SupplierTopology),
		orgMembersBySupp:    make(map[string][]supplier.SupplierOrgMember),
		fleetDriversBySupp:  make(map[string][]supplier.SupplierFleetDriver),
		fleetVehiclesBySupp: make(map[string][]supplier.SupplierFleetVehicle),
		pricingBySupp:       make(map[string]supplier.SupplierPricingRule),
		aiRecommendations:   make(map[string]supplier.AIRecommendation),
		outboxAppender:      outboxAppender,
	}
}

func (r *inMemorySupplierRepo) GetProfile(_ context.Context, supplierID string) (supplier.Profile, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.profiles[supplierID]
	return p, ok, nil
}

func (r *inMemorySupplierRepo) CountSuppliers(_ context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.profiles) == 0 {
		return 1, nil
	}
	return int64(len(r.profiles)), nil
}

func (r *inMemorySupplierRepo) UpdateProfile(ctx context.Context, p supplier.Profile, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.profiles[p.SupplierID] = p
	if _, ok := r.topologyBySupp[p.SupplierID]; !ok && (p.WarehouseLat != 0 || p.WarehouseLng != 0) {
		warehouseName := strings.TrimSpace(p.WarehouseName)
		if warehouseName == "" {
			warehouseName = "Primary Warehouse"
		}
		topology := supplier.SupplierTopology{
			Warehouses: []supplier.WarehouseNode{{
				WarehouseID:      "wh_primary_" + strings.TrimSpace(p.SupplierID),
				Name:             warehouseName,
				Lat:              p.WarehouseLat,
				Lng:              p.WarehouseLng,
				CoverageRadiusKm: 10,
				IsActive:         true,
				IsOnShift:        true,
				CreatedAt:        p.RegisteredAt,
				UpdatedAt:        p.UpdatedAt,
			}},
			Factories: make([]supplier.FactoryNode, 0, p.FactoryCount),
		}
		for i := 0; i < p.FactoryCount; i++ {
			topology.Factories = append(topology.Factories, supplier.FactoryNode{
				FactoryID: "fc_" + strings.TrimSpace(p.SupplierID) + "_" + strconv.Itoa(i+1),
				Name:      "Factory " + strconv.Itoa(i+1),
				Lat:       p.WarehouseLat,
				Lng:       p.WarehouseLng,
				IsActive:  true,
				CreatedAt: p.RegisteredAt,
				UpdatedAt: p.UpdatedAt,
			})
		}
		r.topologyBySupp[p.SupplierID] = topology
	}
	if strings.TrimSpace(p.AuthPasswordHash) != "" && strings.TrimSpace(p.Phone) != "" {
		userID := strings.TrimSpace(p.AuthUserID)
		if userID == "" {
			userID = "root_" + p.SupplierID
		}
		r.authByPhone[strings.TrimSpace(p.Phone)] = supplier.SupplierAuthRecord{
			UserID:       userID,
			SupplierID:   p.SupplierID,
			Phone:        strings.TrimSpace(p.Phone),
			PasswordHash: p.AuthPasswordHash,
			IsRegistered: p.IsRegistered,
			IsConfigured: p.IsConfigured,
		}
	}
	return nil
}

func (r *inMemorySupplierRepo) GetAuthByPhone(_ context.Context, phone string) (supplier.SupplierAuthRecord, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.authByPhone[strings.TrimSpace(phone)]
	return rec, ok, nil
}

func (r *inMemorySupplierRepo) GetTopology(_ context.Context, supplierID string) (supplier.SupplierTopology, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	topo, ok := r.topologyBySupp[supplierID]
	if !ok {
		return supplier.SupplierTopology{}, nil
	}
	return cloneSupplierTopology(topo), nil
}

func (r *inMemorySupplierRepo) ReplaceTopology(ctx context.Context, supplierID string, topology supplier.SupplierTopology, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.topologyBySupp[supplierID] = cloneSupplierTopology(topology)
	return nil
}

func (r *inMemorySupplierRepo) ListOrgMembers(_ context.Context, supplierID string) ([]supplier.SupplierOrgMember, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rows := r.orgMembersBySupp[supplierID]
	return append([]supplier.SupplierOrgMember(nil), rows...), nil
}

func (r *inMemorySupplierRepo) CreateOrgMember(ctx context.Context, member supplier.CreateOrgMemberParams, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.orgMembersBySupp[member.SupplierID] = append(r.orgMembersBySupp[member.SupplierID], supplier.SupplierOrgMember{
		UserID:              member.UserID,
		SupplierID:          member.SupplierID,
		Name:                member.Name,
		Email:               member.Email,
		Phone:               member.Phone,
		SupplierRole:        member.SupplierRole,
		AssignedWarehouseID: member.AssignedWarehouseID,
		AssignedFactoryID:   member.AssignedFactoryID,
		IsActive:            member.IsActive,
		CreatedAt:           member.CreatedAt,
		UpdatedAt:           member.UpdatedAt,
	})
	return nil
}

func (r *inMemorySupplierRepo) UpdateOrgMember(ctx context.Context, supplierID, userID string, patch supplier.UpdateOrgMemberPatch, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	rows := r.orgMembersBySupp[supplierID]
	found := false
	for i := range rows {
		if rows[i].UserID != userID {
			continue
		}
		found = true
		if patch.Name != nil {
			rows[i].Name = *patch.Name
		}
		if patch.SupplierRole != nil {
			rows[i].SupplierRole = *patch.SupplierRole
		}
		if patch.AssignedWarehouseID != nil {
			rows[i].AssignedWarehouseID = *patch.AssignedWarehouseID
		}
		if patch.AssignedFactoryID != nil {
			rows[i].AssignedFactoryID = *patch.AssignedFactoryID
		}
		if patch.IsActive != nil {
			rows[i].IsActive = *patch.IsActive
		}
		rows[i].UpdatedAt = time.Now().UTC()
		r.orgMembersBySupp[supplierID][i] = rows[i]
		break
	}
	if !found {
		return fmt.Errorf("supplier_org_member_not_found")
	}
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *inMemorySupplierRepo) ListFleetDrivers(_ context.Context, supplierID string) ([]supplier.SupplierFleetDriver, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rows := r.fleetDriversBySupp[supplierID]
	return append([]supplier.SupplierFleetDriver(nil), rows...), nil
}

func (r *inMemorySupplierRepo) CreateFleetDriver(ctx context.Context, driverParams supplier.CreateFleetDriverParams, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.fleetDriversBySupp[driverParams.SupplierID] = append(r.fleetDriversBySupp[driverParams.SupplierID], supplier.SupplierFleetDriver{
		DriverID:     driverParams.DriverID,
		SupplierID:   driverParams.SupplierID,
		Name:         driverParams.Name,
		Phone:        driverParams.Phone,
		HomeNodeType: driverParams.HomeNodeType,
		HomeNodeID:   driverParams.HomeNodeID,
		VehicleID:    driverParams.VehicleID,
		IsActive:     driverParams.IsActive,
		CreatedAt:    driverParams.CreatedAt,
		UpdatedAt:    driverParams.UpdatedAt,
	})
	return nil
}

func (r *inMemorySupplierRepo) ListFleetVehicles(_ context.Context, supplierID string) ([]supplier.SupplierFleetVehicle, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rows := r.fleetVehiclesBySupp[supplierID]
	return append([]supplier.SupplierFleetVehicle(nil), rows...), nil
}

func (r *inMemorySupplierRepo) CreateFleetVehicle(ctx context.Context, vehicle supplier.CreateFleetVehicleParams, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.fleetVehiclesBySupp[vehicle.SupplierID] = append(r.fleetVehiclesBySupp[vehicle.SupplierID], supplier.SupplierFleetVehicle{
		VehicleID:    vehicle.VehicleID,
		SupplierID:   vehicle.SupplierID,
		Label:        vehicle.Label,
		LicensePlate: vehicle.LicensePlate,
		HomeNodeType: vehicle.HomeNodeType,
		HomeNodeID:   vehicle.HomeNodeID,
		VehicleClass: vehicle.VehicleClass,
		MaxVolumeVU:  vehicle.MaxVolumeVU,
		IsActive:     vehicle.IsActive,
		CreatedAt:    vehicle.CreatedAt,
		UpdatedAt:    vehicle.UpdatedAt,
	})
	return nil
}

func (r *inMemorySupplierRepo) GetPricingRule(_ context.Context, supplierID string) (supplier.SupplierPricingRule, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rule, ok := r.pricingBySupp[supplierID]
	return rule, ok, nil
}

func (r *inMemorySupplierRepo) UpsertPricingRule(ctx context.Context, rule supplier.SupplierPricingRule, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}

	existing, exists := r.pricingBySupp[rule.SupplierID]
	if exists {
		rule.RuleVersion = existing.RuleVersion + 1
	} else if rule.RuleVersion <= 0 {
		rule.RuleVersion = 1
	}
	if strings.TrimSpace(rule.Currency) == "" {
		if profile, ok := r.profiles[rule.SupplierID]; ok && strings.TrimSpace(profile.Currency) != "" {
			rule.Currency = profile.Currency
		}
	}
	if rule.UpdatedAt.IsZero() {
		rule.UpdatedAt = time.Now().UTC()
	}
	r.pricingBySupp[rule.SupplierID] = rule
	return nil
}

func (r *inMemorySupplierRepo) ListAIRecommendations(_ context.Context, supplierID string, query supplier.AIRecommendationQuery) ([]supplier.AIRecommendation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	limit := query.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	status := strings.ToUpper(strings.TrimSpace(query.Status))
	items := make([]supplier.AIRecommendation, 0, limit)
	for _, item := range r.aiRecommendations {
		if item.SupplierID != supplierID {
			continue
		}
		if status != "" && !strings.EqualFold(item.Status, status) {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].GeneratedAt > items[j].GeneratedAt })
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (r *inMemorySupplierRepo) RecordAIRecommendationDecision(ctx context.Context, supplierID string, decision supplier.AIRecommendationDecision, emit func(outbox.TxnBuffer, supplier.AIRecommendation) error) (supplier.AIRecommendation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.aiRecommendations[strings.TrimSpace(decision.RecommendationID)]
	if !ok || item.SupplierID != supplierID {
		return supplier.AIRecommendation{}, supplier.ErrAIRecommendationNotFound
	}
	item.Status = supplierDecisionStatus(decision.Decision)
	item.Decision = decision.Decision
	item.DecisionNote = strings.TrimSpace(decision.Note)
	item.DecidedBy = strings.TrimSpace(decision.DecidedBy)
	item.DecidedAt = decision.DecidedAt.UTC().Format(time.RFC3339Nano)
	item.UpdatedAt = item.DecidedAt
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn, item); err != nil {
			return supplier.AIRecommendation{}, err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return supplier.AIRecommendation{}, err
			}
		}
	}
	r.aiRecommendations[item.RecommendationID] = item
	return item, nil
}

func supplierDecisionStatus(decision string) string {
	if strings.EqualFold(decision, "REOPENED") {
		return "PENDING"
	}
	return strings.ToUpper(strings.TrimSpace(decision))
}

func cloneSupplierTopology(src supplier.SupplierTopology) supplier.SupplierTopology {
	out := supplier.SupplierTopology{
		Warehouses: make([]supplier.WarehouseNode, 0, len(src.Warehouses)),
		Factories:  make([]supplier.FactoryNode, 0, len(src.Factories)),
	}
	out.Warehouses = append(out.Warehouses, src.Warehouses...)
	out.Factories = append(out.Factories, src.Factories...)
	return out
}
