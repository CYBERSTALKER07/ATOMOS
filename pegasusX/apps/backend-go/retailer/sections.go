package retailer

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"google.golang.org/api/iterator"
)

const (
	SectionActive   = "ACTIVE"
	SectionInactive = "INACTIVE"
)

// SectionDTO is a store department/section.
type SectionDTO struct {
	SectionID  string   `json:"section_id"`
	RetailerID string   `json:"retailer_id"`
	LocationID string   `json:"location_id"`
	Name       string   `json:"name"`
	AisleTag   string   `json:"aisle_tag,omitempty"`
	ShelfTag   string   `json:"shelf_tag,omitempty"`
	SortOrder  int64    `json:"sort_order"`
	Status     string   `json:"status"`
	SkuCount   int      `json:"sku_count,omitempty"`
	StaffIDs   []string `json:"staff_ids,omitempty"`
	CreatedAt  string   `json:"created_at,omitempty"`
	UpdatedAt  string   `json:"updated_at,omitempty"`
}

// HandleSections serves GET/POST /v1/retailer/sections
func (s *Service) HandleSections(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleSectionsGet(w, r)
	case http.MethodPost:
		s.handleSectionCreate(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleSectionsGet(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermStockView) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	locID := strings.TrimSpace(r.URL.Query().Get("location_id"))
	items, err := s.listSections(r.Context(), orgID, locID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_sections_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Service) handleSectionCreate(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermSectionManage) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermSectionManage})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	body, okBody := readLimitedBody(w, r, 16*1024)
	if !okBody {
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req struct {
		LocationID string `json:"location_id"`
		Name       string `json:"name"`
		AisleTag   string `json:"aisle_tag"`
		ShelfTag   string `json:"shelf_tag"`
		SortOrder  int64  `json:"sort_order"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name_required"})
		return
	}
	locID := strings.TrimSpace(req.LocationID)
	if locID == "" {
		locID = strings.TrimSpace(claims.ActiveLocationID)
	}
	if locID == "" {
		if p, e := s.EnsurePrimaryLocation(r.Context(), orgID); e == nil {
			locID = p.LocationID
		}
	}
	if err := s.assertLocationInOrg(r.Context(), orgID, locID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "location_not_found"})
		return
	}
	userID := auth.ResolveRetailerUserID(claims)
	// SECTIONS hard-deps STORE_STOCK — auto-enable both.
	enabled, _ := s.LoadEnabledPacks(r.Context(), orgID)
	if !enabled.Has(PackSTORESTOCK) {
		_ = s.SetPackEnabled(r.Context(), orgID, PackSTORESTOCK, userID, true, map[string]any{})
	}
	if !enabled.Has(PackSECTIONS) {
		_ = s.SetPackEnabled(r.Context(), orgID, PackSECTIONS, userID, true, map[string]any{})
	}
	sec := SectionDTO{
		SectionID:  s.newID(),
		RetailerID: orgID,
		LocationID: locID,
		Name:       name,
		AisleTag:   strings.TrimSpace(req.AisleTag),
		ShelfTag:   strings.TrimSpace(req.ShelfTag),
		SortOrder:  req.SortOrder,
		Status:     SectionActive,
		CreatedAt:  s.now().UTC().Format(time.RFC3339Nano),
	}
	if err := s.saveSection(r.Context(), sec); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "create_section_failed"})
		return
	}
	_ = s.emitPosEvent(r.Context(), orgID, events.EventRetailerSectionCreated, map[string]any{
		"section_id":  sec.SectionID,
		"location_id": locID,
		"name":        name,
	})
	respBytes, _ := json.Marshal(sec)
	idemCommitted = true
	s.saveIdempotency(r.Context(), r, body, http.StatusCreated, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(respBytes)
}

// HandleSectionByID serves GET/PATCH/DELETE /v1/retailer/sections/{sectionID}
func (s *Service) HandleSectionByID(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	sectionID := strings.TrimSpace(chi.URLParam(r, "sectionID"))
	sec, found, err := s.getSection(r.Context(), sectionID)
	if err != nil || !found || sec.RetailerID != orgID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "section_not_found"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		if !auth.HasRetailerPerm(claims, auth.PermStockView) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		skus, _ := s.listSectionSkus(r.Context(), sectionID)
		staff, _ := s.listSectionStaff(r.Context(), sectionID)
		sec.SkuCount = len(skus)
		sec.StaffIDs = staff
		writeJSON(w, http.StatusOK, map[string]any{"section": sec, "skus": skus, "staff_ids": staff})
	case http.MethodPatch, http.MethodPut:
		if !auth.HasRetailerPerm(claims, auth.PermSectionManage) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		var req struct {
			Name      *string `json:"name"`
			AisleTag  *string `json:"aisle_tag"`
			ShelfTag  *string `json:"shelf_tag"`
			SortOrder *int64  `json:"sort_order"`
			Status    *string `json:"status"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
			sec.Name = strings.TrimSpace(*req.Name)
		}
		if req.AisleTag != nil {
			sec.AisleTag = strings.TrimSpace(*req.AisleTag)
		}
		if req.ShelfTag != nil {
			sec.ShelfTag = strings.TrimSpace(*req.ShelfTag)
		}
		if req.SortOrder != nil {
			sec.SortOrder = *req.SortOrder
		}
		if req.Status != nil {
			st := strings.ToUpper(strings.TrimSpace(*req.Status))
			if st == SectionActive || st == SectionInactive {
				sec.Status = st
			}
		}
		sec.UpdatedAt = s.now().UTC().Format(time.RFC3339Nano)
		if err := s.saveSection(r.Context(), sec); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update_failed"})
			return
		}
		_ = s.emitPosEvent(r.Context(), orgID, events.EventRetailerSectionUpdated, map[string]any{"section_id": sectionID})
		writeJSON(w, http.StatusOK, sec)
	case http.MethodDelete:
		if !auth.HasRetailerPerm(claims, auth.PermSectionManage) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		sec.Status = SectionInactive
		sec.UpdatedAt = s.now().UTC().Format(time.RFC3339Nano)
		_ = s.saveSection(r.Context(), sec)
		writeJSON(w, http.StatusOK, sec)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// HandleSectionSkus serves GET/PUT /v1/retailer/sections/{sectionID}/skus
func (s *Service) HandleSectionSkus(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	sectionID := strings.TrimSpace(chi.URLParam(r, "sectionID"))
	sec, found, err := s.getSection(r.Context(), sectionID)
	if err != nil || !found || sec.RetailerID != orgID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "section_not_found"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		if !auth.HasRetailerPerm(claims, auth.PermStockView) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		skus, _ := s.listSectionSkus(r.Context(), sectionID)
		writeJSON(w, http.StatusOK, map[string]any{"section_id": sectionID, "skus": skus})
	case http.MethodPut:
		if !auth.HasRetailerPerm(claims, auth.PermSectionManage) && !auth.HasRetailerPerm(claims, auth.PermStockAdjust) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		var req struct {
			Skus   []string `json:"skus"`
			Add    []string `json:"add"`
			Remove []string `json:"remove"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Skus != nil {
			_ = s.replaceSectionSkus(r.Context(), sec, req.Skus)
		} else {
			if len(req.Add) > 0 {
				_ = s.addSectionSkus(r.Context(), sec, req.Add)
			}
			if len(req.Remove) > 0 {
				_ = s.removeSectionSkus(r.Context(), sectionID, req.Remove)
			}
		}
		skus, _ := s.listSectionSkus(r.Context(), sectionID)
		_ = s.emitPosEvent(r.Context(), orgID, events.EventRetailerSectionSkuMapped, map[string]any{
			"section_id": sectionID,
			"sku_count":  len(skus),
		})
		writeJSON(w, http.StatusOK, map[string]any{"section_id": sectionID, "skus": skus})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// HandleSectionStaff serves GET/PUT /v1/retailer/sections/{sectionID}/staff
func (s *Service) HandleSectionStaff(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	sectionID := strings.TrimSpace(chi.URLParam(r, "sectionID"))
	sec, found, err := s.getSection(r.Context(), sectionID)
	if err != nil || !found || sec.RetailerID != orgID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "section_not_found"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		if !auth.HasRetailerPerm(claims, auth.PermStockView) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		staff, _ := s.listSectionStaff(r.Context(), sectionID)
		writeJSON(w, http.StatusOK, map[string]any{"section_id": sectionID, "user_ids": staff})
	case http.MethodPut:
		if !auth.HasRetailerPerm(claims, auth.PermSectionManage) && !auth.HasRetailerPerm(claims, auth.PermStaffManage) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		var req struct {
			UserIDs []string `json:"user_ids"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = s.replaceSectionStaff(r.Context(), sec, req.UserIDs)
		staff, _ := s.listSectionStaff(r.Context(), sectionID)
		_ = s.emitPosEvent(r.Context(), orgID, events.EventRetailerStaffSectionAssigned, map[string]any{
			"section_id": sectionID,
			"user_ids":   staff,
		})
		writeJSON(w, http.StatusOK, map[string]any{"section_id": sectionID, "user_ids": staff})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// HandleUnassignedSkus serves GET /v1/retailer/sections/unassigned-skus
func (s *Service) HandleUnassignedSkus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermStockView) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	locID := strings.TrimSpace(r.URL.Query().Get("location_id"))
	if locID == "" {
		locID = strings.TrimSpace(claims.ActiveLocationID)
	}
	if locID == "" {
		if p, e := s.EnsurePrimaryLocation(r.Context(), orgID); e == nil {
			locID = p.LocationID
		}
	}
	balances, _ := s.listStockBalances(r.Context(), orgID, locID)
	assigned := s.allAssignedSkusAtLocation(r.Context(), orgID, locID)
	var unassigned []string
	seen := map[string]bool{}
	for _, b := range balances {
		if assigned[b.Sku] || seen[b.Sku] {
			continue
		}
		seen[b.Sku] = true
		unassigned = append(unassigned, b.Sku)
	}
	sort.Strings(unassigned)
	writeJSON(w, http.StatusOK, map[string]any{"location_id": locID, "skus": unassigned})
}

// HandleMySections serves GET /v1/retailer/me/sections
func (s *Service) HandleMySections(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	userID := auth.ResolveRetailerUserID(claims)
	items, err := s.listSectionsForUser(r.Context(), orgID, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// ---- persistence (memory + Spanner) ----

func (s *Service) saveSection(ctx context.Context, sec SectionDTO) error {
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.sections == nil {
			s.sections = map[string]SectionDTO{}
		}
		s.sections[sec.SectionID] = sec
		return nil
	}
	row := map[string]any{
		"SectionId":  sec.SectionID,
		"RetailerId": sec.RetailerID,
		"LocationId": sec.LocationID,
		"Name":       sec.Name,
		"SortOrder":  sec.SortOrder,
		"Status":     sec.Status,
		"CreatedAt":  spanner.CommitTimestamp,
	}
	if sec.AisleTag != "" {
		row["AisleTag"] = sec.AisleTag
	}
	if sec.ShelfTag != "" {
		row["ShelfTag"] = sec.ShelfTag
	}
	if sec.UpdatedAt != "" {
		row["UpdatedAt"] = spanner.CommitTimestamp
	}
	_, err := s.spannerClient.Apply(ctx, []*spanner.Mutation{spanner.InsertOrUpdateMap("RetailerSections", row)})
	return err
}

func (s *Service) getSection(ctx context.Context, sectionID string) (SectionDTO, bool, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		sec, ok := s.sections[sectionID]
		return sec, ok, nil
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerSections", spanner.Key{sectionID},
		[]string{"SectionId", "RetailerId", "LocationId", "Name", "AisleTag", "ShelfTag", "SortOrder", "Status", "CreatedAt", "UpdatedAt"})
	if err != nil {
		if isNotFound(err) {
			return SectionDTO{}, false, nil
		}
		return SectionDTO{}, false, err
	}
	return decodeSectionRow(row)
}

func decodeSectionRow(row *spanner.Row) (SectionDTO, bool, error) {
	var sec SectionDTO
	var aisle, shelf spanner.NullString
	var created time.Time
	var updated spanner.NullTime
	if err := row.Columns(&sec.SectionID, &sec.RetailerID, &sec.LocationID, &sec.Name, &aisle, &shelf, &sec.SortOrder, &sec.Status, &created, &updated); err != nil {
		return SectionDTO{}, false, err
	}
	if aisle.Valid {
		sec.AisleTag = aisle.StringVal
	}
	if shelf.Valid {
		sec.ShelfTag = shelf.StringVal
	}
	sec.CreatedAt = created.UTC().Format(time.RFC3339Nano)
	if updated.Valid {
		sec.UpdatedAt = updated.Time.UTC().Format(time.RFC3339Nano)
	}
	return sec, true, nil
}

func (s *Service) listSections(ctx context.Context, retailerID, locationID string) ([]SectionDTO, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		var out []SectionDTO
		for _, sec := range s.sections {
			if sec.RetailerID != retailerID {
				continue
			}
			if locationID != "" && sec.LocationID != locationID {
				continue
			}
			if sec.Status == SectionInactive {
				continue
			}
			cp := sec
			if set := s.sectionSkus[sec.SectionID]; set != nil {
				cp.SkuCount = len(set)
			}
			if staff := s.staffSections[sec.SectionID]; staff != nil {
				for uid := range staff {
					cp.StaffIDs = append(cp.StaffIDs, uid)
				}
				sort.Strings(cp.StaffIDs)
			}
			out = append(out, cp)
		}
		sort.Slice(out, func(i, j int) bool {
			if out[i].SortOrder != out[j].SortOrder {
				return out[i].SortOrder < out[j].SortOrder
			}
			return out[i].Name < out[j].Name
		})
		return out, nil
	}
	sql := `SELECT SectionId, RetailerId, LocationId, Name, AisleTag, ShelfTag, SortOrder, Status, CreatedAt, UpdatedAt
		FROM RetailerSections WHERE RetailerId = @rid AND Status = @st`
	params := map[string]any{"rid": retailerID, "st": SectionActive}
	if locationID != "" {
		sql += ` AND LocationId = @lid`
		params["lid"] = locationID
	}
	sql += ` ORDER BY SortOrder, Name`
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var out []SectionDTO
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		sec, _, err := decodeSectionRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, sec)
	}
	return out, nil
}

func (s *Service) listSectionSkus(ctx context.Context, sectionID string) ([]string, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		set := s.sectionSkus[sectionID]
		out := make([]string, 0, len(set))
		for sku := range set {
			out = append(out, sku)
		}
		sort.Strings(out)
		return out, nil
	}
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT Sku FROM RetailerSectionSkus WHERE SectionId = @sid ORDER BY Sku`,
		Params: map[string]any{"sid": sectionID},
	})
	defer iter.Stop()
	var out []string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var sku string
		if err := row.Columns(&sku); err != nil {
			return nil, err
		}
		out = append(out, sku)
	}
	return out, nil
}

func (s *Service) replaceSectionSkus(ctx context.Context, sec SectionDTO, skus []string) error {
	clean := uniqueNonEmpty(skus)
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.sectionSkus == nil {
			s.sectionSkus = map[string]map[string]bool{}
		}
		set := map[string]bool{}
		for _, sku := range clean {
			set[sku] = true
		}
		s.sectionSkus[sec.SectionID] = set
		return nil
	}
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		iter := txn.Query(ctx, spanner.Statement{
			SQL:    `SELECT Sku FROM RetailerSectionSkus WHERE SectionId = @sid`,
			Params: map[string]any{"sid": sec.SectionID},
		})
		defer iter.Stop()
		var muts []*spanner.Mutation
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}
			var sku string
			_ = row.Columns(&sku)
			muts = append(muts, spanner.Delete("RetailerSectionSkus", spanner.Key{sec.SectionID, sku}))
		}
		for _, sku := range clean {
			muts = append(muts, spanner.InsertOrUpdateMap("RetailerSectionSkus", map[string]any{
				"SectionId":  sec.SectionID,
				"Sku":        sku,
				"LocationId": sec.LocationID,
				"RetailerId": sec.RetailerID,
				"CreatedAt":  spanner.CommitTimestamp,
			}))
		}
		return txn.BufferWrite(muts)
	})
	return err
}

func (s *Service) addSectionSkus(ctx context.Context, sec SectionDTO, skus []string) error {
	clean := uniqueNonEmpty(skus)
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.sectionSkus == nil {
			s.sectionSkus = map[string]map[string]bool{}
		}
		if s.sectionSkus[sec.SectionID] == nil {
			s.sectionSkus[sec.SectionID] = map[string]bool{}
		}
		for _, sku := range clean {
			s.sectionSkus[sec.SectionID][sku] = true
		}
		return nil
	}
	var muts []*spanner.Mutation
	for _, sku := range clean {
		muts = append(muts, spanner.InsertOrUpdateMap("RetailerSectionSkus", map[string]any{
			"SectionId": sec.SectionID, "Sku": sku, "LocationId": sec.LocationID,
			"RetailerId": sec.RetailerID, "CreatedAt": spanner.CommitTimestamp,
		}))
	}
	_, err := s.spannerClient.Apply(ctx, muts)
	return err
}

func (s *Service) removeSectionSkus(ctx context.Context, sectionID string, skus []string) error {
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		set := s.sectionSkus[sectionID]
		for _, sku := range skus {
			delete(set, strings.TrimSpace(sku))
		}
		return nil
	}
	var muts []*spanner.Mutation
	for _, sku := range skus {
		sku = strings.TrimSpace(sku)
		if sku == "" {
			continue
		}
		muts = append(muts, spanner.Delete("RetailerSectionSkus", spanner.Key{sectionID, sku}))
	}
	_, err := s.spannerClient.Apply(ctx, muts)
	return err
}

func (s *Service) listSectionStaff(ctx context.Context, sectionID string) ([]string, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		set := s.staffSections[sectionID]
		out := make([]string, 0, len(set))
		for uid := range set {
			out = append(out, uid)
		}
		sort.Strings(out)
		return out, nil
	}
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT UserId FROM RetailerStaffSections WHERE SectionId = @sid ORDER BY UserId`,
		Params: map[string]any{"sid": sectionID},
	})
	defer iter.Stop()
	var out []string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var uid string
		_ = row.Columns(&uid)
		out = append(out, uid)
	}
	return out, nil
}

func (s *Service) replaceSectionStaff(ctx context.Context, sec SectionDTO, userIDs []string) error {
	clean := uniqueNonEmpty(userIDs)
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.staffSections == nil {
			s.staffSections = map[string]map[string]bool{}
		}
		set := map[string]bool{}
		for _, uid := range clean {
			set[uid] = true
		}
		s.staffSections[sec.SectionID] = set
		return nil
	}
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		iter := txn.Query(ctx, spanner.Statement{
			SQL:    `SELECT UserId FROM RetailerStaffSections WHERE SectionId = @sid`,
			Params: map[string]any{"sid": sec.SectionID},
		})
		defer iter.Stop()
		var muts []*spanner.Mutation
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}
			var uid string
			_ = row.Columns(&uid)
			muts = append(muts, spanner.Delete("RetailerStaffSections", spanner.Key{uid, sec.SectionID}))
		}
		for _, uid := range clean {
			muts = append(muts, spanner.InsertOrUpdateMap("RetailerStaffSections", map[string]any{
				"UserId": uid, "SectionId": sec.SectionID, "RetailerId": sec.RetailerID,
				"AssignedAt": spanner.CommitTimestamp,
			}))
		}
		return txn.BufferWrite(muts)
	})
	return err
}

func (s *Service) listSectionsForUser(ctx context.Context, retailerID, userID string) ([]SectionDTO, error) {
	all, err := s.listSections(ctx, retailerID, "")
	if err != nil {
		return nil, err
	}
	// Owners/managers without explicit assign see all; others filtered.
	// For simplicity: return sections where user is staff OR role is owner-like (caller filters).
	var out []SectionDTO
	for _, sec := range all {
		staff, _ := s.listSectionStaff(ctx, sec.SectionID)
		for _, uid := range staff {
			if uid == userID {
				out = append(out, sec)
				break
			}
		}
	}
	return out, nil
}

func (s *Service) allAssignedSkusAtLocation(ctx context.Context, retailerID, locationID string) map[string]bool {
	out := map[string]bool{}
	sections, _ := s.listSections(ctx, retailerID, locationID)
	for _, sec := range sections {
		skus, _ := s.listSectionSkus(ctx, sec.SectionID)
		for _, sku := range skus {
			out[sku] = true
		}
	}
	return out
}

func uniqueNonEmpty(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}
