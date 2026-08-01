package retailer

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestPhase6SectionsReportsAssist(t *testing.T) {
	t.Parallel()
	n := 0
	svc := NewService(ServiceConfig{
		Now: time.Now,
		NewID: func() string {
			n++
			return "p6-" + string(rune('A'+n%26)) + string(rune('0'+n/26))
		},
	})
	primary, err := svc.EnsurePrimaryLocation(t.Context(), "ret-p6")
	if err != nil {
		t.Fatal(err)
	}
	owner := auth.Claims{
		Subject: "owner", Role: auth.RoleRetailer, RetailerOrgID: "ret-p6",
		RetailerRole: "OWNER", RetailerUserID: "owner",
	}
	lead := auth.Claims{
		Subject: "lead", Role: auth.RoleRetailer, RetailerOrgID: "ret-p6",
		RetailerRole: "SECTION_LEAD", RetailerUserID: "lead",
	}

	// Seed stock for unassigned pool
	_ = svc.applyDelta(t.Context(), "ret-p6", primary.LocationID, BinFloor, "SKU-A", 10, MoveReceive, "TEST", "o1", "owner", "")
	_ = svc.applyDelta(t.Context(), "ret-p6", primary.LocationID, BinFloor, "SKU-B", 3, MoveReceive, "TEST", "o1", "owner", "")

	// Create section
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/sections",
		bytes.NewBufferString(`{"name":"Dairy","aisle_tag":"A1","location_id":"`+primary.LocationID+`"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), owner))
	rr := httptest.NewRecorder()
	svc.HandleSections(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create section status=%d body=%s", rr.Code, rr.Body.String())
	}
	var sec SectionDTO
	_ = json.Unmarshal(rr.Body.Bytes(), &sec)
	if sec.SectionID == "" || sec.Name != "Dairy" {
		t.Fatalf("section=%+v", sec)
	}
	// Packs auto-enabled
	enabled, _ := svc.LoadEnabledPacks(t.Context(), "ret-p6")
	if !enabled.Has(PackSECTIONS) || !enabled.Has(PackSTORESTOCK) {
		t.Fatalf("expected SECTIONS+STORE_STOCK, got %v", enabled.List())
	}

	// Map SKUs
	reqSkus := httptest.NewRequest(http.MethodPut, "/v1/retailer/sections/"+sec.SectionID+"/skus",
		bytes.NewBufferString(`{"skus":["SKU-A"]}`))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sectionID", sec.SectionID)
	reqSkus = reqSkus.WithContext(auth.WithClaims(reqSkus.Context(), owner))
	reqSkus = reqSkus.WithContext(contextWithChi(reqSkus, rctx))
	rrSkus := httptest.NewRecorder()
	svc.HandleSectionSkus(rrSkus, reqSkus)
	if rrSkus.Code != http.StatusOK {
		t.Fatalf("skus status=%d body=%s", rrSkus.Code, rrSkus.Body.String())
	}

	// Unassigned should include SKU-B
	reqUn := httptest.NewRequest(http.MethodGet, "/v1/retailer/sections/unassigned-skus?location_id="+primary.LocationID, nil)
	reqUn = reqUn.WithContext(auth.WithClaims(reqUn.Context(), owner))
	rrUn := httptest.NewRecorder()
	svc.HandleUnassignedSkus(rrUn, reqUn)
	if rrUn.Code != http.StatusOK {
		t.Fatalf("unassigned status=%d", rrUn.Code)
	}
	var un map[string]any
	_ = json.Unmarshal(rrUn.Body.Bytes(), &un)
	skusRaw, _ := json.Marshal(un["skus"])
	if !strings.Contains(string(skusRaw), "SKU-B") {
		t.Fatalf("expected SKU-B unassigned, got %s", skusRaw)
	}
	if strings.Contains(string(skusRaw), "SKU-A") {
		t.Fatalf("SKU-A should be assigned, got %s", skusRaw)
	}

	// Staff assign
	reqStaff := httptest.NewRequest(http.MethodPut, "/v1/retailer/sections/"+sec.SectionID+"/staff",
		bytes.NewBufferString(`{"user_ids":["lead"]}`))
	rctx2 := chi.NewRouteContext()
	rctx2.URLParams.Add("sectionID", sec.SectionID)
	reqStaff = reqStaff.WithContext(auth.WithClaims(reqStaff.Context(), owner))
	reqStaff = reqStaff.WithContext(contextWithChi(reqStaff, rctx2))
	rrStaff := httptest.NewRecorder()
	svc.HandleSectionStaff(rrStaff, reqStaff)
	if rrStaff.Code != http.StatusOK {
		t.Fatalf("staff status=%d body=%s", rrStaff.Code, rrStaff.Body.String())
	}

	// Assist ticket lifecycle
	reqT := httptest.NewRequest(http.MethodPost, "/v1/retailer/assist/tickets",
		bytes.NewBufferString(`{"section_id":"`+sec.SectionID+`","note":"Need help at dairy"}`))
	reqT = reqT.WithContext(auth.WithClaims(reqT.Context(), owner))
	rrT := httptest.NewRecorder()
	svc.HandleAssistTickets(rrT, reqT)
	if rrT.Code != http.StatusCreated {
		t.Fatalf("ticket create status=%d body=%s", rrT.Code, rrT.Body.String())
	}
	var ticket AssistTicketDTO
	_ = json.Unmarshal(rrT.Body.Bytes(), &ticket)
	if ticket.Status != AssistOpen {
		t.Fatalf("ticket=%+v", ticket)
	}
	enabled, _ = svc.LoadEnabledPacks(t.Context(), "ret-p6")
	if !enabled.Has(PackCUSTOMERASSIST) {
		t.Fatal("expected CUSTOMER_ASSIST enabled")
	}

	// Claim by section lead
	reqClaim := httptest.NewRequest(http.MethodPost, "/v1/retailer/assist/tickets/"+ticket.TicketID+"/claim", nil)
	rctx3 := chi.NewRouteContext()
	rctx3.URLParams.Add("ticketID", ticket.TicketID)
	reqClaim = reqClaim.WithContext(auth.WithClaims(reqClaim.Context(), lead))
	reqClaim = reqClaim.WithContext(contextWithChi(reqClaim, rctx3))
	rrClaim := httptest.NewRecorder()
	svc.HandleAssistClaim(rrClaim, reqClaim)
	if rrClaim.Code != http.StatusOK {
		t.Fatalf("claim status=%d body=%s", rrClaim.Code, rrClaim.Body.String())
	}
	_ = json.Unmarshal(rrClaim.Body.Bytes(), &ticket)
	if ticket.Status != AssistClaimed || ticket.ClaimedByUserID != "lead" {
		t.Fatalf("claimed=%+v", ticket)
	}

	// Complete
	reqDone := httptest.NewRequest(http.MethodPost, "/v1/retailer/assist/tickets/"+ticket.TicketID+"/complete", nil)
	rctx4 := chi.NewRouteContext()
	rctx4.URLParams.Add("ticketID", ticket.TicketID)
	reqDone = reqDone.WithContext(auth.WithClaims(reqDone.Context(), lead))
	reqDone = reqDone.WithContext(contextWithChi(reqDone, rctx4))
	rrDone := httptest.NewRecorder()
	svc.HandleAssistComplete(rrDone, reqDone)
	if rrDone.Code != http.StatusOK {
		t.Fatalf("complete status=%d body=%s", rrDone.Code, rrDone.Body.String())
	}

	// Reports summary (auto-enables REPORTS_PRO)
	reqRep := httptest.NewRequest(http.MethodGet, "/v1/retailer/reports/summary", nil)
	reqRep = reqRep.WithContext(auth.WithClaims(reqRep.Context(), owner))
	rrRep := httptest.NewRecorder()
	svc.HandleReportsSummary(rrRep, reqRep)
	if rrRep.Code != http.StatusOK {
		t.Fatalf("reports summary status=%d body=%s", rrRep.Code, rrRep.Body.String())
	}
	var summary map[string]any
	_ = json.Unmarshal(rrRep.Body.Bytes(), &summary)
	if summary["on_hand_sku_count"] == nil {
		t.Fatalf("summary=%v", summary)
	}
	// Seed a POS sale for sales report
	// Create register + session + sale quickly via memory maps
	regID := svc.newID()
	sessID := svc.newID()
	saleID := svc.newID()
	_ = svc.saveRegister(t.Context(), RegisterDTO{
		RegisterID: regID, RetailerID: "ret-p6", LocationID: primary.LocationID,
		Label: "R1", Status: RegisterStatusActive,
	})
	_ = svc.savePosSession(t.Context(), PosSessionDTO{
		SessionID: sessID, RegisterID: regID, LocationID: primary.LocationID,
		RetailerID: "ret-p6", OpenedByUserID: "owner", Status: PosSessionOpen,
		OpeningFloatMinor: 0, Currency: "UZS",
		OpenedAt: time.Now().UTC().Format(time.RFC3339Nano),
	})
	_ = svc.savePosSale(t.Context(), PosSaleDTO{
		SaleID: saleID, SessionID: sessID, RegisterID: regID, LocationID: primary.LocationID,
		RetailerID: "ret-p6", CashierUserID: "owner", Status: "COMPLETED",
		TotalMinor: 5000, Currency: "UZS", ReceiptNumber: "R1",
		Lines: []PosSaleLine{
			{Sku: "SKU-A", Name: "A", Qty: 1, UnitPriceMinor: 5000, LineTotalMinor: 5000},
		},
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	})

	reqSales := httptest.NewRequest(http.MethodGet, "/v1/retailer/reports/sales?group_by=sku", nil)
	reqSales = reqSales.WithContext(auth.WithClaims(reqSales.Context(), owner))
	rrSales := httptest.NewRecorder()
	svc.HandleReportsSales(rrSales, reqSales)
	if rrSales.Code != http.StatusOK {
		t.Fatalf("sales report status=%d body=%s", rrSales.Code, rrSales.Body.String())
	}
	var salesRep map[string]any
	_ = json.Unmarshal(rrSales.Body.Bytes(), &salesRep)
	items, _ := salesRep["items"].([]any)
	if len(items) < 1 {
		t.Fatalf("expected sales items, got %v", salesRep)
	}

	// CSV export
	reqCSV := httptest.NewRequest(http.MethodGet, "/v1/retailer/reports/export?report=sales", nil)
	reqCSV = reqCSV.WithContext(auth.WithClaims(reqCSV.Context(), owner))
	rrCSV := httptest.NewRecorder()
	svc.HandleReportsExport(rrCSV, reqCSV)
	if rrCSV.Code != http.StatusOK {
		t.Fatalf("csv status=%d", rrCSV.Code)
	}
	if !strings.Contains(rrCSV.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("content-type=%s", rrCSV.Header().Get("Content-Type"))
	}
	if !strings.Contains(rrCSV.Body.String(), "sales_minor") {
		t.Fatalf("csv body=%s", rrCSV.Body.String())
	}
}
