package p2p

import (
	"fmt"
	"math/big"
	"strings"
)

// ComputePOLineTotals calculates integer net, VAT, and total minor units using big.Int
// with half-up basis-point rounding (zero floating point arithmetic).
func ComputePOLineTotals(qtyOrdered, unitCostMinor, vatBps int64) (netMinor, vatMinor, totalMinor int64, err error) {
	if qtyOrdered <= 0 {
		return 0, 0, 0, fmt.Errorf("%w: qty_ordered must be > 0", ErrInvalidInput)
	}
	if unitCostMinor <= 0 {
		return 0, 0, 0, fmt.Errorf("%w: unit_cost_minor must be > 0", ErrInvalidInput)
	}
	if vatBps < 0 || vatBps > 10000 {
		return 0, 0, 0, fmt.Errorf("%w: vat_bps must be between 0 and 10000", ErrInvalidInput)
	}

	netBig := new(big.Int).Mul(big.NewInt(qtyOrdered), big.NewInt(unitCostMinor))
	if !netBig.IsInt64() {
		return 0, 0, 0, fmt.Errorf("%w: line net amount overflows int64", ErrInvalidInput)
	}
	netMinor = netBig.Int64()

	// vatMinor = (netMinor * vatBps + 5000) / 10000
	vatNum := new(big.Int).Mul(netBig, big.NewInt(vatBps))
	vatNum.Add(vatNum, big.NewInt(5000))
	vatBig := new(big.Int).Div(vatNum, big.NewInt(10000))
	if !vatBig.IsInt64() {
		return 0, 0, 0, fmt.Errorf("%w: line vat amount overflows int64", ErrInvalidInput)
	}
	vatMinor = vatBig.Int64()

	totBig := new(big.Int).Add(netBig, vatBig)
	if !totBig.IsInt64() {
		return 0, 0, 0, fmt.Errorf("%w: line total amount overflows int64", ErrInvalidInput)
	}
	totalMinor = totBig.Int64()
	return netMinor, vatMinor, totalMinor, nil
}

// ComputeNetMinor safely multiplies qty * unitCostMinor in int64.
func ComputeNetMinor(qty, unitCostMinor int64) (int64, error) {
	if qty <= 0 || unitCostMinor <= 0 {
		return 0, fmt.Errorf("%w: qty and unit_cost_minor must be > 0", ErrInvalidInput)
	}
	prod := new(big.Int).Mul(big.NewInt(qty), big.NewInt(unitCostMinor))
	if !prod.IsInt64() {
		return 0, fmt.Errorf("%w: line net amount overflows int64", ErrInvalidInput)
	}
	return prod.Int64(), nil
}

// ThreeWayMatchEvalInput bundles PO, receipts, invoice, optional inbound ESF, and tolerances.
type ThreeWayMatchEvalInput struct {
	PurchaseOrder  PurchaseOrder
	Receipts       []GoodsReceipt
	VendorInvoice  VendorInvoice
	InboundESF     *InboundESFDoc
	ToleranceBps   int64
	ToleranceMinor int64
}

// ThreeWayMatchEvalResult captures the deterministic 3-way match outcome.
type ThreeWayMatchEvalResult struct {
	Matched         bool
	ReceiptNetMinor int64
	ExpectedVat     int64
	ExpectedTotal   int64
	VarianceMinor   int64
	VarianceReason  string
}

// EvaluateThreeWayMatch validates PurchaseOrder <-> GoodsReceipts <-> VendorInvoice (+ optional Inbound ESF)
// using pure int64 arithmetic and explicit bps/minor tolerances.
func EvaluateThreeWayMatch(in ThreeWayMatchEvalInput) ThreeWayMatchEvalResult {
	po := in.PurchaseOrder
	inv := in.VendorInvoice

	if len(in.Receipts) == 0 {
		return ThreeWayMatchEvalResult{
			Matched:        false,
			VarianceMinor:  inv.TotalMinor,
			VarianceReason: "NO_GOODS_RECEIPT: purchase order has no posted goods receipt",
		}
	}

	var totalQtyReceived int64
	var receiptNetMinor int64
	var expectedReceiptVat int64
	for _, r := range in.Receipts {
		if len(r.Lines) > 0 {
			for _, l := range r.Lines {
				totalQtyReceived += l.QtyReceived
				lineNet := l.LineNetMinor
				if lineNet == 0 && l.QtyReceived > 0 && l.UnitCostMinor > 0 {
					lineNet = l.QtyReceived * l.UnitCostMinor
				}
				receiptNetMinor += lineNet
				vatBps := l.VatBps
				if vatBps == 0 {
					vatBps = inferPOVatBps(po, l.Sku, l.LineNum)
				}
				expectedReceiptVat += computeVatHalfUp(lineNet, vatBps)
			}
		} else {
			receiptNetMinor += r.TotalReceivedMinor
			expectedReceiptVat += computeVatHalfUp(r.TotalReceivedMinor, dominantPOVatBps(po))
		}
	}

	if receiptNetMinor <= 0 || (len(in.Receipts[0].Lines) > 0 && totalQtyReceived <= 0) {
		return ThreeWayMatchEvalResult{
			Matched:        false,
			VarianceMinor:  inv.TotalMinor,
			VarianceReason: "NO_GOODS_RECEIPT: zero received quantity/value on referenced goods receipt",
		}
	}

	expectedTotal := receiptNetMinor + expectedReceiptVat
	var reasons []string

	// 1. Validate PO currency vs VendorInvoice currency.
	if po.Currency != "" && inv.Currency != "" && !strings.EqualFold(po.Currency, inv.Currency) {
		reasons = append(reasons, fmt.Sprintf("CURRENCY_MISMATCH: po=%s invoice=%s", po.Currency, inv.Currency))
	}
	if po.VendorID != "" && inv.VendorID != "" && po.VendorID != inv.VendorID {
		reasons = append(reasons, fmt.Sprintf("VENDOR_MISMATCH: po=%s invoice=%s", po.VendorID, inv.VendorID))
	}

	// 2. Compare receipt net value against invoice subtotal within tolerance.
	subDiff := inv.SubtotalMinor - receiptNetMinor
	if !withinTolerance(absInt64(subDiff), receiptNetMinor, in.ToleranceBps, in.ToleranceMinor) {
		reasons = append(reasons, fmt.Sprintf("SUBTOTAL_MISMATCH: invoice_subtotal=%d receipt_net=%d variance=%d",
			inv.SubtotalMinor, receiptNetMinor, subDiff))
	}

	// 3. Ensure invoice does not exceed PO subtotal beyond tolerance (prevents over-billing PO).
	if inv.SubtotalMinor > po.SubtotalMinor {
		poOver := inv.SubtotalMinor - po.SubtotalMinor
		if !withinTolerance(poOver, po.SubtotalMinor, in.ToleranceBps, in.ToleranceMinor) {
			reasons = append(reasons, fmt.Sprintf("PO_OVERBILL: invoice_subtotal=%d po_subtotal=%d variance=%d",
				inv.SubtotalMinor, po.SubtotalMinor, poOver))
		}
	}

	// 4. Compare expected VAT on received goods against invoice VAT within tolerance.
	vatDiff := inv.VatMinor - expectedReceiptVat
	vatBase := expectedReceiptVat
	if vatBase <= 0 {
		vatBase = receiptNetMinor
	}
	if !withinTolerance(absInt64(vatDiff), vatBase, in.ToleranceBps, in.ToleranceMinor) {
		reasons = append(reasons, fmt.Sprintf("VAT_MISMATCH: invoice_vat=%d expected_vat=%d variance=%d",
			inv.VatMinor, expectedReceiptVat, vatDiff))
	}

	// 5. Optional Inbound ESF verification (3rd party fiscal document tie-out).
	esfVariance := int64(0)
	if in.InboundESF != nil {
		esf := in.InboundESF
		if !strings.EqualFold(esf.Direction, "INBOUND") {
			reasons = append(reasons, fmt.Sprintf("ESF_DIRECTION_MISMATCH: direction=%s", esf.Direction))
		}
		if strings.EqualFold(esf.Status, "REJECTED") || strings.EqualFold(esf.Status, "CANCELLED") || strings.EqualFold(esf.Status, "FAILED") {
			reasons = append(reasons, fmt.Sprintf("ESF_INVALID_STATUS: status=%s", esf.Status))
		}
		if esf.PoID != "" && esf.PoID != po.PoID {
			reasons = append(reasons, fmt.Sprintf("ESF_PO_MISMATCH: esf_po=%s po=%s", esf.PoID, po.PoID))
		}
		if po.VendorTin != "" && esf.CounterpartyTin != "" && strings.TrimSpace(po.VendorTin) != strings.TrimSpace(esf.CounterpartyTin) {
			reasons = append(reasons, fmt.Sprintf("ESF_TIN_MISMATCH: esf_tin=%s vendor_tin=%s", esf.CounterpartyTin, po.VendorTin))
		}
		esfDiff := inv.TotalMinor - esf.TotalMinor
		if !withinTolerance(absInt64(esfDiff), esf.TotalMinor, in.ToleranceBps, in.ToleranceMinor) {
			reasons = append(reasons, fmt.Sprintf("ESF_TOTAL_MISMATCH: invoice_total=%d esf_total=%d variance=%d",
				inv.TotalMinor, esf.TotalMinor, esfDiff))
			esfVariance = esfDiff
		}
		esfVatDiff := inv.VatMinor - esf.VatMinor
		if !withinTolerance(absInt64(esfVatDiff), maxInt64(esf.VatMinor, 1), in.ToleranceBps, in.ToleranceMinor) {
			reasons = append(reasons, fmt.Sprintf("ESF_VAT_MISMATCH: invoice_vat=%d esf_vat=%d variance=%d",
				inv.VatMinor, esf.VatMinor, esfVatDiff))
			if esfVariance == 0 {
				esfVariance = esfVatDiff
			}
		}
	}

	totalDiff := inv.TotalMinor - expectedTotal
	variance := totalDiff
	if variance == 0 && subDiff != 0 {
		variance = subDiff
	}
	if variance == 0 && esfVariance != 0 {
		variance = esfVariance
	}

	if len(reasons) > 0 {
		return ThreeWayMatchEvalResult{
			Matched:         false,
			ReceiptNetMinor: receiptNetMinor,
			ExpectedVat:     expectedReceiptVat,
			ExpectedTotal:   expectedTotal,
			VarianceMinor:   variance,
			VarianceReason:  strings.Join(reasons, "; "),
		}
	}

	reason := ""
	if variance != 0 {
		reason = "WITHIN_TOLERANCE"
	}
	return ThreeWayMatchEvalResult{
		Matched:         true,
		ReceiptNetMinor: receiptNetMinor,
		ExpectedVat:     expectedReceiptVat,
		ExpectedTotal:   expectedTotal,
		VarianceMinor:   variance,
		VarianceReason:  reason,
	}
}

// BuildThreeWayMatchGLLines constructs a strictly balanced GL entry clearing GRNI + TAX:VAT:INPUT -> AP:VENDOR
// (plus COGS variance leg if subtotal differs within tolerance).
func BuildThreeWayMatchGLLines(po PurchaseOrder, inv VendorInvoice, receiptNetMinor int64) ([]GLLine, error) {
	if inv.TotalMinor <= 0 {
		return nil, fmt.Errorf("%w: vendor invoice total_minor must be > 0", ErrInvalidInput)
	}
	if receiptNetMinor < 0 {
		return nil, fmt.Errorf("%w: receipt_net_minor must be >= 0", ErrInvalidInput)
	}
	lines := make([]GLLine, 0, 4)
	if receiptNetMinor > 0 {
		lines = append(lines, GLLine{
			AccountCode: AccountGRNI,
			Direction:   DirectionDebit,
			AmountMinor: receiptNetMinor,
			PartyType:   "VENDOR",
			PartyID:     po.VendorID,
			Description: "Clear GRNI on 3-way match for PO " + po.PoID,
		})
	}
	if inv.VatMinor > 0 {
		lines = append(lines, GLLine{
			AccountCode: AccountVatInput,
			Direction:   DirectionDebit,
			AmountMinor: inv.VatMinor,
			PartyType:   "VENDOR",
			PartyID:     po.VendorID,
			Description: "Recoverable Input VAT on Vendor Invoice " + inv.VendorInvoiceID,
		})
	}
	if delta := inv.SubtotalMinor - receiptNetMinor; delta != 0 {
		dir := DirectionDebit
		amt := delta
		if delta < 0 {
			dir = DirectionCredit
			amt = -delta
		}
		lines = append(lines, GLLine{
			AccountCode: AccountPriceVariance,
			Direction:   dir,
			AmountMinor: amt,
			PartyType:   "VENDOR",
			PartyID:     po.VendorID,
			Description: "Purchase price variance within tolerance for PO " + po.PoID,
		})
	}
	lines = append(lines, GLLine{
		AccountCode: AccountVendorPayable,
		Direction:   DirectionCredit,
		AmountMinor: inv.TotalMinor,
		PartyType:   "VENDOR",
		PartyID:     po.VendorID,
		Description: "Vendor Payable recognized on 3-way match " + inv.VendorInvoiceID,
	})
	return lines, nil
}

func withinTolerance(absDiff, base, tolBps, tolMinor int64) bool {
	if absDiff == 0 {
		return true
	}
	if tolMinor > 0 && absDiff <= tolMinor {
		return true
	}
	if tolBps > 0 && base > 0 {
		lhs := new(big.Int).Mul(big.NewInt(absDiff), big.NewInt(10000))
		rhs := new(big.Int).Mul(big.NewInt(base), big.NewInt(tolBps))
		if lhs.Cmp(rhs) <= 0 {
			return true
		}
	}
	return false
}

func computeVatHalfUp(netMinor, vatBps int64) int64 {
	if netMinor <= 0 || vatBps <= 0 {
		return 0
	}
	num := new(big.Int).Mul(big.NewInt(netMinor), big.NewInt(vatBps))
	num.Add(num, big.NewInt(5000))
	res := new(big.Int).Div(num, big.NewInt(10000))
	return res.Int64()
}

func inferPOVatBps(po PurchaseOrder, sku string, lineNum int64) int64 {
	for _, l := range po.Lines {
		if (lineNum > 0 && l.LineNum == lineNum) || (sku != "" && strings.EqualFold(l.Sku, sku)) {
			return l.VatBps
		}
	}
	return dominantPOVatBps(po)
}

func dominantPOVatBps(po PurchaseOrder) int64 {
	if len(po.Lines) == 0 {
		return DefaultVatRateBps
	}
	return po.Lines[0].VatBps
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
