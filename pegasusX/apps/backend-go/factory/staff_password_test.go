package factory

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVerifyFactoryStaffSecret_RejectsUnsetAndPlain(t *testing.T) {
	if verifyFactoryStaffSecret("unset", "unset") {
		t.Fatal("unset sentinel must never authenticate")
	}
	if verifyFactoryStaffSecret("plaintext", "plaintext") {
		t.Fatal("plain hash compare is gone")
	}
	hash, err := hashFactoryStaffSecret("1234")
	if err != nil {
		t.Fatal(err)
	}
	if !verifyFactoryStaffSecret(hash, "1234") {
		t.Fatal("bcrypt must verify")
	}
	if verifyFactoryStaffSecret(hash, "9999") {
		t.Fatal("wrong pin must fail")
	}
}

func TestHandleStaffSetPassword_RequiresSecret(t *testing.T) {
	repo := &factoryRepoSpy{}
	svc := newFactoryTestService(repo, &factoryCacheBackendSpy{})
	repo.svc = svc
	req := withFactoryRouteParam(
		httptest.NewRequest(http.MethodPost, "/v1/factory/staff/stf_x/set-password", strings.NewReader(`{}`)),
		"staffID", "stf_x",
	)
	rr := httptest.NewRecorder()
	svc.HandleStaffSetPassword(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400", rr.Code)
	}
}

func TestHandleStaffSetPassword_Memory(t *testing.T) {
	repo := &factoryRepoSpy{}
	svc := newFactoryTestService(repo, &factoryCacheBackendSpy{})
	repo.svc = svc
	svc.mu.Lock()
	svc.staff = []StaffRow{{StaffID: "stf_x", Name: "X", Role: "FACTORY_OPERATOR"}}
	svc.mu.Unlock()
	req := withFactoryRouteParam(
		httptest.NewRequest(http.MethodPost, "/v1/factory/staff/stf_x/set-password", strings.NewReader(`{"pin":"4321"}`)),
		"staffID", "stf_x",
	)
	rr := httptest.NewRecorder()
	svc.HandleStaffSetPassword(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["password_set"] != true {
		t.Fatalf("body=%v", body)
	}
	svc.mu.Lock()
	hash := svc.staff[0].PasswordHash
	svc.mu.Unlock()
	if !verifyFactoryStaffSecret(hash, "4321") {
		t.Fatal("stored hash must verify pin")
	}
}
