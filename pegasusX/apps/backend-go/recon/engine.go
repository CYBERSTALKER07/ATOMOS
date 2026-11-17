package recon

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// EvaluateSnapshot runs pure deterministic four-way reconciliation checks over a consistent Snapshot.
// All amounts and deltas stay strictly in int64 minor units (`DeltaMinor = ActualMinor - ExpectedMinor`).
func EvaluateSnapshot(snap Snapshot, requestedDims []string, asOf time.Time) (ReconciliationRun, []ReconciliationBreak) {
	if asOf.IsZero() {
		asOf = snap.AsOf
	}
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	currency := strings.ToUpper(strings.TrimSpace(snap.Currency))
	if currency == "" {
		currency = "UZS"
	}

	dims := normalizeDimensions(requestedDims)
	runID := "recon-run-" + uuid.NewString()

	checks := make(map[string]DimensionCheckSummary, len(dims))
	var breaks []ReconciliationBreak
	var totalExposure int64
	var summary RunSummary

	for _, dim := range dims {
		var dimCheck DimensionCheckSummary
		var dimBreaks []ReconciliationBreak
		switch dim {
		case DimensionOrdersAR:
			dimCheck, dimBreaks = evaluateOrdersAR(snap, runID, currency, asOf)
		case DimensionStockGL:
			dimCheck, dimBreaks = evaluateStockGL(snap, runID, currency, asOf)
		case DimensionCashPSP:
			dimCheck, dimBreaks = evaluateCashPSP(snap, runID, currency, asOf)
		case DimensionTaxFiscal:
			dimCheck, dimBreaks = evaluateTaxFiscal(snap, runID, currency, asOf)
		}
		var dimExposure int64
		for _, b := range dimBreaks {
			dimExposure += absInt64(b.DeltaMinor)
			switch b.Severity {
			case SeverityCritical:
				summary.CriticalBreaks++
			case SeverityHigh:
				summary.HighBreaks++
			case SeverityMedium:
				summary.MediumBreaks++
			default:
				summary.LowBreaks++
			}
		}
		dimCheck.BreakCount = int64(len(dimBreaks))
		dimCheck.ExposureMinor = dimExposure
		if len(dimBreaks) == 0 {
			dimCheck.Status = DimStatusGreen
			summary.GreenDimensions++
		} else {
			dimCheck.Status = DimStatusBreak
			summary.BreakDimensions++
		}
		checks[dim] = dimCheck
		breaks = append(breaks, dimBreaks...)
		totalExposure += dimExposure
	}

	summary.TotalBreaks = int64(len(breaks))
	runStatus := RunStatusGreen
	if len(breaks) > 0 {
		runStatus = RunStatusBreaksFound
	}

	run := ReconciliationRun{
		RunID:         runID,
		EntityID:      snap.EntityID,
		PeriodID:      snap.PeriodID,
		Status:        runStatus,
		BreakCount:    int64(len(breaks)),
		ExposureMinor: totalExposure,
		Currency:      currency,
		Checks:        checks,
		Summary:       summary,
		StartedAt:     asOf,
		CompletedAt:   asOf,
	}
	return run, breaks
}

func evaluateOrdersAR(snap Snapshot, runID, currency string, now time.Time) (DimensionCheckSummary, []ReconciliationBreak) {
	var dimBreaks []ReconciliationBreak
	for _, a := range snap.DeliveredCreditAnomalies {
		delta := a.InvoicePrincipalMinor - a.OrderTotalMinor
		sev := maxSeverity(SeverityHigh, severityForDelta(delta))
		note := fmt.Sprintf("delivered credit order %s expects ArInvoice principal %d, found %d (status=%s)",
			a.OrderID, a.OrderTotalMinor, a.InvoicePrincipalMinor, coalesce(a.InvoiceStatus, "MISSING"))
		dimBreaks = append(dimBreaks, newOpenBreak(
			runID, snap.EntityID, DimensionOrdersAR, ObjectOrder, a.OrderID,
			a.OrderTotalMinor, a.InvoicePrincipalMinor, delta, currency, sev, note, now,
		))
	}

	delta := snap.GLARRetailerNetMinor - snap.OpenARBalanceMinor
	if delta != 0 {
		sev := severityForDelta(delta)
		note := fmt.Sprintf("GL AR:RETAILER net balance (%d) diverges from open ArInvoices balance (%d)",
			snap.GLARRetailerNetMinor, snap.OpenARBalanceMinor)
		dimBreaks = append(dimBreaks, newOpenBreak(
			runID, snap.EntityID, DimensionOrdersAR, ObjectDimension, DimensionOrdersAR,
			snap.OpenARBalanceMinor, snap.GLARRetailerNetMinor, delta, currency, sev, note, now,
		))
	}

	check := DimensionCheckSummary{
		Dimension:     DimensionOrdersAR,
		ExpectedMinor: snap.OpenARBalanceMinor,
		ActualMinor:   snap.GLARRetailerNetMinor,
		DeltaMinor:    delta,
		Currency:      currency,
		Metrics: map[string]int64{
			"delivered_credit_orders_count": snap.DeliveredCreditOrderCount,
			"delivered_credit_orders_minor": snap.DeliveredCreditOrderMinor,
			"order_ar_anomalies_count":      int64(len(snap.DeliveredCreditAnomalies)),
			"open_ar_invoices_count":        snap.OpenARInvoiceCount,
			"open_ar_balance_minor":         snap.OpenARBalanceMinor,
			"gl_ar_retailer_net_minor":      snap.GLARRetailerNetMinor,
		},
	}
	return check, dimBreaks
}

func evaluateStockGL(snap Snapshot, runID, currency string, now time.Time) (DimensionCheckSummary, []ReconciliationBreak) {
	var dimBreaks []ReconciliationBreak

	for _, lot := range snap.MissingWarehouseLots {
		expected := lot.QuantityTotal
		if lot.UnitCostMinor > 0 {
			expected = lot.QuantityTotal * lot.UnitCostMinor
		}
		note := fmt.Sprintf("active lot %s (sku=%s qty=%d unit_cost=%d) has empty WarehouseId",
			lot.LotID, lot.ProductID, lot.QuantityTotal, lot.UnitCostMinor)
		dimBreaks = append(dimBreaks, newOpenBreak(
			runID, snap.EntityID, DimensionStockGL, ObjectStockLot, lot.LotID,
			expected, 0, -expected, currency, SeverityHigh, note, now,
		))
	}

	for _, m := range snap.V2Mismatches {
		var expected, actual int64
		onHandChanged := m.LotAvailable != m.V2Available
		if m.UnitCostMinor > 0 {
			if onHandChanged {
				expected = m.LotAvailable * m.UnitCostMinor
				actual = m.V2Available * m.UnitCostMinor
			} else {
				// Bucket mismatch (e.g. reserved vs V2 reserved) at equal total on-hand units.
				expected = m.LotReserved * m.UnitCostMinor
				actual = m.V2Reserved * m.UnitCostMinor
			}
		} else {
			if onHandChanged {
				expected = m.LotAvailable
				actual = m.V2Available
			} else {
				expected = m.LotReserved
				actual = m.V2Reserved
			}
		}
		delta := actual - expected
		sev := maxSeverity(SeverityHigh, severityForDelta(delta))
		note := fmt.Sprintf("StockLots vs V2 mismatch in %s:%s (missing_v2=%t delta_avail=%d delta_resv=%d delta_quar=%d unit_cost=%d)",
			m.WarehouseID, m.ProductID, m.MissingV2, m.DeltaAvailable, m.DeltaReserved, m.DeltaQuarantine, m.UnitCostMinor)
		dimBreaks = append(dimBreaks, newOpenBreak(
			runID, snap.EntityID, DimensionStockGL, ObjectStockLot, m.WarehouseID+":"+m.ProductID,
			expected, actual, delta, currency, sev, note, now,
		))
	}

	for _, u := range snap.UncostedLots {
		note := fmt.Sprintf("uncosted active StockLot %s (warehouse=%s product=%s qty=%d) lacks UnitCostMinor and catalog price",
			u.LotID, u.WarehouseID, u.ProductID, u.Quantity)
		dimBreaks = append(dimBreaks, newOpenBreak(
			runID, snap.EntityID, DimensionStockGL, ObjectStockLotUncosted, u.LotID,
			0, 0, 0, currency, SeverityMedium, note, now,
		))
	}

	expectedStockVal := snap.CostedStockValuedMinor
	sourceLabel := "StockLots"
	if snap.HasCostLayers {
		expectedStockVal = snap.CostLayersValuedMinor
		sourceLabel = "InventoryCostLayers"
	}

	delta := snap.GLInventoryNetMinor - expectedStockVal
	if delta != 0 {
		sev := severityForDelta(delta)
		note := fmt.Sprintf("GL INVENTORY balance (%d) diverges from valued %s (%d)",
			snap.GLInventoryNetMinor, sourceLabel, expectedStockVal)
		dimBreaks = append(dimBreaks, newOpenBreak(
			runID, snap.EntityID, DimensionStockGL, ObjectDimension, DimensionStockGL,
			expectedStockVal, snap.GLInventoryNetMinor, delta, currency, sev, note, now,
		))
	}

	check := DimensionCheckSummary{
		Dimension:     DimensionStockGL,
		ExpectedMinor: expectedStockVal,
		ActualMinor:   snap.GLInventoryNetMinor,
		DeltaMinor:    delta,
		Currency:      currency,
		Metrics: map[string]int64{
			"warehouses_checked":        int64(snap.WarehousesChecked),
			"v2_mismatch_count":         int64(len(snap.V2Mismatches)),
			"uncosted_lots_count":       int64(len(snap.UncostedLots)),
			"cost_layers_count":         snap.CostLayersCount,
			"cost_layers_qty_remaining": snap.CostLayersQtyRemaining,
			"cost_layers_valued_minor":  snap.CostLayersValuedMinor,
			"stock_valued_minor":        expectedStockVal,
			"gl_inventory_minor":        snap.GLInventoryNetMinor,
		},
	}
	return check, dimBreaks
}

func evaluateCashPSP(snap Snapshot, runID, currency string, now time.Time) (DimensionCheckSummary, []ReconciliationBreak) {
	var dimBreaks []ReconciliationBreak

	// 1. PSP:GATEWAY vs settled PaymentSessions + fallback card legs - chargebacks (+ verified AR card)
	expectedPSP := snap.SettledPSPSessionsMinor + snap.FallbackCapturedCardLegsMinor -
		snap.PaymentChargebacksMinor + snap.VerifiedARCardGLMinor
	deltaPSP := snap.GLPSPGatewayNetMinor - expectedPSP
	if deltaPSP != 0 {
		note := fmt.Sprintf("GL PSP:GATEWAY (%d) diverges from settled PSP subledger (%d)",
			snap.GLPSPGatewayNetMinor, expectedPSP)
		dimBreaks = append(dimBreaks, newOpenBreak(
			runID, snap.EntityID, DimensionCashPSP, ObjectGLAccount, "PSP:GATEWAY",
			expectedPSP, snap.GLPSPGatewayNetMinor, deltaPSP, currency, severityForDelta(deltaPSP), note, now,
		))
	}

	// 2. CASH:OFFICE vs accepted shortfall repayments into office vault + verified AR cash-office receipts.
	// Balanced cash shifts (DifferenceMinor == 0) post 0 GL rows and remain in CASH:DRIVER.
	expectedCashOffice := snap.CashReconAcceptedShortfallRepaidMinor + snap.VerifiedARCashOfficeGLMinor
	deltaCashOffice := snap.GLCashOfficeNetMinor - expectedCashOffice
	if deltaCashOffice != 0 {
		note := fmt.Sprintf("GL CASH:OFFICE (%d) diverges from expected vault inflows (%d)",
			snap.GLCashOfficeNetMinor, expectedCashOffice)
		dimBreaks = append(dimBreaks, newOpenBreak(
			runID, snap.EntityID, DimensionCashPSP, ObjectGLAccount, "CASH:OFFICE",
			expectedCashOffice, snap.GLCashOfficeNetMinor, deltaCashOffice, currency, severityForDelta(deltaCashOffice), note, now,
		))
	}

	// 3. Disputed CashReconciliations needing finance resolution
	for _, cr := range snap.DisputedCashRecons {
		deltaCR := cr.DeclaredCashMinor - cr.ExpectedCashMinor
		sev := maxSeverity(SeverityMedium, severityForDelta(deltaCR))
		note := fmt.Sprintf("DISPUTED CashReconciliation %s for driver %s (expected=%d declared=%d diff=%d)",
			cr.ReconciliationID, cr.DriverID, cr.ExpectedCashMinor, cr.DeclaredCashMinor, cr.DifferenceMinor)
		dimBreaks = append(dimBreaks, newOpenBreak(
			runID, snap.EntityID, DimensionCashPSP, ObjectCashRecon, cr.ReconciliationID,
			cr.ExpectedCashMinor, cr.DeclaredCashMinor, deltaCR, currency, sev, note, now,
		))
	}

	// 4. CASH:DRIVER + CASH:OFFICE combined tie-out against cash deliveries, AR cash receipts,
	// written-off/pending shortfalls, and driver overages.
	expectedCashCombined := snap.DriverCashCollectedMinor +
		snap.VerifiedARCashOfficeGLMinor +
		snap.VerifiedARCashDriverGLMinor -
		snap.CashReconWrittenOffShortfallMinor -
		snap.CashReconPendingOrDisputedShortfallMinor +
		snap.CashReconOverageMinor
	actualCashCombined := snap.GLCashDriverNetMinor + snap.GLCashOfficeNetMinor
	deltaCashCombined := actualCashCombined - expectedCashCombined
	if deltaCashCombined != 0 && deltaCashCombined != deltaCashOffice {
		note := fmt.Sprintf("GL CASH:DRIVER+CASH:OFFICE (%d) diverges from expected driver/office cash (%d)",
			actualCashCombined, expectedCashCombined)
		dimBreaks = append(dimBreaks, newOpenBreak(
			runID, snap.EntityID, DimensionCashPSP, ObjectGLAccount, "CASH:DRIVER",
			expectedCashCombined, actualCashCombined, deltaCashCombined, currency, severityForDelta(deltaCashCombined), note, now,
		))
	}

	// 5. Overall 4-account liquidity tie-out (PSP:GATEWAY + CASH:DRIVER + CASH:OFFICE + BANK:OPERATING)
	expectedTotal := expectedPSP +
		expectedCashCombined +
		snap.VerifiedARBankGLMinor +
		snap.VerifiedCreditTopupBankMinor -
		snap.BankPaidPayoutsMinor
	actualTotal := snap.GLPSPGatewayNetMinor +
		actualCashCombined +
		snap.GLBankOperatingNetMinor
	deltaTotal := actualTotal - expectedTotal

	// Avoid duplicating a break when the total delta is already explained 1:1 by PSP:GATEWAY / CASH accounts.
	if deltaTotal != 0 && deltaTotal != deltaPSP && deltaTotal != deltaCashCombined && deltaTotal != deltaCashOffice &&
		deltaTotal != deltaPSP+deltaCashCombined && deltaTotal != deltaPSP+deltaCashOffice {
		note := fmt.Sprintf("GL 4-account cash/bank/PSP total (%d) diverges from subledger expected (%d)",
			actualTotal, expectedTotal)
		dimBreaks = append(dimBreaks, newOpenBreak(
			runID, snap.EntityID, DimensionCashPSP, ObjectDimension, DimensionCashPSP,
			expectedTotal, actualTotal, deltaTotal, currency, severityForDelta(deltaTotal), note, now,
		))
	}

	check := DimensionCheckSummary{
		Dimension:     DimensionCashPSP,
		ExpectedMinor: expectedTotal,
		ActualMinor:   actualTotal,
		DeltaMinor:    deltaTotal,
		Currency:      currency,
		Metrics: map[string]int64{
			"settled_psp_minor":                 snap.SettledPSPSessionsMinor + snap.FallbackCapturedCardLegsMinor - snap.PaymentChargebacksMinor,
			"gl_psp_gateway_minor":              snap.GLPSPGatewayNetMinor,
			"driver_cash_collected_minor":       snap.DriverCashCollectedMinor,
			"cash_recon_shortfall_repaid_minor": snap.CashReconAcceptedShortfallRepaidMinor,
			"cash_recon_written_off_minor":      snap.CashReconWrittenOffShortfallMinor,
			"cash_recon_overage_minor":          snap.CashReconOverageMinor,
			"gl_cash_driver_minor":              snap.GLCashDriverNetMinor,
			"gl_cash_office_minor":              snap.GLCashOfficeNetMinor,
			"gl_bank_operating_minor":           snap.GLBankOperatingNetMinor,
			"disputed_cash_recon_count":         int64(len(snap.DisputedCashRecons)),
		},
	}
	return check, dimBreaks
}

func evaluateTaxFiscal(snap Snapshot, runID, currency string, now time.Time) (DimensionCheckSummary, []ReconciliationBreak) {
	var dimBreaks []ReconciliationBreak

	registeredVatNetMinor := snap.SnapshotVatMinor - snap.CreditNoteVatMinor
	delta := snap.GLVatOutputCreditMinor - registeredVatNetMinor

	for _, mo := range snap.MissingSnapshotOrders {
		// Surface missing fiscal snapshot when GL recorded output VAT or a fiscal receipt exists without line snapshot.
		note := fmt.Sprintf("fiscalized order %s (total=%d) has no OrderLineFiscalSnapshots rows", mo.OrderID, mo.TotalMinor)
		dimBreaks = append(dimBreaks, newOpenBreak(
			runID, snap.EntityID, DimensionTaxFiscal, ObjectOrder, mo.OrderID,
			mo.TotalMinor, 0, -mo.TotalMinor, currency, SeverityHigh, note, now,
		))
	}

	if delta != 0 {
		sev := severityForDelta(delta)
		note := fmt.Sprintf("GL TAX:VAT:OUTPUT net credit (%d) diverges from fiscal registered VAT net (%d)",
			snap.GLVatOutputCreditMinor, registeredVatNetMinor)
		dimBreaks = append(dimBreaks, newOpenBreak(
			runID, snap.EntityID, DimensionTaxFiscal, ObjectDimension, DimensionTaxFiscal,
			registeredVatNetMinor, snap.GLVatOutputCreditMinor, delta, currency, sev, note, now,
		))
	}

	check := DimensionCheckSummary{
		Dimension:     DimensionTaxFiscal,
		ExpectedMinor: registeredVatNetMinor,
		ActualMinor:   snap.GLVatOutputCreditMinor,
		DeltaMinor:    delta,
		Currency:      currency,
		Metrics: map[string]int64{
			"fiscalized_orders_count":  snap.FiscalizedOrderCount,
			"snapshot_vat_minor":       snap.SnapshotVatMinor,
			"credit_note_vat_minor":    snap.CreditNoteVatMinor,
			"registered_vat_net_minor": registeredVatNetMinor,
			"gl_vat_output_minor":      snap.GLVatOutputCreditMinor,
		},
	}
	return check, dimBreaks
}

func newOpenBreak(
	runID, entityID, dimension, objectType, objectID string,
	expectedMinor, actualMinor, deltaMinor int64,
	currency, severity, note string,
	now time.Time,
) ReconciliationBreak {
	due := now.Add(slaDurationForSeverity(severity))
	return ReconciliationBreak{
		BreakID:        uuid.NewString(),
		RunID:          runID,
		EntityID:       entityID,
		Dimension:      dimension,
		ObjectType:     objectType,
		ObjectID:       objectID,
		ExpectedMinor:  expectedMinor,
		ActualMinor:    actualMinor,
		DeltaMinor:     deltaMinor,
		Currency:       currency,
		Severity:       severity,
		Status:         BreakStatusOpen,
		OwnerID:        ownerForDimension(dimension),
		SlaDueAt:       &due,
		ResolutionNote: note,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func normalizeDimensions(requested []string) []string {
	if len(requested) == 0 {
		out := make([]string, len(AllDimensions))
		copy(out, AllDimensions)
		return out
	}
	allowed := map[string]bool{
		DimensionOrdersAR:  true,
		DimensionStockGL:   true,
		DimensionCashPSP:   true,
		DimensionTaxFiscal: true,
	}
	seen := make(map[string]bool, len(requested))
	out := make([]string, 0, len(requested))
	for _, raw := range requested {
		d := strings.ToUpper(strings.TrimSpace(raw))
		if allowed[d] && !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	return out
}

func ownerForDimension(dimension string) string {
	switch dimension {
	case DimensionOrdersAR:
		return OwnerFinanceController
	case DimensionStockGL:
		return OwnerWarehouseManager
	case DimensionCashPSP:
		return OwnerTreasuryOps
	case DimensionTaxFiscal:
		return OwnerTaxCompliance
	default:
		return OwnerFinanceController
	}
}

func severityForDelta(delta int64) string {
	abs := absInt64(delta)
	switch {
	case abs >= 10_000_000:
		return SeverityCritical
	case abs >= 1_000_000:
		return SeverityHigh
	case abs >= 100_000:
		return SeverityMedium
	default:
		return SeverityLow
	}
}

func slaDurationForSeverity(sev string) time.Duration {
	switch strings.ToUpper(strings.TrimSpace(sev)) {
	case SeverityCritical:
		return 4 * time.Hour
	case SeverityHigh:
		return 24 * time.Hour
	case SeverityMedium:
		return 48 * time.Hour
	default:
		return 72 * time.Hour
	}
}

func severityRank(sev string) int {
	switch strings.ToUpper(strings.TrimSpace(sev)) {
	case SeverityCritical:
		return 4
	case SeverityHigh:
		return 3
	case SeverityMedium:
		return 2
	case SeverityLow:
		return 1
	default:
		return 0
	}
}

func maxSeverity(a, b string) string {
	if severityRank(a) >= severityRank(b) {
		return a
	}
	return b
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func coalesce(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
