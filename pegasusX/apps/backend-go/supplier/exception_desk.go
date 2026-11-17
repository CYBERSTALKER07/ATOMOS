package supplier

import (
	"context"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/entityresolution"
	"google.golang.org/api/iterator"
)

// Explicit 6-handler desk taxonomy constants (#158 / #166).
const (
	HandlerFiscalAndExportBlockers = "FISCAL_AND_EXPORT_BLOCKERS"
	HandlerCashDiscrepancy         = "CASH_DISCREPANCY"
	HandlerClaimsReview            = "CLAIMS_REVIEW"
	HandlerRouteAndClosedDoor      = "ROUTE_AND_CLOSED_DOOR"
	HandlerCreditAndPromise        = "CREDIT_AND_PROMISE"
	HandlerFulfillmentAndCatalog   = "FULFILLMENT_AND_CATALOG"
)

const (
	SeverityTierCritical = "CRITICAL"
	SeverityTierHigh     = "HIGH"
	SeverityTierMedium   = "MEDIUM"
	SeverityTierLow      = "LOW"
)

const highExposureThresholdMinor int64 = 50000000 // 500,000 UZS

type exceptionEntityMeta struct {
	DisplayName string
	Phone       string
	TIN         string
}

// SetExtraExceptionRowsQuery registers an additive test hook for injecting Spanner-shaped exceptions.
func (s *Service) SetExtraExceptionRowsQuery(fn func(context.Context, string) ([]SupplierExceptionRow, error)) {
	if s == nil {
		return
	}
	s.extraExceptionRowsFn = fn
}

// SetEntityMetaLookup registers a test hook for retailer/driver display metadata lookup.
func (s *Service) SetEntityMetaLookup(fn func(context.Context, string, []string, []string) (map[string]exceptionEntityMeta, map[string]exceptionEntityMeta, error)) {
	if s == nil {
		return
	}
	s.entityMetaLookupFn = fn
}

// ClassifyExceptionHandler maps an exception kind into one of the 6 explicit Supplier Desk handlers.
func ClassifyExceptionHandler(kind string) string {
	switch strings.ToUpper(strings.TrimSpace(kind)) {
	case "FISCAL_FAILED", "FISCAL_DEAD_LETTER", "EXPORT_REJECTED", "FISCAL_TIMEOUT", "FISCAL_FAILURE":
		return HandlerFiscalAndExportBlockers
	case "CASH_DISCREPANCY":
		return HandlerCashDiscrepancy
	case "CREDIT_NOTE_DRAFT", "CLAIM_REVIEW", "BUYER_REJECTED":
		return HandlerClaimsReview
	case "SHOP_CLOSED", "MANIFEST_EXCEPTION":
		return HandlerRouteAndClosedDoor
	case "CREDIT_FREEZE", "PROMISE_BREACH":
		return HandlerCreditAndPromise
	case "ORDER_CANCELLED", "VET_REJECTED", "AWAITING_REVIEW", "OPERATOR_NOTE", "TIN_DEDUP_CANDIDATE":
		return HandlerFulfillmentAndCatalog
	default:
		return HandlerFulfillmentAndCatalog
	}
}

// ClassifyExceptionSeverity assigns CRITICAL|HIGH|MEDIUM|LOW deterministically from kind and exposure.
func ClassifyExceptionSeverity(row SupplierExceptionRow) string {
	explicit := strings.ToUpper(strings.TrimSpace(row.SeverityTier))
	if explicit == "" {
		explicit = strings.ToUpper(strings.TrimSpace(row.Severity))
	}
	if severityTierRank(explicit) > 0 {
		return explicit
	}

	kind := strings.ToUpper(strings.TrimSpace(row.Kind))
	status := strings.ToUpper(strings.TrimSpace(row.Status))
	switch kind {
	case "FISCAL_FAILED", "FISCAL_DEAD_LETTER", "FISCAL_TIMEOUT", "FISCAL_FAILURE":
		return SeverityTierCritical
	case "EXPORT_REJECTED":
		if row.MoneyAtRiskMinor > 0 {
			return SeverityTierCritical
		}
		return SeverityTierHigh
	case "CASH_DISCREPANCY":
		if status == "DISPUTED" || row.MoneyAtRiskMinor >= highExposureThresholdMinor {
			return SeverityTierCritical
		}
		return SeverityTierHigh
	case "CREDIT_FREEZE", "PROMISE_BREACH":
		if status == "BLACKLISTED" || status == "BREACHED" || row.MoneyAtRiskMinor >= highExposureThresholdMinor {
			return SeverityTierCritical
		}
		return SeverityTierHigh
	case "CREDIT_NOTE_DRAFT", "CLAIM_REVIEW", "BUYER_REJECTED":
		if row.MoneyAtRiskMinor >= highExposureThresholdMinor {
			return SeverityTierCritical
		}
		return SeverityTierHigh
	case "SHOP_CLOSED", "MANIFEST_EXCEPTION":
		return SeverityTierHigh
	case "TIN_DEDUP_CANDIDATE", "VET_REJECTED", "ORDER_CANCELLED":
		return SeverityTierMedium
	case "AWAITING_REVIEW", "OPERATOR_NOTE":
		return SeverityTierLow
	default:
		return SeverityTierLow
	}
}

func severityTierRank(tier string) int {
	switch strings.ToUpper(strings.TrimSpace(tier)) {
	case SeverityTierCritical:
		return 4
	case SeverityTierHigh:
		return 3
	case SeverityTierMedium:
		return 2
	case SeverityTierLow:
		return 1
	default:
		return 0
	}
}

// SortSupplierExceptions sorts deterministically by severity_tier DESC, money_at_risk_minor DESC, updated_at DESC.
func SortSupplierExceptions(rows []SupplierExceptionRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		ri := severityTierRank(rows[i].SeverityTier)
		rj := severityTierRank(rows[j].SeverityTier)
		if ri != rj {
			return ri > rj
		}
		if rows[i].MoneyAtRiskMinor != rows[j].MoneyAtRiskMinor {
			return rows[i].MoneyAtRiskMinor > rows[j].MoneyAtRiskMinor
		}
		if rows[i].UpdatedAt != rows[j].UpdatedAt {
			return rows[i].UpdatedAt > rows[j].UpdatedAt
		}
		if rows[i].Kind != rows[j].Kind {
			return rows[i].Kind < rows[j].Kind
		}
		return rows[i].OrderID < rows[j].OrderID
	})
}

func (s *Service) enrichSupplierExceptionRows(ctx context.Context, supplierID string, rows []SupplierExceptionRow) {
	if len(rows) == 0 {
		return
	}
	retailerSet := make(map[string]struct{})
	driverSet := make(map[string]struct{})
	for i := range rows {
		rows[i].HandlerGroup = ClassifyExceptionHandler(rows[i].Kind)
		rows[i].SeverityTier = ClassifyExceptionSeverity(rows[i])
		rows[i].Severity = strings.ToLower(rows[i].SeverityTier)
		if rid := strings.TrimSpace(rows[i].RetailerID); rid != "" {
			retailerSet[rid] = struct{}{}
		}
		if did := strings.TrimSpace(rows[i].DriverID); did != "" {
			driverSet[did] = struct{}{}
		} else if strings.HasPrefix(rows[i].Note, "driver:") {
			if did = strings.TrimSpace(strings.TrimPrefix(rows[i].Note, "driver:")); did != "" {
				rows[i].DriverID = did
				driverSet[did] = struct{}{}
			}
		}
	}

	retailerIDs := make([]string, 0, len(retailerSet))
	for rid := range retailerSet {
		retailerIDs = append(retailerIDs, rid)
	}
	driverIDs := make([]string, 0, len(driverSet))
	for did := range driverSet {
		driverIDs = append(driverIDs, did)
	}

	retMeta, drvMeta := s.loadExceptionEntityMetadata(ctx, supplierID, retailerIDs, driverIDs)
	for i := range rows {
		if rid := strings.TrimSpace(rows[i].RetailerID); rid != "" {
			if m, ok := retMeta[rid]; ok {
				if rows[i].DisplayName == "" && m.DisplayName != "" {
					rows[i].DisplayName = m.DisplayName
				}
				if rows[i].Phone == "" && m.Phone != "" {
					rows[i].Phone = m.Phone
				}
				if rows[i].TIN == "" && m.TIN != "" {
					rows[i].TIN = m.TIN
				}
			}
		}
		if rows[i].DisplayName == "" {
			if did := strings.TrimSpace(rows[i].DriverID); did != "" {
				if dm, ok := drvMeta[did]; ok {
					rows[i].DisplayName = dm.DisplayName
					if rows[i].Phone == "" {
						rows[i].Phone = dm.Phone
					}
				}
			}
		}
	}
}

func (s *Service) loadExceptionEntityMetadata(ctx context.Context, supplierID string, retailerIDs, driverIDs []string) (map[string]exceptionEntityMeta, map[string]exceptionEntityMeta) {
	retMeta := make(map[string]exceptionEntityMeta)
	drvMeta := make(map[string]exceptionEntityMeta)

	if s.entityMetaLookupFn != nil {
		rm, dm, err := s.entityMetaLookupFn(ctx, supplierID, retailerIDs, driverIDs)
		if err == nil {
			for k, v := range rm {
				retMeta[k] = v
			}
			for k, v := range dm {
				drvMeta[k] = v
			}
		}
		return retMeta, drvMeta
	}

	// Merge SupplierRetailerLinks (both Spanner & in-memory) so legal_name, TIN, phone are available offline too.
	if links := s.RetailerLinks(); links != nil && supplierID != "" {
		if list, err := links.List(ctx, supplierID, "", 500); err == nil {
			for _, link := range list {
				if link.RetailerID == "" {
					continue
				}
				retMeta[link.RetailerID] = exceptionEntityMeta{
					DisplayName: strings.TrimSpace(link.LegalName),
					Phone:       strings.TrimSpace(link.NormalizedPhone),
					TIN:         strings.TrimSpace(link.TaxIDTIN),
				}
			}
		}
	}

	if s.portalSpanner == nil {
		// Also inspect in-memory repo fleet drivers if available.
		if s.repo != nil && supplierID != "" && len(driverIDs) > 0 {
			if drivers, err := s.repo.ListFleetDrivers(ctx, supplierID); err == nil {
				for _, d := range drivers {
					drvMeta[d.DriverID] = exceptionEntityMeta{
						DisplayName: strings.TrimSpace(d.Name),
						Phone:       strings.TrimSpace(d.Phone),
					}
				}
			}
		}
		return retMeta, drvMeta
	}

	if len(retailerIDs) > 0 {
		if supplierID != "" {
			linkIter := s.portalSpanner.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, spanner.Statement{
				SQL: `SELECT RetailerId, COALESCE(LegalName, ''), COALESCE(NormalizedPhone, ''), COALESCE(TaxIdTIN, '')
					FROM SupplierRetailerLinks
					WHERE SupplierId = @sid AND RetailerId IN UNNEST(@rids)`,
				Params: map[string]any{"sid": supplierID, "rids": retailerIDs},
			})
			for {
				row, err := linkIter.Next()
				if err == iterator.Done || err != nil {
					break
				}
				var rid, name, phone, tin string
				if err := row.Columns(&rid, &name, &phone, &tin); err == nil {
					m := retMeta[rid]
					m.DisplayName = firstNonEmptyExceptionStr(name, m.DisplayName)
					m.Phone = firstNonEmptyExceptionStr(phone, m.Phone)
					m.TIN = firstNonEmptyExceptionStr(tin, m.TIN)
					retMeta[rid] = m
				}
			}
			linkIter.Stop()
		}

		retIter := s.portalSpanner.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, spanner.Statement{
			SQL: `SELECT RetailerId, COALESCE(Name, ''), COALESCE(Phone, '')
				FROM Retailers
				WHERE RetailerId IN UNNEST(@rids)`,
			Params: map[string]any{"rids": retailerIDs},
		})
		for {
			row, err := retIter.Next()
			if err == iterator.Done || err != nil {
				break
			}
			var rid, name, phone string
			if err := row.Columns(&rid, &name, &phone); err == nil {
				m := retMeta[rid]
				m.DisplayName = firstNonEmptyExceptionStr(m.DisplayName, name)
				m.Phone = firstNonEmptyExceptionStr(m.Phone, phone)
				retMeta[rid] = m
			}
		}
		retIter.Stop()
	}

	if len(driverIDs) > 0 {
		drvIter := s.portalSpanner.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, spanner.Statement{
			SQL: `SELECT DriverId, COALESCE(Name, ''), COALESCE(Phone, '')
				FROM Drivers
				WHERE DriverId IN UNNEST(@dids)`,
			Params: map[string]any{"dids": driverIDs},
		})
		for {
			row, err := drvIter.Next()
			if err == iterator.Done || err != nil {
				break
			}
			var did, name, phone string
			if err := row.Columns(&did, &name, &phone); err == nil {
				drvMeta[did] = exceptionEntityMeta{
					DisplayName: strings.TrimSpace(name),
					Phone:       strings.TrimSpace(phone),
				}
			}
		}
		drvIter.Stop()
	}

	return retMeta, drvMeta
}

func (s *Service) markInMemoryOrderExceptionResolved(kind, id string) {
	if s == nil {
		return
	}
	s.retailerLinksMu.Lock()
	if s.resolvedOrderNotes == nil {
		s.resolvedOrderNotes = make(map[string]string)
	}
	s.resolvedOrderNotes[strings.ToUpper(strings.TrimSpace(kind))+"|"+strings.TrimSpace(id)] = "resolved"
	s.retailerLinksMu.Unlock()
}

func (s *Service) filterResolvedInMemoryExceptions(rows []SupplierExceptionRow) []SupplierExceptionRow {
	if s == nil || len(rows) == 0 {
		return rows
	}
	s.retailerLinksMu.Lock()
	defer s.retailerLinksMu.Unlock()
	if len(s.resolvedOrderNotes) == 0 {
		return rows
	}
	out := make([]SupplierExceptionRow, 0, len(rows))
	for _, row := range rows {
		key := strings.ToUpper(strings.TrimSpace(row.Kind)) + "|" + strings.TrimSpace(row.OrderID)
		if s.resolvedOrderNotes[key] != "" {
			continue
		}
		out = append(out, row)
	}
	return out
}

func firstNonEmptyExceptionStr(values ...string) string {
	for _, v := range values {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return ""
}

// Keep entityresolution imported cleanly.
var _ = entityresolution.DedupStatusDuplicateCandidate
