package exportpacks

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

var canonicalDPDBuckets = []string{"0-30", "31-60", "61-90", "90+"}

// CollectBank collects all 15 standard audit tables plus Bank Credit Tape tables inside a
// single ReadOnlyTransaction snapshot using deterministic keyset pagination.
func (c *SpannerCollector) CollectBank(ctx context.Context, entityID, periodID string, from, to time.Time) (*PackDatasets, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("%w: spanner collector unavailable", ErrRepoUnavailable)
	}
	eid := strings.TrimSpace(entityID)
	pid := strings.TrimSpace(periodID)
	if eid == "" {
		return nil, fmt.Errorf("%w: entity_id required for dataset collection", ErrInvalidInput)
	}

	batch := c.batchSize
	if batch <= 0 {
		batch = defaultCollectorBatchSize
	}

	roTxn := c.client.ReadOnlyTransaction()
	defer roTxn.Close()

	out, err := collectStandardDatasetsInTxn(ctx, roTxn, eid, pid, from, to, batch)
	if err != nil {
		return nil, err
	}
	if out.BankArInvoices, err = collectBankArInvoices(ctx, roTxn, eid, to, batch); err != nil {
		return nil, err
	}
	if out.RetailerPaymentTerms, err = collectRetailerPaymentTerms(ctx, roTxn, eid, batch); err != nil {
		return nil, err
	}
	if out.RetailerCreditProfiles, err = collectRetailerCreditProfiles(ctx, roTxn, eid, batch); err != nil {
		return nil, err
	}
	if out.ArLedgerEntries, err = collectArLedgerEntries(ctx, roTxn, eid, to, batch); err != nil {
		return nil, err
	}

	retailerSet := make(map[string]struct{})
	for _, t := range out.RetailerPaymentTerms {
		if rid := strings.TrimSpace(t.RetailerID); rid != "" {
			retailerSet[rid] = struct{}{}
		}
	}
	for _, p := range out.RetailerCreditProfiles {
		if rid := strings.TrimSpace(p.RetailerID); rid != "" {
			retailerSet[rid] = struct{}{}
		}
	}
	for _, inv := range out.BankArInvoices {
		if rid := strings.TrimSpace(inv.RetailerID); rid != "" {
			retailerSet[rid] = struct{}{}
		}
	}
	retailerIDs := make([]string, 0, len(retailerSet))
	for rid := range retailerSet {
		retailerIDs = append(retailerIDs, rid)
	}
	sort.Strings(retailerIDs)

	if out.CreditScoreHistory, err = collectCreditScoreHistory(ctx, roTxn, eid, retailerIDs, batch); err != nil {
		return nil, err
	}
	return out, nil
}

func collectBankArInvoices(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, eid string, to time.Time, batch int) ([]ArInvoiceRow, error) {
	var rows []ArInvoiceRow
	err := runKeysetPages(ctx, roTxn, batch, "ArInvoices(bank)", func(has bool, lastID string) spanner.Statement {
		sql := `SELECT InvoiceId, SupplierId, RetailerId, OrderId, Status, PrincipalMinor, BalanceMinor, Currency, CreditLeaveAt, DueAt,
		               TermsDays, GracePeriodDays, COALESCE(AgingBucket, ''), LastDunnedAt, DunningStep, CreatedAt, UpdatedAt
		        FROM ArInvoices WHERE SupplierId = @eid AND CreatedAt <= @to`
		params := map[string]any{"eid": eid, "to": to.UTC()}
		if has {
			sql += ` AND InvoiceId > @k1`
			params["k1"] = lastID
		}
		sql += ` ORDER BY InvoiceId ASC LIMIT @lim`
		return spanner.Statement{SQL: sql, Params: params}
	}, func(row *spanner.Row) (string, error) {
		var r ArInvoiceRow
		var lastDun spanner.NullTime
		if err := row.Columns(&r.InvoiceID, &r.SupplierID, &r.RetailerID, &r.OrderID, &r.Status, &r.PrincipalMinor, &r.BalanceMinor,
			&r.Currency, &r.CreditLeaveAt, &r.DueAt, &r.TermsDays, &r.GracePeriodDays, &r.AgingBucket, &lastDun, &r.DunningStep, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return "", err
		}
		r.CreditLeaveAt = r.CreditLeaveAt.UTC()
		r.DueAt = r.DueAt.UTC()
		r.LastDunnedAt = optTimePtr(lastDun)
		r.CreatedAt = r.CreatedAt.UTC()
		r.UpdatedAt = r.UpdatedAt.UTC()
		rows = append(rows, r)
		return r.InvoiceID, nil
	})
	return rows, err
}

func collectRetailerPaymentTerms(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, eid string, batch int) ([]RetailerPaymentTermsRow, error) {
	var rows []RetailerPaymentTermsRow
	err := runKeysetPages(ctx, roTxn, batch, "RetailerPaymentTerms", func(has bool, lastRetailerID string) spanner.Statement {
		sql := `SELECT RetailerId, SupplierId, CreditEnabled, EnabledAt, DisabledAt, TermsDays, GracePeriodDays,
		               CreditLimitMinor, UseGlobalDefaults, ConsentVerified, Version, UpdatedAt
		        FROM RetailerPaymentTerms WHERE SupplierId = @eid`
		params := map[string]any{"eid": eid}
		if has {
			sql += ` AND RetailerId > @k1`
			params["k1"] = lastRetailerID
		}
		sql += ` ORDER BY RetailerId ASC LIMIT @lim`
		return spanner.Statement{SQL: sql, Params: params}
	}, func(row *spanner.Row) (string, error) {
		var r RetailerPaymentTermsRow
		var enAt, disAt spanner.NullTime
		if err := row.Columns(&r.RetailerID, &r.SupplierID, &r.CreditEnabled, &enAt, &disAt, &r.TermsDays, &r.GracePeriodDays,
			&r.CreditLimitMinor, &r.UseGlobalDefaults, &r.ConsentVerified, &r.Version, &r.UpdatedAt); err != nil {
			return "", err
		}
		r.EnabledAt = optTimePtr(enAt)
		r.DisabledAt = optTimePtr(disAt)
		r.UpdatedAt = r.UpdatedAt.UTC()
		rows = append(rows, r)
		return r.RetailerID, nil
	})
	return rows, err
}

func collectRetailerCreditProfiles(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, eid string, batch int) ([]RetailerCreditProfileRow, error) {
	var rows []RetailerCreditProfileRow
	err := runKeysetPages(ctx, roTxn, batch, "RetailerCreditProfiles", func(has bool, lastRetailerID string) spanner.Statement {
		sql := `SELECT RetailerId, SupplierId, CreditLimitMinor, CurrentBalanceMinor, ReservedMinor, AvailableCreditMinor,
		               RiskScore, DelinquencyCount, Status, COALESCE(PrepayOnlyLocked, false), LastEvaluatedAt, CreatedAt, UpdatedAt
		        FROM RetailerCreditProfiles WHERE SupplierId = @eid`
		params := map[string]any{"eid": eid}
		if has {
			sql += ` AND RetailerId > @k1`
			params["k1"] = lastRetailerID
		}
		sql += ` ORDER BY RetailerId ASC LIMIT @lim`
		return spanner.Statement{SQL: sql, Params: params}
	}, func(row *spanner.Row) (string, error) {
		var r RetailerCreditProfileRow
		var lastEval spanner.NullTime
		if err := row.Columns(&r.RetailerID, &r.SupplierID, &r.CreditLimitMinor, &r.CurrentBalanceMinor, &r.ReservedMinor,
			&r.AvailableCreditMinor, &r.RiskScore, &r.DelinquencyCount, &r.Status, &r.PrepayOnlyLocked, &lastEval, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return "", err
		}
		r.LastEvaluatedAt = optTimePtr(lastEval)
		r.CreatedAt = r.CreatedAt.UTC()
		r.UpdatedAt = r.UpdatedAt.UTC()
		rows = append(rows, r)
		return r.RetailerID, nil
	})
	return rows, err
}

func collectArLedgerEntries(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, eid string, to time.Time, batch int) ([]ArLedgerEntryRow, error) {
	var rows []ArLedgerEntryRow
	err := runKeysetPages(ctx, roTxn, batch, "ArLedgerEntries", func(has bool, lastID string) spanner.Statement {
		sql := `SELECT EntryId, InvoiceId, SupplierId, RetailerId, EntryType, AmountMinor, IdempotencyKey, COALESCE(RefOrderId, ''), CreatedAt
		        FROM ArLedgerEntries WHERE SupplierId = @eid AND CreatedAt <= @to`
		params := map[string]any{"eid": eid, "to": to.UTC()}
		if has {
			sql += ` AND EntryId > @k1`
			params["k1"] = lastID
		}
		sql += ` ORDER BY EntryId ASC LIMIT @lim`
		return spanner.Statement{SQL: sql, Params: params}
	}, func(row *spanner.Row) (string, error) {
		var r ArLedgerEntryRow
		if err := row.Columns(&r.EntryID, &r.InvoiceID, &r.SupplierID, &r.RetailerID, &r.EntryType, &r.AmountMinor, &r.IdempotencyKey, &r.RefOrderID, &r.CreatedAt); err != nil {
			return "", err
		}
		r.CreatedAt = r.CreatedAt.UTC()
		rows = append(rows, r)
		return r.EntryID, nil
	})
	return rows, err
}

type scoreHistKey struct {
	retailerID string
	scoreID    string
}

func collectCreditScoreHistory(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, eid string, retailerIDs []string, batch int) ([]CreditScoreHistoryRow, error) {
	var rows []CreditScoreHistoryRow
	seen := make(map[string]struct{})

	appendUnique := func(r CreditScoreHistoryRow) {
		k := r.RetailerID + "|" + r.ScoreID
		if _, dup := seen[k]; dup {
			return
		}
		seen[k] = struct{}{}
		rows = append(rows, r)
	}

	// 1. Relationship scores scoped to this supplier.
	err := runKeysetPages(ctx, roTxn, batch, "CreditScoreHistory(relationship)", func(has bool, k scoreHistKey) spanner.Statement {
		sql := `SELECT RetailerId, ScoreId, Scope, COALESCE(SupplierId, ''), ConsentVerified, COALESCE(ConsentScope, ''),
		               ModelVersion, ScorerSource, Score, PdBps, RiskTier, SuggestedLimitMinor, CurrentLimitMinor,
		               DisputesIsolatedCount, WindowStart, WindowEnd, ComputedAt
		        FROM CreditScoreHistory WHERE SupplierId = @eid`
		params := map[string]any{"eid": eid}
		if has {
			sql += ` AND (RetailerId > @k1 OR (RetailerId = @k1 AND ScoreId > @k2))`
			params["k1"] = k.retailerID
			params["k2"] = k.scoreID
		}
		sql += ` ORDER BY RetailerId ASC, ScoreId ASC LIMIT @lim`
		return spanner.Statement{SQL: sql, Params: params}
	}, func(row *spanner.Row) (scoreHistKey, error) {
		r, sErr := scanCreditScoreHistoryRow(row)
		if sErr != nil {
			return scoreHistKey{}, sErr
		}
		appendUnique(r)
		return scoreHistKey{retailerID: r.RetailerID, scoreID: r.ScoreID}, nil
	})
	if err != nil {
		return nil, err
	}

	// 2. Network scores for retailers in this supplier's credit book (batched by retailer_id).
	const ridChunkSize = 200
	for start := 0; start < len(retailerIDs); start += ridChunkSize {
		end := start + ridChunkSize
		if end > len(retailerIDs) {
			end = len(retailerIDs)
		}
		chunk := retailerIDs[start:end]
		iter := roTxn.Query(ctx, spanner.Statement{
			SQL: `SELECT RetailerId, ScoreId, Scope, COALESCE(SupplierId, ''), ConsentVerified, COALESCE(ConsentScope, ''),
			             ModelVersion, ScorerSource, Score, PdBps, RiskTier, SuggestedLimitMinor, CurrentLimitMinor,
			             DisputesIsolatedCount, WindowStart, WindowEnd, ComputedAt
			      FROM CreditScoreHistory
			      WHERE RetailerId IN UNNEST(@rids) AND Scope = 'NETWORK'
			      ORDER BY RetailerId ASC, ComputedAt DESC, ScoreId DESC`,
			Params: map[string]any{"rids": chunk},
		})
		for {
			row, qErr := iter.Next()
			if qErr == iterator.Done {
				break
			}
			if qErr != nil {
				iter.Stop()
				return nil, fmt.Errorf("collect CreditScoreHistory(network): %w", qErr)
			}
			r, sErr := scanCreditScoreHistoryRow(row)
			if sErr != nil {
				iter.Stop()
				return nil, fmt.Errorf("scan CreditScoreHistory(network): %w", sErr)
			}
			appendUnique(r)
		}
		iter.Stop()
	}

	return rows, nil
}

func scanCreditScoreHistoryRow(row *spanner.Row) (CreditScoreHistoryRow, error) {
	var r CreditScoreHistoryRow
	if err := row.Columns(
		&r.RetailerID, &r.ScoreID, &r.Scope, &r.SupplierID, &r.ConsentVerified, &r.ConsentScope,
		&r.ModelVersion, &r.ScorerSource, &r.Score, &r.PdBps, &r.RiskTier, &r.SuggestedLimitMinor,
		&r.CurrentLimitMinor, &r.DisputesIsolatedCount, &r.WindowStart, &r.WindowEnd, &r.ComputedAt,
	); err != nil {
		return CreditScoreHistoryRow{}, err
	}
	r.WindowStart = r.WindowStart.UTC()
	r.WindowEnd = r.WindowEnd.UTC()
	r.ComputedAt = r.ComputedAt.UTC()
	return r, nil
}

// ComputeBankCreditPack deterministically computes all 4 Bank Credit Tape schedules
// using integer int64 / math/big arithmetic (zero float64).
func ComputeBankCreditPack(pack *ExportPack, ds *PackDatasets) *BankCreditScheduleData {
	if ds == nil {
		ds = &PackDatasets{}
	}
	asOf := time.Now().UTC()
	if pack != nil && !pack.ToDate.IsZero() {
		asOf = pack.ToDate.UTC()
	}

	invoices := ds.BankArInvoices
	if len(invoices) == 0 {
		invoices = ds.ArInvoices
	}

	termsByRetailer := make(map[string]RetailerPaymentTermsRow, len(ds.RetailerPaymentTerms))
	for _, t := range ds.RetailerPaymentTerms {
		termsByRetailer[t.RetailerID] = t
	}
	profileByRetailer := make(map[string]RetailerCreditProfileRow, len(ds.RetailerCreditProfiles))
	for _, p := range ds.RetailerCreditProfiles {
		profileByRetailer[p.RetailerID] = p
	}

	relScoreByRetailer := make(map[string]CreditScoreHistoryRow)
	netScoreByRetailer := make(map[string]CreditScoreHistoryRow)
	for _, sc := range ds.CreditScoreHistory {
		scope := strings.ToUpper(strings.TrimSpace(sc.Scope))
		switch scope {
		case "RELATIONSHIP":
			if prev, ok := relScoreByRetailer[sc.RetailerID]; !ok || isNewerScore(sc, prev) {
				relScoreByRetailer[sc.RetailerID] = sc
			}
		case "NETWORK":
			if prev, ok := netScoreByRetailer[sc.RetailerID]; !ok || isNewerScore(sc, prev) {
				netScoreByRetailer[sc.RetailerID] = sc
			}
		}
	}

	type ledgerAgg struct {
		recoveredMinor  int64
		writtenOffMinor int64
		openedMinor     int64
	}
	ledgerByInvoice := make(map[string]ledgerAgg)
	for _, e := range ds.ArLedgerEntries {
		agg := ledgerByInvoice[e.InvoiceID]
		amt := absInt64(e.AmountMinor)
		switch strings.ToUpper(strings.TrimSpace(e.EntryType)) {
		case "PAYMENT", "ADVANCE_CREDIT":
			agg.recoveredMinor += amt
		case "WRITE_OFF", "VOID", "WAIVE":
			agg.writtenOffMinor += amt
		case "OPEN", "OPENING_BALANCE":
			agg.openedMinor += amt
		}
		ledgerByInvoice[e.InvoiceID] = agg
	}

	currency := "UZS"
	for _, inv := range invoices {
		if c := strings.TrimSpace(inv.Currency); c != "" {
			currency = strings.ToUpper(c)
			break
		}
	}

	sortedInvoices := append([]ArInvoiceRow(nil), invoices...)
	sort.Slice(sortedInvoices, func(i, j int) bool {
		if sortedInvoices[i].RetailerID != sortedInvoices[j].RetailerID {
			return sortedInvoices[i].RetailerID < sortedInvoices[j].RetailerID
		}
		return sortedInvoices[i].InvoiceID < sortedInvoices[j].InvoiceID
	})

	type bucketAcc struct {
		invoiceCount   int64
		retailers      map[string]struct{}
		principalMinor int64
		exposureMinor  int64
	}
	buckets := make(map[string]*bucketAcc, len(canonicalDPDBuckets))
	for _, b := range canonicalDPDBuckets {
		buckets[b] = &bucketAcc{retailers: make(map[string]struct{})}
	}

	type debtorAcc struct {
		retailerID       string
		invoiceCount     int64
		exposureMinor    int64
		principalMinor   int64
		creditLimitMinor int64
		maxDPDDays       int64
		consentVerified  bool
		relScore         int64
		netScore         int64
		currency         string
	}
	debtors := make(map[string]*debtorAcc)

	type vintageAcc struct {
		month                string
		invoiceCount         int64
		originationMinor     int64
		recoveredMinor       int64
		writtenOffMinor      int64
		outstandingMinor     int64
		defaultExposureMinor int64
		currency             string
	}
	vintages := make(map[string]*vintageAcc)

	retailerSeen := make(map[string]struct{})
	consentRetailers := make(map[string]struct{})
	tape := make([]BankCreditTapeRow, 0, len(sortedInvoices))

	var totalPrincipal, totalExposure, totalRecovered, totalWrittenOff int64

	for _, inv := range sortedInvoices {
		rid := strings.TrimSpace(inv.RetailerID)
		retailerSeen[rid] = struct{}{}
		terms, hasTerms := termsByRetailer[rid]
		prof, hasProf := profileByRetailer[rid]
		relSc, hasRel := relScoreByRetailer[rid]
		netSc, hasNet := netScoreByRetailer[rid]

		consent := (hasTerms && terms.ConsentVerified) || (hasRel && relSc.ConsentVerified) || (hasNet && netSc.ConsentVerified)
		if consent {
			consentRetailers[rid] = struct{}{}
		}

		limitMinor := int64(0)
		if hasProf && prof.CreditLimitMinor > 0 {
			limitMinor = prof.CreditLimitMinor
		} else if hasTerms && terms.CreditLimitMinor > 0 {
			limitMinor = terms.CreditLimitMinor
		}

		termsDays := inv.TermsDays
		if termsDays == 0 && hasTerms {
			termsDays = terms.TermsDays
		}
		graceDays := inv.GracePeriodDays
		if graceDays == 0 && hasTerms {
			graceDays = terms.GracePeriodDays
		}

		dpdDays := computeDPDDays(asOf, inv)
		bucket := dpdBucketName(dpdDays)

		la := ledgerByInvoice[inv.InvoiceID]
		recovered := la.recoveredMinor
		writtenOff := la.writtenOffMinor
		upperStatus := strings.ToUpper(strings.TrimSpace(inv.Status))
		if recovered == 0 && writtenOff == 0 && inv.PrincipalMinor > inv.BalanceMinor {
			if upperStatus == "VOID" || upperStatus == "WRITTEN_OFF" {
				writtenOff = inv.PrincipalMinor - inv.BalanceMinor
			} else {
				recovered = inv.PrincipalMinor - inv.BalanceMinor
			}
		}

		relScore := int64(0)
		relPdBps := int64(0)
		relTier := ""
		if hasRel {
			relScore = relSc.Score
			relPdBps = relSc.PdBps
			relTier = relSc.RiskTier
		} else if hasProf {
			relScore = prof.RiskScore
		}

		netScore := int64(0)
		netPdBps := int64(0)
		netTier := ""
		if consent && hasNet {
			netScore = netSc.Score
			netPdBps = netSc.PdBps
			netTier = netSc.RiskTier
		}

		invCurr := firstNonEmpty(strings.ToUpper(strings.TrimSpace(inv.Currency)), currency)
		utilBps := int64(0)
		if limitMinor > 0 {
			bal := inv.BalanceMinor
			if hasProf && prof.CurrentBalanceMinor > bal {
				bal = prof.CurrentBalanceMinor
			}
			utilBps = calcBps(bal, limitMinor)
		}

		tape = append(tape, BankCreditTapeRow{
			RetailerID:           rid,
			InvoiceID:            inv.InvoiceID,
			OrderID:              inv.OrderID,
			Status:               inv.Status,
			Currency:             invCurr,
			PrincipalMinor:       inv.PrincipalMinor,
			BalanceMinor:         inv.BalanceMinor,
			RecoveredMinor:       recovered,
			WrittenOffMinor:      writtenOff,
			CreditLimitMinor:     limitMinor,
			UtilizationBps:       utilBps,
			TermsDays:            termsDays,
			GracePeriodDays:      graceDays,
			DPDDays:              dpdDays,
			DPDBucket:            bucket,
			ConsentVerified:      consent,
			RelationshipScore:    relScore,
			RelationshipPdBps:    relPdBps,
			RelationshipRiskTier: relTier,
			NetworkScore:         netScore,
			NetworkPdBps:         netPdBps,
			NetworkRiskTier:      netTier,
			ProfileRiskScore:     prof.RiskScore,
			DelinquencyCount:     prof.DelinquencyCount,
			PrepayOnlyLocked:     prof.PrepayOnlyLocked,
			CreditLeaveAt:        inv.CreditLeaveAt.UTC().Format(time.RFC3339),
			DueAt:                inv.DueAt.UTC().Format(time.RFC3339),
		})

		totalPrincipal += inv.PrincipalMinor
		totalRecovered += recovered
		totalWrittenOff += writtenOff

		if isActiveExposureInvoice(inv) {
			totalExposure += inv.BalanceMinor
			bAcc := buckets[bucket]
			bAcc.invoiceCount++
			bAcc.retailers[rid] = struct{}{}
			bAcc.principalMinor += inv.PrincipalMinor
			bAcc.exposureMinor += inv.BalanceMinor

			dAcc, ok := debtors[rid]
			if !ok {
				dAcc = &debtorAcc{
					retailerID:       rid,
					creditLimitMinor: limitMinor,
					consentVerified:  consent,
					relScore:         relScore,
					netScore:         netScore,
					currency:         invCurr,
				}
				debtors[rid] = dAcc
			}
			dAcc.invoiceCount++
			dAcc.exposureMinor += inv.BalanceMinor
			dAcc.principalMinor += inv.PrincipalMinor
			if dpdDays > dAcc.maxDPDDays {
				dAcc.maxDPDDays = dpdDays
			}
		}

		vMonth := vintageMonthForInvoice(inv)
		vAcc, ok := vintages[vMonth]
		if !ok {
			vAcc = &vintageAcc{month: vMonth, currency: invCurr}
			vintages[vMonth] = vAcc
		}
		vAcc.invoiceCount++
		vAcc.originationMinor += inv.PrincipalMinor
		vAcc.recoveredMinor += recovered
		vAcc.writtenOffMinor += writtenOff
		if isActiveExposureInvoice(inv) {
			vAcc.outstandingMinor += inv.BalanceMinor
			if dpdDays > 90 {
				vAcc.defaultExposureMinor += inv.BalanceMinor
			}
		}
		vAcc.defaultExposureMinor += writtenOff
	}

	dpdRows := make([]DPDBucketRow, 0, len(canonicalDPDBuckets))
	for _, b := range canonicalDPDBuckets {
		acc := buckets[b]
		dpdRows = append(dpdRows, DPDBucketRow{
			Bucket:         b,
			InvoiceCount:   acc.invoiceCount,
			RetailerCount:  int64(len(acc.retailers)),
			PrincipalMinor: acc.principalMinor,
			ExposureMinor:  acc.exposureMinor,
			ShareBps:       calcBps(acc.exposureMinor, totalExposure),
			Currency:       currency,
		})
	}

	debtorList := make([]*debtorAcc, 0, len(debtors))
	for _, d := range debtors {
		debtorList = append(debtorList, d)
	}
	sort.Slice(debtorList, func(i, j int) bool {
		if debtorList[i].exposureMinor != debtorList[j].exposureMinor {
			return debtorList[i].exposureMinor > debtorList[j].exposureMinor
		}
		return debtorList[i].retailerID < debtorList[j].retailerID
	})

	concRows := make([]DebtorConcentrationRow, 0, len(debtorList))
	var cumulativeExposure int64
	var top1Bps, top5Bps, top10Bps int64
	for idx, d := range debtorList {
		rank := int64(idx + 1)
		cumulativeExposure += d.exposureMinor
		cBps := calcBps(d.exposureMinor, totalExposure)
		cumBps := calcBps(cumulativeExposure, totalExposure)
		concRows = append(concRows, DebtorConcentrationRow{
			Rank:              rank,
			RetailerID:        d.retailerID,
			InvoiceCount:      d.invoiceCount,
			ExposureMinor:     d.exposureMinor,
			PrincipalMinor:    d.principalMinor,
			CreditLimitMinor:  d.creditLimitMinor,
			ConcentrationBps:  cBps,
			CumulativeBps:     cumBps,
			MaxDPDDays:        d.maxDPDDays,
			ConsentVerified:   d.consentVerified,
			RelationshipScore: d.relScore,
			NetworkScore:      d.netScore,
			Currency:          d.currency,
		})
		if rank == 1 {
			top1Bps = cumBps
		}
		if rank <= 5 {
			top5Bps = cumBps
		}
		if rank <= 10 {
			top10Bps = cumBps
		}
	}

	vintageKeys := make([]string, 0, len(vintages))
	for k := range vintages {
		vintageKeys = append(vintageKeys, k)
	}
	sort.Strings(vintageKeys)
	curveRows := make([]LossRecoveryCurveRow, 0, len(vintageKeys))
	for _, k := range vintageKeys {
		v := vintages[k]
		curveRows = append(curveRows, LossRecoveryCurveRow{
			VintageMonth:         v.month,
			InvoiceCount:         v.invoiceCount,
			OriginationMinor:     v.originationMinor,
			RecoveredMinor:       v.recoveredMinor,
			WrittenOffMinor:      v.writtenOffMinor,
			OutstandingMinor:     v.outstandingMinor,
			DefaultExposureMinor: v.defaultExposureMinor,
			RecoveryRateBps:      calcBps(v.recoveredMinor, v.originationMinor),
			LossRateBps:          calcBps(v.writtenOffMinor, v.originationMinor),
			DefaultRateBps:       calcBps(v.defaultExposureMinor, v.originationMinor),
			Currency:             v.currency,
		})
	}

	// Score averages across all retailers with scores.
	var relSum, relCount, netSum, netCount int64
	for rid := range retailerSeen {
		if sc, ok := relScoreByRetailer[rid]; ok && sc.Score > 0 {
			relSum += sc.Score
			relCount++
		} else if p, ok := profileByRetailer[rid]; ok && p.RiskScore > 0 {
			relSum += p.RiskScore
			relCount++
		}
		if _, consented := consentRetailers[rid]; consented {
			if sc, ok := netScoreByRetailer[rid]; ok && sc.Score > 0 {
				netSum += sc.Score
				netCount++
			}
		}
	}
	var avgRel, avgNet int64
	if relCount > 0 {
		avgRel = relSum / relCount
	}
	if netCount > 0 {
		avgNet = netSum / netCount
	}

	return &BankCreditScheduleData{
		Summary: BankCreditSummary{
			AsOf:                     asOf,
			Currency:                 currency,
			TotalInvoices:            int64(len(tape)),
			TotalRetailers:           int64(len(retailerSeen)),
			ConsentVerifiedRetailers: int64(len(consentRetailers)),
			TotalPrincipalMinor:      totalPrincipal,
			TotalExposureMinor:       totalExposure,
			TotalRecoveredMinor:      totalRecovered,
			TotalWrittenOffMinor:     totalWrittenOff,
			Top1ConcentrationBps:     top1Bps,
			Top5ConcentrationBps:     top5Bps,
			Top10ConcentrationBps:    top10Bps,
			AvgRelationshipScore:     avgRel,
			AvgNetworkScore:          avgNet,
			DPDBuckets:               dpdRows,
			DebtorConcentration:      concRows,
			LossRecoveryCurves:       curveRows,
		},
		Tape:               tape,
		DPDBuckets:         dpdRows,
		Concentration:      concRows,
		LossRecoveryCurves: curveRows,
	}
}

func bankTableSpecs(data *BankCreditScheduleData) []tableSpec {
	if data == nil {
		return nil
	}
	tapeRows := make([][]string, len(data.Tape))
	for i, r := range data.Tape {
		tapeRows[i] = []string{
			r.RetailerID, r.InvoiceID, r.OrderID, r.Status, r.Currency,
			i64(r.PrincipalMinor), i64(r.BalanceMinor), i64(r.RecoveredMinor), i64(r.WrittenOffMinor),
			i64(r.CreditLimitMinor), i64(r.UtilizationBps), i64(r.TermsDays), i64(r.GracePeriodDays),
			i64(r.DPDDays), r.DPDBucket, strconv.FormatBool(r.ConsentVerified),
			i64(r.RelationshipScore), i64(r.RelationshipPdBps), r.RelationshipRiskTier,
			i64(r.NetworkScore), i64(r.NetworkPdBps), r.NetworkRiskTier,
			i64(r.ProfileRiskScore), i64(r.DelinquencyCount), strconv.FormatBool(r.PrepayOnlyLocked),
			r.CreditLeaveAt, r.DueAt,
		}
	}

	dpdRows := make([][]string, len(data.DPDBuckets))
	for i, r := range data.DPDBuckets {
		dpdRows[i] = []string{
			r.Bucket, i64(r.InvoiceCount), i64(r.RetailerCount),
			i64(r.PrincipalMinor), i64(r.ExposureMinor), i64(r.ShareBps), r.Currency,
		}
	}

	concRows := make([][]string, len(data.Concentration))
	for i, r := range data.Concentration {
		concRows[i] = []string{
			i64(r.Rank), r.RetailerID, i64(r.InvoiceCount),
			i64(r.ExposureMinor), i64(r.PrincipalMinor), i64(r.CreditLimitMinor),
			i64(r.ConcentrationBps), i64(r.CumulativeBps), i64(r.MaxDPDDays),
			strconv.FormatBool(r.ConsentVerified), i64(r.RelationshipScore), i64(r.NetworkScore), r.Currency,
		}
	}

	curveRows := make([][]string, len(data.LossRecoveryCurves))
	for i, r := range data.LossRecoveryCurves {
		curveRows[i] = []string{
			r.VintageMonth, i64(r.InvoiceCount),
			i64(r.OriginationMinor), i64(r.RecoveredMinor), i64(r.WrittenOffMinor),
			i64(r.OutstandingMinor), i64(r.DefaultExposureMinor),
			i64(r.RecoveryRateBps), i64(r.LossRateBps), i64(r.DefaultRateBps), r.Currency,
		}
	}

	return []tableSpec{
		{
			name:      "bank_credit_tape",
			sheetName: "Bank_Credit_Tape",
			csvPath:   "csv/bank_credit_tape.csv",
			headers: []string{
				"retailer_id", "invoice_id", "order_id", "status", "currency",
				"principal_minor", "balance_minor", "recovered_minor", "written_off_minor",
				"credit_limit_minor", "utilization_bps", "terms_days", "grace_period_days",
				"dpd_days", "dpd_bucket", "consent_verified",
				"relationship_score", "relationship_pd_bps", "relationship_risk_tier",
				"network_score", "network_pd_bps", "network_risk_tier",
				"profile_risk_score", "delinquency_count", "prepay_only_locked",
				"credit_leave_at", "due_at",
			},
			rows: tapeRows,
		},
		{
			name:      "dpd_buckets",
			sheetName: "DPD_Buckets",
			csvPath:   "csv/dpd_buckets.csv",
			headers:   []string{"bucket", "invoice_count", "retailer_count", "principal_minor", "exposure_minor", "share_bps", "currency"},
			rows:      dpdRows,
		},
		{
			name:      "debtor_concentration",
			sheetName: "Debtor_Concentration",
			csvPath:   "csv/debtor_concentration.csv",
			headers: []string{
				"rank", "retailer_id", "invoice_count", "exposure_minor", "principal_minor",
				"credit_limit_minor", "concentration_bps", "cumulative_bps", "max_dpd_days",
				"consent_verified", "relationship_score", "network_score", "currency",
			},
			rows: concRows,
		},
		{
			name:      "loss_recovery_curves",
			sheetName: "Loss_LossRecovery_Curves",
			csvPath:   "csv/loss_recovery_curves.csv",
			headers: []string{
				"vintage_month", "invoice_count", "origination_minor", "recovered_minor",
				"written_off_minor", "outstanding_minor", "default_exposure_minor",
				"recovery_rate_bps", "loss_rate_bps", "default_rate_bps", "currency",
			},
			rows: curveRows,
		},
	}
}

func bankCombinedNormalizedRows(entityID, watermark string, data *BankCreditScheduleData) [][]string {
	if data == nil {
		return nil
	}
	asOfStr := data.Summary.AsOf.UTC().Format(time.RFC3339)
	var out [][]string
	for _, r := range data.Tape {
		out = append(out, []string{"bank_credit_tape", r.InvoiceID, r.DueAt, entityID, r.RetailerID, r.DPDBucket, i64(r.BalanceMinor), r.Currency, jsonString(r), watermark})
	}
	for _, r := range data.DPDBuckets {
		out = append(out, []string{"dpd_buckets", r.Bucket, asOfStr, entityID, r.Bucket, i64(r.ShareBps), i64(r.ExposureMinor), r.Currency, jsonString(r), watermark})
	}
	for _, r := range data.Concentration {
		out = append(out, []string{"debtor_concentration", r.RetailerID, asOfStr, entityID, i64(r.Rank), i64(r.ConcentrationBps), i64(r.ExposureMinor), r.Currency, jsonString(r), watermark})
	}
	for _, r := range data.LossRecoveryCurves {
		out = append(out, []string{"loss_recovery_curves", r.VintageMonth, asOfStr, entityID, r.VintageMonth, i64(r.RecoveryRateBps), i64(r.OriginationMinor), r.Currency, jsonString(r), watermark})
	}
	return out
}

func isNewerScore(cand, prev CreditScoreHistoryRow) bool {
	if !cand.ComputedAt.Equal(prev.ComputedAt) {
		return cand.ComputedAt.After(prev.ComputedAt)
	}
	return cand.ScoreID > prev.ScoreID
}

func isActiveExposureInvoice(inv ArInvoiceRow) bool {
	if inv.BalanceMinor <= 0 {
		return false
	}
	switch strings.ToUpper(strings.TrimSpace(inv.Status)) {
	case "PAID", "VOID", "WRITTEN_OFF", "CANCELLED":
		return false
	default:
		return true
	}
}

func computeDPDDays(asOf time.Time, inv ArInvoiceRow) int64 {
	if !isActiveExposureInvoice(inv) || inv.DueAt.IsZero() {
		return 0
	}
	asOfDay := time.Date(asOf.UTC().Year(), asOf.UTC().Month(), asOf.UTC().Day(), 0, 0, 0, 0, time.UTC)
	dueDay := time.Date(inv.DueAt.UTC().Year(), inv.DueAt.UTC().Month(), inv.DueAt.UTC().Day(), 0, 0, 0, 0, time.UTC)
	if !asOfDay.After(dueDay) {
		return 0
	}
	return int64(asOfDay.Sub(dueDay) / (24 * time.Hour))
}

func dpdBucketName(dpdDays int64) string {
	switch {
	case dpdDays <= 30:
		return "0-30"
	case dpdDays <= 60:
		return "31-60"
	case dpdDays <= 90:
		return "61-90"
	default:
		return "90+"
	}
}

func vintageMonthForInvoice(inv ArInvoiceRow) string {
	t := inv.CreditLeaveAt
	if t.IsZero() {
		t = inv.CreatedAt
	}
	if t.IsZero() {
		return "unknown"
	}
	return t.UTC().Format("2006-01")
}

// calcBps computes integer basis points (out of 10000) using arbitrary-precision integers (zero float64).
func calcBps(numerator, denominator int64) int64 {
	if denominator <= 0 || numerator == 0 {
		return 0
	}
	neg := false
	if numerator < 0 {
		neg = true
		numerator = -numerator
	}
	num := new(big.Int).Mul(big.NewInt(numerator), big.NewInt(10000))
	den := big.NewInt(denominator)
	half := new(big.Int).Quo(den, big.NewInt(2))
	num.Add(num, half)
	q := new(big.Int).Quo(num, den)
	res := q.Int64()
	if neg {
		res = -res
	}
	return res
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
