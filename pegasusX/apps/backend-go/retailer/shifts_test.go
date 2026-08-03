package retailer

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestShiftsClockAndPOSRequire(t *testing.T) {
	t.Parallel()
	n := 0
	svc := NewService(ServiceConfig{
		Now: time.Now,
		NewID: func() string {
			n++
			return "sh-" + string(rune('A'+n%26)) + string(rune('0'+n/26))
		},
	})
	primary, err := svc.EnsurePrimaryLocation(t.Context(), "ret-sh")
	if err != nil {
		t.Fatal(err)
	}
	ownerClaims := auth.Claims{
		Subject: "owner", Role: auth.RoleRetailer, RetailerOrgID: "ret-sh", RetailerRole: "OWNER", RetailerUserID: "owner",
	}
	cashierClaims := auth.Claims{
		Subject: "cashier", Role: auth.RoleRetailer, RetailerOrgID: "ret-sh", RetailerRole: "CASHIER", RetailerUserID: "cashier",
	}

	// Enable SHIFTS pack (require_shift_to_open_register defaults true).
	_ = svc.SetPackEnabled(t.Context(), "ret-sh", PackTEAM, "owner", true, nil)
	_ = svc.SetPackEnabled(t.Context(), "ret-sh", PackSHIFTS, "owner", true, map[string]any{
		"require_clock_in":               true,
		"require_shift_to_open_register": true,
		"max_shift_hours":                12,
		"variance_alert_minor":           int64(1000),
	})
	// POS deps
	_ = svc.SetPackEnabled(t.Context(), "ret-sh", PackSTORESTOCK, "owner", true, nil)
	_ = svc.SetPackEnabled(t.Context(), "ret-sh", PackPOS, "owner", true, nil)

	// Create register
	reqReg := httptest.NewRequest(http.MethodPost, "/v1/retailer/registers", bytes.NewBufferString(`{"label":"Till A"}`))
	reqReg = reqReg.WithContext(auth.WithClaims(reqReg.Context(), ownerClaims))
	rrReg := httptest.NewRecorder()
	svc.HandleRegisters(rrReg, reqReg)
	if rrReg.Code != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", rrReg.Code, rrReg.Body.String())
	}
	var reg RegisterDTO
	_ = json.Unmarshal(rrReg.Body.Bytes(), &reg)

	// POS open without clock-in → conflict
	reqOpen := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sessions/open",
		bytes.NewBufferString(`{"register_id":"`+reg.RegisterID+`","opening_float_minor":5000,"currency":"UZS"}`))
	reqOpen = reqOpen.WithContext(auth.WithClaims(reqOpen.Context(), cashierClaims))
	rrOpen := httptest.NewRecorder()
	svc.HandlePosSessionOpen(rrOpen, reqOpen)
	if rrOpen.Code != http.StatusConflict {
		t.Fatalf("expected clock_in_required conflict, got %d body=%s", rrOpen.Code, rrOpen.Body.String())
	}
	var errBody map[string]string
	_ = json.Unmarshal(rrOpen.Body.Bytes(), &errBody)
	if errBody["error"] != "clock_in_required" {
		t.Fatalf("error=%v", errBody)
	}

	// Clock in
	reqIn := httptest.NewRequest(http.MethodPost, "/v1/retailer/time/clock-in",
		bytes.NewBufferString(`{"location_id":"`+primary.LocationID+`"}`))
	reqIn = reqIn.WithContext(auth.WithClaims(reqIn.Context(), cashierClaims))
	rrIn := httptest.NewRecorder()
	svc.HandleClockIn(rrIn, reqIn)
	if rrIn.Code != http.StatusCreated {
		t.Fatalf("clock-in status=%d body=%s", rrIn.Code, rrIn.Body.String())
	}
	var entry TimeEntryDTO
	_ = json.Unmarshal(rrIn.Body.Bytes(), &entry)
	if entry.Status != TimeEntryOpen || entry.UserID != "cashier" {
		t.Fatalf("entry=%+v", entry)
	}

	// Double clock-in → conflict
	reqIn2 := httptest.NewRequest(http.MethodPost, "/v1/retailer/time/clock-in",
		bytes.NewBufferString(`{"location_id":"`+primary.LocationID+`"}`))
	reqIn2 = reqIn2.WithContext(auth.WithClaims(reqIn2.Context(), cashierClaims))
	rrIn2 := httptest.NewRecorder()
	svc.HandleClockIn(rrIn2, reqIn2)
	if rrIn2.Code != http.StatusConflict {
		t.Fatalf("double clock-in want conflict, got %d", rrIn2.Code)
	}

	// Open shift with float
	reqShift := httptest.NewRequest(http.MethodPost, "/v1/retailer/shifts",
		bytes.NewBufferString(`{"location_id":"`+primary.LocationID+`","register_id":"`+reg.RegisterID+`","opening_float_minor":5000,"currency":"UZS"}`))
	reqShift = reqShift.WithContext(auth.WithClaims(reqShift.Context(), cashierClaims))
	rrShift := httptest.NewRecorder()
	svc.HandleShifts(rrShift, reqShift)
	if rrShift.Code != http.StatusCreated {
		t.Fatalf("open shift status=%d body=%s", rrShift.Code, rrShift.Body.String())
	}
	var shift ShiftDTO
	_ = json.Unmarshal(rrShift.Body.Bytes(), &shift)
	if shift.Status != ShiftOpen || shift.OpeningFloatMinor != 5000 {
		t.Fatalf("shift=%+v", shift)
	}

	// POS open succeeds after clock-in
	reqOpen2 := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sessions/open",
		bytes.NewBufferString(`{"register_id":"`+reg.RegisterID+`","opening_float_minor":5000,"currency":"UZS"}`))
	reqOpen2 = reqOpen2.WithContext(auth.WithClaims(reqOpen2.Context(), cashierClaims))
	rrOpen2 := httptest.NewRecorder()
	svc.HandlePosSessionOpen(rrOpen2, reqOpen2)
	if rrOpen2.Code != http.StatusCreated && rrOpen2.Code != http.StatusOK {
		t.Fatalf("pos open status=%d body=%s", rrOpen2.Code, rrOpen2.Body.String())
	}
	var sess PosSessionDTO
	_ = json.Unmarshal(rrOpen2.Body.Bytes(), &sess)
	if sess.SessionID == "" {
		t.Fatal("missing session")
	}

	// Linked shift should carry POS session
	linked, found, _ := svc.getShift(t.Context(), shift.ShiftID)
	if !found || linked.LinkedPosSessionID != sess.SessionID {
		t.Fatalf("expected linked session %s, got %+v", sess.SessionID, linked)
	}

	// Close POS with large variance (no cash sales → expected = float)
	reqClosePos := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sessions/"+sess.SessionID+"/close",
		bytes.NewBufferString(`{"closing_cash_minor":0}`))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionID", sess.SessionID)
	reqClosePos = reqClosePos.WithContext(auth.WithClaims(reqClosePos.Context(), cashierClaims))
	reqClosePos = reqClosePos.WithContext(contextWithChi(reqClosePos, rctx))
	rrClosePos := httptest.NewRecorder()
	svc.HandlePosSessionClose(rrClosePos, reqClosePos)
	if rrClosePos.Code != http.StatusOK {
		t.Fatalf("pos close status=%d body=%s", rrClosePos.Code, rrClosePos.Body.String())
	}

	// Close shift with variance
	reqCloseSh := httptest.NewRequest(http.MethodPost, "/v1/retailer/shifts/"+shift.ShiftID+"/close",
		bytes.NewBufferString(`{"closing_cash_minor":0}`))
	rctx2 := chi.NewRouteContext()
	rctx2.URLParams.Add("shiftID", shift.ShiftID)
	reqCloseSh = reqCloseSh.WithContext(auth.WithClaims(reqCloseSh.Context(), cashierClaims))
	reqCloseSh = reqCloseSh.WithContext(contextWithChi(reqCloseSh, rctx2))
	rrCloseSh := httptest.NewRecorder()
	svc.HandleShiftClose(rrCloseSh, reqCloseSh)
	if rrCloseSh.Code != http.StatusOK {
		t.Fatalf("shift close status=%d body=%s", rrCloseSh.Code, rrCloseSh.Body.String())
	}
	var closed ShiftDTO
	_ = json.Unmarshal(rrCloseSh.Body.Bytes(), &closed)
	if closed.Status != ShiftClosed || closed.VarianceMinor == nil {
		t.Fatalf("closed=%+v", closed)
	}
	// expected cash: opening float + cash sales (0) = 5000; closing 0 → variance -5000
	if *closed.ExpectedCashMinor != 5000 || *closed.VarianceMinor != -5000 {
		t.Fatalf("expected=5000 variance=-5000 got expected=%v variance=%v", closed.ExpectedCashMinor, closed.VarianceMinor)
	}

	// Clock out
	reqOut := httptest.NewRequest(http.MethodPost, "/v1/retailer/time/clock-out", nil)
	reqOut = reqOut.WithContext(auth.WithClaims(reqOut.Context(), cashierClaims))
	rrOut := httptest.NewRecorder()
	svc.HandleClockOut(rrOut, reqOut)
	if rrOut.Code != http.StatusOK {
		t.Fatalf("clock-out status=%d body=%s", rrOut.Code, rrOut.Body.String())
	}

	// Entries list
	reqList := httptest.NewRequest(http.MethodGet, "/v1/retailer/time/entries", nil)
	reqList = reqList.WithContext(auth.WithClaims(reqList.Context(), cashierClaims))
	rrList := httptest.NewRecorder()
	svc.HandleTimeEntries(rrList, reqList)
	if rrList.Code != http.StatusOK {
		t.Fatalf("entries status=%d", rrList.Code)
	}
	var list map[string]any
	_ = json.Unmarshal(rrList.Body.Bytes(), &list)
	if list["clocked_in"] != false {
		t.Fatalf("should not be clocked in: %v", list)
	}
}

func TestShiftsPOSNoRequireWhenPackOff(t *testing.T) {
	t.Parallel()
	n := 0
	svc := NewService(ServiceConfig{
		Now: time.Now,
		NewID: func() string {
			n++
			return "nosh-" + string(rune('A'+n%26))
		},
	})
	_, _ = svc.EnsurePrimaryLocation(t.Context(), "ret-nosh")
	// No SHIFTS pack — POS open should work without clock-in
	reqReg := httptest.NewRequest(http.MethodPost, "/v1/retailer/registers", bytes.NewBufferString(`{"label":"T"}`))
	reqReg = reqReg.WithContext(auth.WithClaims(reqReg.Context(), auth.Claims{
		Subject: "o", Role: auth.RoleRetailer, RetailerOrgID: "ret-nosh", RetailerRole: "OWNER",
	}))
	rrReg := httptest.NewRecorder()
	svc.HandleRegisters(rrReg, reqReg)
	var reg RegisterDTO
	_ = json.Unmarshal(rrReg.Body.Bytes(), &reg)

	reqOpen := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sessions/open",
		bytes.NewBufferString(`{"register_id":"`+reg.RegisterID+`","opening_float_minor":0}`))
	reqOpen = reqOpen.WithContext(auth.WithClaims(reqOpen.Context(), auth.Claims{
		Subject: "c", Role: auth.RoleRetailer, RetailerOrgID: "ret-nosh", RetailerRole: "CASHIER", RetailerUserID: "c",
	}))
	rrOpen := httptest.NewRecorder()
	svc.HandlePosSessionOpen(rrOpen, reqOpen)
	if rrOpen.Code != http.StatusCreated && rrOpen.Code != http.StatusOK {
		t.Fatalf("pos open without shifts pack status=%d body=%s", rrOpen.Code, rrOpen.Body.String())
	}
}
