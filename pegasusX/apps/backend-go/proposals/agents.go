package proposals

import (
	"context"
	"fmt"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning"
	"github.com/pegasusx/pegasusx/apps/backend-go/roi"
)

// RunAgentPass runs one L1 agent ("REORDER_AGENT", "COLLECTIONS_AGENT", "EXPIRY_REBALANCE_AGENT")
// or "ALL" to scan live domain signals and emit governed PENDING proposals into approvals.Engine.
func (s *Service) RunAgentPass(ctx context.Context, tenantID, agentID string) ([]Proposal, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	if tid == "" {
		return nil, fmt.Errorf("%w: tenant_id is required", ErrInvalidInput)
	}
	norm := strings.ToUpper(strings.TrimSpace(agentID))
	switch norm {
	case AgentReorder:
		return s.RunReorderAgent(ctx, tid)
	case AgentCollections:
		return s.RunCollectionsAgent(ctx, tid)
	case AgentExpiryRebalance:
		return s.RunExpiryRebalanceAgent(ctx, tid)
	case "", "ALL":
		var all []Proposal
		r1, err := s.RunReorderAgent(ctx, tid)
		if err != nil {
			return nil, err
		}
		all = append(all, r1...)
		r2, err := s.RunCollectionsAgent(ctx, tid)
		if err != nil {
			return nil, err
		}
		all = append(all, r2...)
		r3, err := s.RunExpiryRebalanceAgent(ctx, tid)
		if err != nil {
			return nil, err
		}
		all = append(all, r3...)
		return all, nil
	default:
		return nil, fmt.Errorf("%w: unsupported agent_id %q", ErrInvalidInput, agentID)
	}
}

// RunReorderAgent inspects low-cover SKUs, pending replenishment insights, and winning forecasts
// to propose governed reorder / supply request proposals (never self-approving).
func (s *Service) RunReorderAgent(ctx context.Context, tenantID string) ([]Proposal, error) {
	if s == nil || s.repo == nil || s.dataReader == nil {
		return nil, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	cands, err := s.dataReader.ListReorderCandidates(ctx, tid, 50)
	if err != nil {
		return nil, err
	}
	today := s.currentNow().Format("2006-01-02")
	var created []Proposal
	for _, c := range cands {
		targetType := "Product"
		targetID := c.ProductID
		governedAction := string(planning.AgentOpenSupplyRequest)
		if c.InsightID != "" {
			targetType = "ReplenishmentInsight"
			targetID = c.InsightID
			if c.TargetFactoryID == "" {
				governedAction = string(planning.AgentApproveInsight)
			}
		} else if c.SuggestionKey != "" {
			targetType = "ReorderSuggestion"
			targetID = c.SuggestionKey
			governedAction = "approve_reorder_suggestion"
		}
		if strings.TrimSpace(targetID) == "" {
			continue
		}

		active, err := s.repo.HasActiveForTarget(ctx, tid, targetType, targetID, TypeReorder)
		if err != nil {
			return created, err
		}
		if active {
			continue
		}

		unitPrice := c.UnitPriceMinor
		if unitPrice <= 0 {
			unitPrice = 10000
		}
		// Estimated protected gross margin (approx 20% = 2000 bps of order value) in int64 minor units.
		impactMinor := (c.SuggestedQty * unitPrice * 2000) / 10000
		if impactMinor <= 0 {
			impactMinor = c.SuggestedQty * unitPrice
		}
		conf := c.ConfidenceBps
		if conf <= 0 {
			conf = 8200
		}
		cur := nonEmptyOr(c.Currency, "UZS")

		refs := []EvidenceRef{{Type: targetType, ID: targetID, Label: fmt.Sprintf("Stock %d -> suggest +%d", c.CurrentStock, c.SuggestedQty)}}
		if c.WarehouseID != "" {
			refs = append(refs, EvidenceRef{Type: "Warehouse", ID: c.WarehouseID, Label: "Source demand node"})
		}
		if c.ProductID != "" {
			refs = append(refs, EvidenceRef{Type: "Product", ID: c.ProductID, Label: nonEmptyOr(c.ProductName, c.ProductID)})
		}
		if c.ForecastRunID != "" {
			refs = append(refs, EvidenceRef{Type: "ForecastRun", ID: c.ForecastRunID, Label: fmt.Sprintf("P50 %d milli-units/day", c.ForecastP50Milli)})
		}

		payload := map[string]any{
			"action":             governedAction,
			"insight_id":         c.InsightID,
			"suggestion_key":     c.SuggestionKey,
			"warehouse_id":       c.WarehouseID,
			"product_id":         c.ProductID,
			"retailer_id":        c.RetailerID,
			"target_factory_id":  c.TargetFactoryID,
			"current_stock":      c.CurrentStock,
			"reorder_threshold":  c.ReorderThreshold,
			"suggested_qty":      c.SuggestedQty,
			"unit_price_minor":   unitPrice,
			"forecast_p50_milli": c.ForecastP50Milli,
			"currency":           cur,
		}

		skuLabel := nonEmptyOr(c.ProductName, c.ProductID)
		p, err := s.CreateProposal(ctx, CreateProposalInput{
			TenantID:             tid,
			AgentID:              AgentReorder,
			ProposalType:         TypeReorder,
			TargetObjectType:     targetType,
			TargetObjectID:       targetID,
			Title:                fmt.Sprintf("Reorder %s (+%d units)", skuLabel, c.SuggestedQty),
			Summary:              fmt.Sprintf("Current stock %d (threshold %d); replenish %d units", c.CurrentStock, c.ReorderThreshold, c.SuggestedQty),
			Rationale:            fmt.Sprintf("Detected %s urgency (%s) with %d units on hand; forecast P50=%d milli/day", nonEmptyOr(c.UrgencyLevel, "HIGH"), nonEmptyOr(c.ReasonCode, "LOW_COVER"), c.CurrentStock, c.ForecastP50Milli),
			Payload:              payload,
			EvidenceRefs:         refs,
			ConfidenceBps:        conf,
			EstimatedImpactMinor: impactMinor,
			Currency:             cur,
			RoiCategory:          roi.CategoryStockoutAvoided,
			CreatedBy:            "agent:" + strings.ToLower(AgentReorder),
			ActorType:            auth.ActorTypeAgent,
			IdempotencyKey:       fmt.Sprintf("l1:%s:%s:%s:%s", AgentReorder, tid, targetID, today),
		})
		if err != nil {
			return created, err
		}
		created = append(created, p)
	}
	return created, nil
}

// RunCollectionsAgent scans overdue ArInvoices and generates governed collections proposals
// with bilingual UZ and RU reminder drafts citing real invoice & order IDs.
func (s *Service) RunCollectionsAgent(ctx context.Context, tenantID string) ([]Proposal, error) {
	if s == nil || s.repo == nil || s.dataReader == nil {
		return nil, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	now := s.currentNow()
	invs, err := s.dataReader.ListOverdueInvoices(ctx, tid, now, 50)
	if err != nil {
		return nil, err
	}
	today := now.Format("2006-01-02")
	var created []Proposal

	for _, inv := range invs {
		active, err := s.repo.HasActiveForTarget(ctx, tid, "ArInvoice", inv.InvoiceID, TypeCollections)
		if err != nil {
			return created, err
		}
		if active {
			continue
		}
		cur := nonEmptyOr(inv.Currency, "UZS")
		arInv := ar.Invoice{
			InvoiceID:    inv.InvoiceID,
			SupplierID:   tid,
			RetailerID:   inv.RetailerID,
			OrderID:      inv.OrderID,
			BalanceMinor: inv.BalanceMinor,
			Currency:     cur,
			DueAt:        inv.DueAt,
			DunningStep:  inv.DunningStep,
		}
		step := inv.DunningStep + 1
		if step < ar.StepOverdue {
			step = ar.StepOverdue
		}
		_, titleUZ, bodyUZ := ar.LocalizedNotifyMessage(arInv, step, "uz")
		_, titleRU, bodyRU := ar.LocalizedNotifyMessage(arInv, step, "ru")

		actionMode := "collections_nudge"
		if inv.DaysOverdue >= 30 {
			actionMode = "collections_escalate"
		}

		refs := []EvidenceRef{
			{Type: "ArInvoice", ID: inv.InvoiceID, Label: fmt.Sprintf("Overdue %dd · %s", inv.DaysOverdue, ar.FormatWholeSom(inv.BalanceMinor, cur))},
			{Type: "Retailer", ID: inv.RetailerID, Label: nonEmptyOr(inv.RetailerName, inv.RetailerID)},
		}
		if inv.OrderID != "" {
			refs = append(refs, EvidenceRef{Type: "Order", ID: inv.OrderID, Label: "Originating credit order"})
		}

		payload := map[string]any{
			"action":         actionMode,
			"invoice_id":     inv.InvoiceID,
			"order_id":       inv.OrderID,
			"retailer_id":    inv.RetailerID,
			"retailer_name":  inv.RetailerName,
			"phone":          inv.Phone,
			"balance_minor":  inv.BalanceMinor,
			"currency":       cur,
			"days_overdue":   inv.DaysOverdue,
			"dunning_step":   step,
			"preferred_lang": nonEmptyOr(inv.Locale, "uz"),
			"draft_uz":       strings.TrimSpace(titleUZ + " — " + bodyUZ),
			"draft_ru":       strings.TrimSpace(titleRU + " — " + bodyRU),
		}

		retLabel := nonEmptyOr(inv.RetailerName, inv.RetailerID)
		p, err := s.CreateProposal(ctx, CreateProposalInput{
			TenantID:             tid,
			AgentID:              AgentCollections,
			ProposalType:         TypeCollections,
			TargetObjectType:     "ArInvoice",
			TargetObjectID:       inv.InvoiceID,
			Title:                fmt.Sprintf("Collections nudge: %s (%s)", retLabel, ar.FormatWholeSom(inv.BalanceMinor, cur)),
			Summary:              fmt.Sprintf("Invoice %s overdue by %d days; bilingual UZ/RU reminder ready", inv.InvoiceID, inv.DaysOverdue),
			Rationale:            fmt.Sprintf("Open AR balance %s past due %s (%d days overdue, dunning step %d)", ar.FormatWholeSom(inv.BalanceMinor, cur), inv.DueAt.Format("2006-01-02"), inv.DaysOverdue, inv.DunningStep),
			Payload:              payload,
			EvidenceRefs:         refs,
			ConfidenceBps:        8800,
			EstimatedImpactMinor: inv.BalanceMinor,
			Currency:             cur,
			RoiCategory:          roi.CategoryBadDebtAvoided,
			CreatedBy:            "agent:" + strings.ToLower(AgentCollections),
			ActorType:            auth.ActorTypeAgent,
			IdempotencyKey:       fmt.Sprintf("l1:%s:%s:%s:%s", AgentCollections, tid, inv.InvoiceID, today),
		})
		if err != nil {
			return created, err
		}
		created = append(created, p)
	}
	return created, nil
}

// RunExpiryRebalanceAgent scans near-expiry StockLots & InventoryCostLayers and generates
// governed rebalance proposals emitting inventory.rebalance.proposed.
func (s *Service) RunExpiryRebalanceAgent(ctx context.Context, tenantID string) ([]Proposal, error) {
	if s == nil || s.repo == nil || s.dataReader == nil {
		return nil, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	now := s.currentNow()
	cands, err := s.dataReader.ListExpiryRiskCandidates(ctx, tid, now, 30, 50)
	if err != nil {
		return nil, err
	}
	today := now.Format("2006-01-02")
	var created []Proposal

	for _, c := range cands {
		targetType := "StockLot"
		targetID := c.LotID
		if targetID == "" {
			targetType = "InventoryCostLayer"
			targetID = c.LayerID
		}
		if targetID == "" || c.QtyAvailable <= 0 {
			continue
		}

		active, err := s.repo.HasActiveForTarget(ctx, tid, targetType, targetID, TypeExpiryRebalance)
		if err != nil {
			return created, err
		}
		if active {
			continue
		}

		cur := nonEmptyOr(c.Currency, "UZS")
		actionMode := "rebalance_transfer"
		if c.TargetWarehouseID == "" || c.TargetWarehouseID == "MARKDOWN_CHANNEL" {
			actionMode = "markdown_clearance"
		}

		var refs []EvidenceRef
		if c.LotID != "" {
			refs = append(refs, EvidenceRef{Type: "StockLot", ID: c.LotID, Label: fmt.Sprintf("%d units exp %s (%dd)", c.QtyAvailable, c.ExpiryDate, c.DaysToExpiry)})
		}
		if c.LayerID != "" {
			refs = append(refs, EvidenceRef{Type: "InventoryCostLayer", ID: c.LayerID, Label: fmt.Sprintf("Unit cost %d %s", c.UnitCostMinor, cur)})
		}
		if c.SourceWarehouseID != "" {
			refs = append(refs, EvidenceRef{Type: "Warehouse", ID: c.SourceWarehouseID, Label: "Source warehouse"})
		}

		payload := map[string]any{
			"action":              actionMode,
			"lot_id":              c.LotID,
			"layer_id":            c.LayerID,
			"product_id":          c.ProductID,
			"product_name":        c.ProductName,
			"source_warehouse_id": c.SourceWarehouseID,
			"target_warehouse_id": c.TargetWarehouseID,
			"qty_to_rebalance":    c.QtyAvailable,
			"unit_cost_minor":     c.UnitCostMinor,
			"value_at_risk_minor": c.ValueAtRiskMinor,
			"currency":            cur,
			"expiry_date":         c.ExpiryDate,
			"days_to_expiry":      c.DaysToExpiry,
		}

		skuLabel := nonEmptyOr(c.ProductName, c.ProductID)
		p, err := s.CreateProposal(ctx, CreateProposalInput{
			TenantID:             tid,
			AgentID:              AgentExpiryRebalance,
			ProposalType:         TypeExpiryRebalance,
			TargetObjectType:     targetType,
			TargetObjectID:       targetID,
			Title:                fmt.Sprintf("Expiry rebalance %s (%d units · %dd left)", skuLabel, c.QtyAvailable, c.DaysToExpiry),
			Summary:              fmt.Sprintf("Move %d units from %s to %s before %s", c.QtyAvailable, c.SourceWarehouseID, c.TargetWarehouseID, c.ExpiryDate),
			Rationale:            fmt.Sprintf("Lot/layer expires %s (%d days remaining); %d units at risk (%s)", c.ExpiryDate, c.DaysToExpiry, c.QtyAvailable, ar.FormatWholeSom(c.ValueAtRiskMinor, cur)),
			Payload:              payload,
			EvidenceRefs:         refs,
			ConfidenceBps:        9000,
			EstimatedImpactMinor: c.ValueAtRiskMinor,
			Currency:             cur,
			RoiCategory:          roi.CategoryExpirySaved,
			CreatedBy:            "agent:" + strings.ToLower(AgentExpiryRebalance),
			ActorType:            auth.ActorTypeAgent,
			IdempotencyKey:       fmt.Sprintf("l1:%s:%s:%s:%s", AgentExpiryRebalance, tid, targetID, today),
		})
		if err != nil {
			return created, err
		}
		created = append(created, p)
	}
	return created, nil
}
