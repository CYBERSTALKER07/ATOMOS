package p2p

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ShippedUZTemplates returns the four built-in Uzbekistan ERP process templates:
// O2C-DIRECT-UZ, O2C-PREORDER-UZ, P2P-STD-UZ, and RETURNS-SUPPLIER-UZ.
func ShippedUZTemplates(tenantID string, now time.Time) []ProcessTemplate {
	if now.IsZero() {
		now = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	now = now.UTC()
	tid := strings.TrimSpace(tenantID)

	return []ProcessTemplate{
		{
			TenantID:   tid,
			TemplateID: "tpl-uz-o2c-direct",
			Code:       TemplateO2CDirectUZ,
			Name:       "Uzbekistan Direct Order-to-Cash (O2C)",
			Domain:     DomainO2C,
			IsDefault:  true,
			BuiltIn:    true,
			Status:     TemplateStatusActive,
			CreatedAt:  now,
			UpdatedAt:  now,
			Steps: []ProcessStep{
				{Code: "CREATED", Name: "Order Captured", Sequence: 1, SLASeconds: 900, RequiredRole: "SALES_MANAGER"},
				{Code: "CONFIRMED", Name: "Order Confirmed", Sequence: 2, SLASeconds: 3600, RequiredRole: "DISPATCHER"},
				{Code: "ALLOCATED", Name: "Stock Allocated", Sequence: 3, SLASeconds: 7200, RequiredRole: "WAREHOUSE_MANAGER"},
				{Code: "PICKED", Name: "Warehouse Picked", Sequence: 4, SLASeconds: 14400, RequiredRole: "DISPATCHER"},
				{Code: "DISPATCHED", Name: "Dispatched to Fleet", Sequence: 5, SLASeconds: 28800, RequiredRole: "DISPATCHER"},
				{Code: "DELIVERED", Name: "Delivered & POD Signed", Sequence: 6, SLASeconds: 86400, RequiredRole: "DRIVER"},
				{Code: "INVOICED", Name: "Outbound ESF / Fiscalized", Sequence: 7, SLASeconds: 3600, RequiredRole: "ACCOUNTANT"},
				{Code: "SETTLED", Name: "AR Settled & Reconciled", Sequence: 8, SLASeconds: 604800, RequiredRole: "FINANCE_MANAGER"},
			},
		},
		{
			TenantID:   tid,
			TemplateID: "tpl-uz-o2c-preorder",
			Code:       TemplateO2CPreorderUZ,
			Name:       "Uzbekistan Pre-Order Fulfillment (O2C)",
			Domain:     DomainO2C,
			IsDefault:  false,
			BuiltIn:    true,
			Status:     TemplateStatusActive,
			CreatedAt:  now,
			UpdatedAt:  now,
			Steps: []ProcessStep{
				{Code: "PREORDER_CAPTURED", Name: "Pre-Order Captured", Sequence: 1, SLASeconds: 3600, RequiredRole: "SALES_MANAGER"},
				{Code: "CONFIRMATION_REQUESTED", Name: "Buyer Confirmation Requested", Sequence: 2, SLASeconds: 21600, RequiredRole: "SALES_MANAGER"},
				{Code: "CONFIRMED", Name: "Pre-Order Confirmed", Sequence: 3, SLASeconds: 43200, RequiredRole: "SALES_MANAGER"},
				{Code: "PRODUCTION_RESERVED", Name: "Production / Lot Reserved", Sequence: 4, SLASeconds: 86400, RequiredRole: "WAREHOUSE_MANAGER"},
				{Code: "ALLOCATED", Name: "Stock Allocated", Sequence: 5, SLASeconds: 14400, RequiredRole: "WAREHOUSE_MANAGER"},
				{Code: "DISPATCHED", Name: "Dispatched to Fleet", Sequence: 6, SLASeconds: 28800, RequiredRole: "DISPATCHER"},
				{Code: "DELIVERED", Name: "Delivered & POD Signed", Sequence: 7, SLASeconds: 86400, RequiredRole: "DRIVER"},
				{Code: "SETTLED", Name: "AR Settled", Sequence: 8, SLASeconds: 604800, RequiredRole: "FINANCE_MANAGER"},
			},
		},
		{
			TenantID:   tid,
			TemplateID: "tpl-uz-p2p-std",
			Code:       TemplateP2PStdUZ,
			Name:       "Uzbekistan Standard Procure-to-Pay (P2P + 3-Way Match)",
			Domain:     DomainP2P,
			IsDefault:  true,
			BuiltIn:    true,
			Status:     TemplateStatusActive,
			CreatedAt:  now,
			UpdatedAt:  now,
			Steps: []ProcessStep{
				{Code: "PO_CREATED", Name: "Purchase Order Created", Sequence: 1, SLASeconds: 3600, RequiredRole: "ACCOUNTANT"},
				{Code: "PO_APPROVED", Name: "Purchase Order Approved", Sequence: 2, SLASeconds: 14400, RequiredRole: "FINANCE_MANAGER"},
				{Code: "GOODS_RECEIVED", Name: "Goods Receipt Posted (DR INVENTORY / CR GRNI)", Sequence: 3, SLASeconds: 86400, RequiredRole: "WAREHOUSE_MANAGER"},
				{Code: "INVOICE_RECEIVED", Name: "Vendor Invoice / Inbound ESF Recorded", Sequence: 4, SLASeconds: 43200, RequiredRole: "ACCOUNTANT"},
				{Code: "MATCHED_3WAY", Name: "3-Way Match Verified & AP Posted", Sequence: 5, SLASeconds: 14400, RequiredRole: "FINANCE_MANAGER"},
			},
		},
		{
			TenantID:   tid,
			TemplateID: "tpl-uz-returns-supplier",
			Code:       TemplateReturnsSupplierUZ,
			Name:       "Uzbekistan Supplier Return-to-Vendor (RTV)",
			Domain:     DomainReturns,
			IsDefault:  true,
			BuiltIn:    true,
			Status:     TemplateStatusActive,
			CreatedAt:  now,
			UpdatedAt:  now,
			Steps: []ProcessStep{
				{Code: "RETURN_REQUESTED", Name: "Return-to-Vendor Requested", Sequence: 1, SLASeconds: 3600, RequiredRole: "WAREHOUSE_MANAGER"},
				{Code: "RTV_APPROVED", Name: "RTV Approved", Sequence: 2, SLASeconds: 14400, RequiredRole: "FINANCE_MANAGER"},
				{Code: "RTV_DISPATCHED", Name: "RTV Goods Dispatched", Sequence: 3, SLASeconds: 43200, RequiredRole: "DISPATCHER"},
				{Code: "CREDIT_NOTE_POSTED", Name: "Supplier Credit Note & VAT Reversal Posted", Sequence: 4, SLASeconds: 86400, RequiredRole: "ACCOUNTANT"},
				{Code: "CLOSED", Name: "RTV Case Closed", Sequence: 5, SLASeconds: 86400, RequiredRole: "FINANCE_MANAGER"},
			},
		},
	}
}

// MergeTemplates combines tenant-saved templates with built-in shipped UZ templates
// (tenant overrides take precedence by Code).
func MergeTemplates(tenantID string, custom []ProcessTemplate, domainFilter string) []ProcessTemplate {
	df := normalizeDomain(domainFilter)
	byCode := make(map[string]ProcessTemplate)
	for _, tpl := range ShippedUZTemplates(tenantID, time.Time{}) {
		byCode[strings.ToUpper(tpl.Code)] = tpl
	}
	for _, tpl := range custom {
		if strings.TrimSpace(tpl.Code) == "" {
			continue
		}
		byCode[strings.ToUpper(strings.TrimSpace(tpl.Code))] = tpl
	}

	out := make([]ProcessTemplate, 0, len(byCode))
	for _, tpl := range byCode {
		if df != "" && !strings.EqualFold(tpl.Domain, df) {
			continue
		}
		out = append(out, tpl)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Domain != out[j].Domain {
			return out[i].Domain < out[j].Domain
		}
		if out[i].IsDefault != out[j].IsDefault {
			return out[i].IsDefault
		}
		return out[i].Code < out[j].Code
	})
	return out
}

// ResolveTemplate looks up a template by templateID or templateCode across merged templates.
func ResolveTemplate(templates []ProcessTemplate, templateID, templateCode, defaultDomain string) (ProcessTemplate, bool) {
	tid := strings.TrimSpace(templateID)
	tcode := strings.ToUpper(strings.TrimSpace(templateCode))
	dom := normalizeDomain(defaultDomain)

	for _, tpl := range templates {
		if tid != "" && tpl.TemplateID == tid {
			return tpl, true
		}
	}
	for _, tpl := range templates {
		if tcode != "" && strings.EqualFold(tpl.Code, tcode) {
			return tpl, true
		}
	}
	if tid == "" && tcode == "" && dom != "" {
		for _, tpl := range templates {
			if strings.EqualFold(tpl.Domain, dom) && tpl.IsDefault {
				return tpl, true
			}
		}
		for _, tpl := range templates {
			if strings.EqualFold(tpl.Domain, dom) {
				return tpl, true
			}
		}
	}
	return ProcessTemplate{}, false
}

// LookupStepSLASeconds finds the SLA target in int64 seconds for toStep within domain.
func LookupStepSLASeconds(templates []ProcessTemplate, domain, toStep string) int64 {
	target := normalizeStepName(toStep)
	dom := normalizeDomain(domain)
	for _, tpl := range templates {
		if dom != "" && !strings.EqualFold(tpl.Domain, dom) {
			continue
		}
		for _, st := range tpl.Steps {
			if strings.EqualFold(st.Code, target) && st.SLASeconds > 0 {
				return st.SLASeconds
			}
		}
	}
	switch target {
	case "CONFIRMED", "PO_CREATED", "RETURN_REQUESTED", "INVOICED":
		return 3600
	case "ALLOCATED", "PICKED", "PO_APPROVED", "MATCHED_3WAY", "RTV_APPROVED":
		return 14400
	case "DISPATCHED", "IN_TRANSIT":
		return 28800
	case "INVOICE_RECEIVED", "RTV_DISPATCHED":
		return 43200
	case "DELIVERED", "GOODS_RECEIVED", "COMPLETED":
		return 86400
	default:
		return DefaultMiningSLASec
	}
}

func normalizeStepName(raw string) string {
	s := strings.ToUpper(strings.TrimSpace(raw))
	switch s {
	case "":
		return "START"
	case "PENDING":
		return "CREATED"
	case "IN_TRANSIT":
		return "DISPATCHED"
	case "COMPLETED":
		return "DELIVERED"
	default:
		return s
	}
}

// AggregateMiningSamples computes exact integer process mining metrics from raw transition samples.
// Never fabricates metrics when samples is empty.
func AggregateMiningSamples(tenantID, entityID, domain string, samples []MiningSample) ProcessMiningReport {
	report := ProcessMiningReport{
		TenantID: strings.TrimSpace(tenantID),
		EntityID: strings.TrimSpace(entityID),
		Domain:   normalizeDomain(domain),
		Edges:    []MiningEdge{},
	}
	if len(samples) == 0 {
		report.Empty = true
		return report
	}

	type edgeKey struct {
		domain   string
		fromStep string
		toStep   string
	}
	type edgeAcc struct {
		cases     map[string]struct{}
		durations []int64
		slaTarget int64
		breaches  int64
	}

	allCases := make(map[string]struct{})
	byEdge := make(map[edgeKey]*edgeAcc)

	for _, sm := range samples {
		dom := normalizeDomain(sm.Domain)
		if dom == "" {
			dom = DomainO2C
		}
		if report.Domain != "" && !strings.EqualFold(dom, report.Domain) {
			continue
		}
		from := strings.ToUpper(strings.TrimSpace(sm.FromStep))
		if from == "" {
			from = "START"
		}
		to := strings.ToUpper(strings.TrimSpace(sm.ToStep))
		if to == "" {
			continue
		}
		dur := sm.DurationSec
		if dur < 0 {
			dur = 0
		}
		sla := sm.SLATarget
		if sla <= 0 {
			sla = DefaultMiningSLASec
		}

		if sm.CaseID != "" {
			allCases[dom+":"+sm.CaseID] = struct{}{}
		}

		k := edgeKey{domain: dom, fromStep: from, toStep: to}
		acc, ok := byEdge[k]
		if !ok {
			acc = &edgeAcc{
				cases:     make(map[string]struct{}),
				slaTarget: sla,
			}
			byEdge[k] = acc
		}
		if sm.CaseID != "" {
			acc.cases[sm.CaseID] = struct{}{}
		}
		acc.durations = append(acc.durations, dur)
		if dur > acc.slaTarget {
			acc.breaches++
		}
	}

	if len(byEdge) == 0 {
		report.Empty = true
		return report
	}

	edges := make([]MiningEdge, 0, len(byEdge))
	var totalTransitions int64
	var totalBreaches int64
	var top *MiningBottleneck

	for k, acc := range byEdge {
		sort.Slice(acc.durations, func(i, j int) bool { return acc.durations[i] < acc.durations[j] })
		n := int64(len(acc.durations))
		var sum int64
		for _, d := range acc.durations {
			sum += d
		}
		avg := sum / n
		med := integerMedian(acc.durations)
		p95 := integerPercentile(acc.durations, 95)

		caseCnt := int64(len(acc.cases))
		if caseCnt == 0 {
			caseCnt = n
		}
		edge := MiningEdge{
			FromStep:          k.fromStep,
			ToStep:            k.toStep,
			Domain:            k.domain,
			CaseCount:         caseCnt,
			TransitionCount:   n,
			AvgDurationSec:    avg,
			MedianDurationSec: med,
			P95DurationSec:    p95,
			SLATargetSec:      acc.slaTarget,
			SLABreachCount:    acc.breaches,
		}
		edges = append(edges, edge)
		totalTransitions += n
		totalBreaches += acc.breaches

		if top == nil || isWorseBottleneck(edge, *top) {
			cp := MiningBottleneck{
				FromStep:          edge.FromStep,
				ToStep:            edge.ToStep,
				Domain:            edge.Domain,
				CaseCount:         edge.CaseCount,
				TransitionCount:   edge.TransitionCount,
				MedianDurationSec: edge.MedianDurationSec,
				P95DurationSec:    edge.P95DurationSec,
				SLATargetSec:      edge.SLATargetSec,
				SLABreachCount:    edge.SLABreachCount,
			}
			top = &cp
		}
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Domain != edges[j].Domain {
			return edges[i].Domain < edges[j].Domain
		}
		if edges[i].P95DurationSec != edges[j].P95DurationSec {
			return edges[i].P95DurationSec > edges[j].P95DurationSec
		}
		if edges[i].FromStep != edges[j].FromStep {
			return edges[i].FromStep < edges[j].FromStep
		}
		return edges[i].ToStep < edges[j].ToStep
	})

	report.TotalCases = int64(len(allCases))
	if report.TotalCases == 0 {
		report.TotalCases = totalTransitions
	}
	report.TotalTransitions = totalTransitions
	report.SLABreachCount = totalBreaches
	report.BottleneckStage = top
	report.Edges = edges
	report.Empty = false
	return report
}

func isWorseBottleneck(cand MiningEdge, cur MiningBottleneck) bool {
	if (cand.SLABreachCount > 0) != (cur.SLABreachCount > 0) {
		return cand.SLABreachCount > 0
	}
	if cand.SLABreachCount != cur.SLABreachCount {
		return cand.SLABreachCount > cur.SLABreachCount
	}
	if cand.P95DurationSec != cur.P95DurationSec {
		return cand.P95DurationSec > cur.P95DurationSec
	}
	if cand.MedianDurationSec != cur.MedianDurationSec {
		return cand.MedianDurationSec > cur.MedianDurationSec
	}
	if cand.TransitionCount != cur.TransitionCount {
		return cand.TransitionCount > cur.TransitionCount
	}
	return fmt.Sprintf("%s->%s", cand.FromStep, cand.ToStep) < fmt.Sprintf("%s->%s", cur.FromStep, cur.ToStep)
}

func integerMedian(sorted []int64) int64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

func integerPercentile(sorted []int64, pct int64) int64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	idx := (pct*int64(n)+99)/100 - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= int64(n) {
		idx = int64(n - 1)
	}
	return sorted[idx]
}
