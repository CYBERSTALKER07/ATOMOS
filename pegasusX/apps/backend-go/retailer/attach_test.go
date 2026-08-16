package retailer

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type captureRetailerRepo struct {
	testRetailerRepo
	last Retailer
}

func (r *captureRetailerRepo) CreateRetailer(_ context.Context, p Retailer, emit func(outbox.TxnBuffer) error) error {
	r.last = p
	if emit != nil {
		return emit(nopAttachBuf{})
	}
	return nil
}

type nopAttachBuf struct{}

func (nopAttachBuf) BufferOutbox(context.Context, outbox.Event) error { return nil }

func testRegisterReq(supplierID string) RegisterRequest {
	return RegisterRequest{
		Phone:      "+998901111111",
		Name:       "Shop",
		SupplierID: supplierID,
		Lat:        41.3,
		Lng:        69.2,
		H3Cell:     "8928308280fffff",
	}
}

func TestRegister_RequiresTradingPartner(t *testing.T) {
	svc := NewService(ServiceConfig{Repo: &captureRetailerRepo{}, SeedSupplierID: "seed-1"})
	_, err := svc.Register(context.Background(), testRegisterReq(""))
	if !errors.Is(err, ErrTradingPartnerRequired) {
		t.Fatalf("empty partner must not PreferTenant(seed): %v", err)
	}
}

func TestRegister_RejectsSeedOutsideSSMR(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	repo := &captureRetailerRepo{}
	svc := NewService(ServiceConfig{Repo: repo, SeedSupplierID: "seed-1"})
	_, err := svc.Register(context.Background(), testRegisterReq("seed-1"))
	if !errors.Is(err, ErrSeedAttachForbidden) {
		t.Fatalf("err=%v", err)
	}
	if repo.last.RetailerID != "" {
		t.Fatal("must not persist a shop on seed")
	}
}

func TestRegister_AllowsExplicitPartner(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	repo := &captureRetailerRepo{}
	svc := NewService(ServiceConfig{Repo: repo, SeedSupplierID: "seed-1"})
	resp, err := svc.Register(context.Background(), testRegisterReq("sup-minted"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.SupplierID != "sup-minted" || repo.last.SupplierID != "sup-minted" {
		t.Fatalf("partner=%s persisted=%s", resp.SupplierID, repo.last.SupplierID)
	}
}

func TestRegister_UnknownPartner404(t *testing.T) {
	svc := NewService(ServiceConfig{Repo: &captureRetailerRepo{}, SeedSupplierID: "seed-1"})
	svc.SetTradingPartnerLookup(func(_ context.Context, _ string) (bool, error) {
		return false, nil
	})
	_, err := svc.Register(context.Background(), testRegisterReq("ghost"))
	if !errors.Is(err, ErrUnknownTradingPartner) {
		t.Fatalf("err=%v", err)
	}
}

func TestRegister_InviteToken(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	repo := &captureRetailerRepo{}
	svc := NewService(ServiceConfig{
		Repo: repo, SeedSupplierID: "seed-1", JWTSecret: "invite-secret",
		Now: func() time.Time { return time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC) },
	})
	tok, _, err := MintTradingPartnerInvite("invite-secret", "sup-minted", time.Hour, svc.now())
	if err != nil {
		t.Fatal(err)
	}
	req := testRegisterReq("")
	req.InviteToken = tok
	resp, err := svc.Register(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.SupplierID != "sup-minted" {
		t.Fatalf("supplier_id=%s", resp.SupplierID)
	}
}

func TestRegister_SSMRAllowsExplicitSeed(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "ssmr")
	repo := &captureRetailerRepo{}
	svc := NewService(ServiceConfig{Repo: repo, SeedSupplierID: "seed-1"})
	resp, err := svc.Register(context.Background(), testRegisterReq("seed-1"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.SupplierID != "seed-1" {
		t.Fatalf("ssmr seed attach should be explicit: %s", resp.SupplierID)
	}
}

func TestDemoPasswordDisabledOutsideSSMR(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	if secretMatchesDemo("1234") {
		t.Fatal("1234 must not be a master key outside ssmr")
	}
	svc := NewService(ServiceConfig{Repo: &testRetailerRepo{found: true, retailer: Retailer{RetailerID: "ret-1", Phone: "+1"}}, SeedSupplierID: "seed-1"})
	_, ok, err := svc.resolveRetailerLogin(context.Background(), "+1", "1234")
	if err != nil || ok {
		t.Fatalf("demo login must fail outside ssmr ok=%v err=%v", ok, err)
	}
}

func TestDemoPasswordAllowedInSSMR(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "ssmr")
	if !secretMatchesDemo("1234") {
		t.Fatal("ssmr still allows demo secret")
	}
}

func TestHandleRegister_MissingPartner400(t *testing.T) {
	svc := NewService(ServiceConfig{Repo: &captureRetailerRepo{}, SeedSupplierID: "seed-1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/register",
		strings.NewReader(`{"phone":"+99890","name":"S","lat":41,"lng":69,"h3_cell":"8928308280fffff"}`))
	rr := httptest.NewRecorder()
	svc.HandleRegister(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
