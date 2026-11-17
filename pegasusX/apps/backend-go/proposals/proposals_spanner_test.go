package proposals_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/proposals"
	"github.com/pegasusx/pegasusx/apps/backend-go/roi"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func openProposalsSpannerClient(t *testing.T) *spanner.Client {
	t.Helper()
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST unset; skipping live Spanner proposals test")
	}
	envOr := func(k, def string) string {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
		return def
	}
	dbPath := "projects/" + envOr("SPANNER_PROJECT", "pegasusx-local") +
		"/instances/" + envOr("SPANNER_INSTANCE", "pegasusx-instance") +
		"/databases/" + envOr("SPANNER_DATABASE", "pegasusx-db")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, err := spanner.NewClient(ctx, dbPath,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		if requireSpanner {
			t.Fatalf("spanner dial: %v", err)
		}
		t.Skipf("spanner unavailable: %v", err)
	}
	ro := c.ReadOnlyTransaction()
	_, pErr := ro.ReadRow(ctx, "Proposals", spanner.Key{"__probe__", "__probe__"}, []string{"ProposalId"})
	ro.Close()
	if pErr != nil && spanner.ErrCode(pErr) != codes.NotFound {
		c.Close()
		if requireSpanner {
			t.Fatalf("Proposals unavailable: %v", pErr)
		}
		t.Skipf("Proposals unavailable: %v", pErr)
	}
	t.Cleanup(c.Close)
	return c
}

func TestProposals_LiveSpanner_3AgentsGovernedExecutionAndROI(t *testing.T) {
	client := openProposalsSpannerClient(t)
	ctx := context.Background()
	tid := "sup-pr-sp-" + uuid.NewString()[:8]
	now := time.Now().UTC()

	wh1 := "wh1-" + uuid.NewString()[:8]
	wh2 := "wh2-" + uuid.NewString()[:8]
	facID := "fac-" + uuid.NewString()[:8]
	prod1 := "sku1-" + uuid.NewString()[:8]
	prod2 := "sku2-" + uuid.NewString()[:8]
	insID := "ins-" + uuid.NewString()[:8]
	invID := "inv-" + uuid.NewString()[:8]
	retID := "ret-" + uuid.NewString()[:8]
	lotID := "lot-" + uuid.NewString()[:8]

	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = client.Apply(cleanCtx, []*spanner.Mutation{
			spanner.Delete("ReplenishmentInsights", spanner.Key{insID}),
			spanner.Delete("ArInvoices", spanner.Key{invID}),
			spanner.Delete("Retailers", spanner.Key{retID}),
			spanner.Delete("StockLots", spanner.Key{lotID}),
			spanner.Delete("Products", spanner.Key{prod1}),
			spanner.Delete("Products", spanner.Key{prod2}),
			spanner.Delete("Warehouses", spanner.Key{wh1}),
			spanner.Delete("Warehouses", spanner.Key{wh2}),
			spanner.Delete("Proposals", spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ApprovalRequests", spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ApprovalDecisions", spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid}, Kind: spanner.ClosedClosed}),
			spanner.Delete("RoiEvents", spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid}, Kind: spanner.ClosedClosed}),
		})
	})

	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("Warehouses", map[string]any{
			"WarehouseId": wh1, "SupplierId": tid, "Name": "Main WH", "CoverageRadiusKm": float64(50),
			"PrimaryFactoryId": facID, "IsActive": true, "IsOnShift": true,
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Warehouses", map[string]any{
			"WarehouseId": wh2, "SupplierId": tid, "Name": "North WH", "CoverageRadiusKm": float64(50),
			"PrimaryFactoryId": facID, "IsActive": true, "IsOnShift": true,
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Products", map[string]any{
			"ProductId": prod1, "SupplierId": tid, "CategoryId": "cat-1",
			"Name": "Butter 200g", "PriceMinor": int64(20000), "Currency": "UZS",
			"StockQuantity": int64(5), "IsActive": true, "Version": int64(1),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Products", map[string]any{
			"ProductId": prod2, "SupplierId": tid, "CategoryId": "cat-1",
			"Name": "Kefir 1L", "PriceMinor": int64(10000), "Currency": "UZS",
			"StockQuantity": int64(50), "IsActive": true, "Version": int64(1),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("ReplenishmentInsights", map[string]any{
			"InsightId": insID, "WarehouseId": wh1, "ProductId": prod1, "SupplierId": tid,
			"CurrentStock": int64(2), "SuggestedQuantity": int64(50), "UrgencyLevel": "CRITICAL",
			"ReasonCode": "LOW_STOCK", "Status": "PENDING", "TargetFactoryId": facID,
			"CreatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Retailers", map[string]any{
			"RetailerId": retID, "Phone": "+998901234567", "Name": "Navruz Savdo", "CountryCode": "UZ",
			"CreatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("ArInvoices", map[string]any{
			"InvoiceId": invID, "SupplierId": tid, "RetailerId": retID, "OrderId": "ord-" + uuid.NewString()[:8],
			"Status": "OPEN", "PrincipalMinor": int64(450000), "BalanceMinor": int64(450000),
			"Currency": "UZS", "CreditLeaveAt": now.Add(-20 * 24 * time.Hour), "DueAt": now.Add(-7 * 24 * time.Hour),
			"TermsDays": int64(13), "GracePeriodDays": int64(0), "DunningStep": int64(1),
			"Version": int64(1), "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("StockLots", map[string]any{
			"LotId": lotID, "SupplierId": tid, "WarehouseId": wh1, "ProductId": prod2, "LocationId": "bin-a1",
			"ExpiryDate": civil.DateOf(now.Add(6 * 24 * time.Hour)), "QuantityOnHand": int64(30),
			"QuantityReserved": int64(0), "ReceivedAt": spanner.CommitTimestamp, "Status": "AVAILABLE",
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
	}); err != nil {
		t.Fatalf("seed live Spanner proposals data: %v", err)
	}

	appSvc := approvals.NewService(client)
	roiSvc := roi.NewService(client)
	propSvc := proposals.NewService(client)
	propSvc.SetApprovals(appSvc)
	propSvc.SetROI(roiSvc)
	propSvc.SetDomainExecutor(&proposals.SpannerDomainExecutor{Client: client})

	created, err := propSvc.RunAgentPass(ctx, tid, "ALL")
	if err != nil {
		t.Fatalf("RunAgentPass(ALL) live Spanner: %v", err)
	}
	if len(created) != 3 {
		t.Fatalf("expected 3 live Spanner proposals across 3 L1 agents, got %d", len(created))
	}

	for _, p := range created {
		if _, err := propSvc.ApproveOrEditProposal(ctx, proposals.ApproveOrEditInput{
			TenantID: tid, ProposalID: p.ProposalID,
			ApproverID: "owner-sp-1", ActorType: auth.ActorTypeHuman, ApproverRole: auth.SupplierRoleOwner,
			Comment: "approved in Spanner test",
		}); err != nil {
			t.Fatalf("ApproveOrEditProposal(%s): %v", p.AgentID, err)
		}
		res, err := propSvc.ExecuteProposal(ctx, proposals.ExecuteInput{
			TenantID: tid, ProposalID: p.ProposalID,
			ExecutorID: "owner-sp-1", ActorType: auth.ActorTypeHuman, ExecutorRole: auth.SupplierRoleOwner,
		})
		if err != nil {
			t.Fatalf("ExecuteProposal(%s): %v", p.AgentID, err)
		}
		if res.Proposal.Status != proposals.StatusExecuted || res.Proposal.RoiEventID == "" {
			t.Fatalf("expected EXECUTED + RoiEventID for %s, got %+v", p.AgentID, res.Proposal)
		}
		// Double execute must fail with ErrAlreadyExecuted.
		if _, err := propSvc.ExecuteProposal(ctx, proposals.ExecuteInput{
			TenantID: tid, ProposalID: p.ProposalID,
			ExecutorID: "owner-sp-1", ActorType: auth.ActorTypeHuman, ExecutorRole: auth.SupplierRoleOwner,
		}); !errors.Is(err, proposals.ErrAlreadyExecuted) {
			t.Fatalf("expected ErrAlreadyExecuted on replay, got %v", err)
		}
	}

	// Verify ROI summary totals across all 3 executed proposals!
	sum, err := roiSvc.GetSummary(ctx, tid, 90)
	if err != nil {
		t.Fatalf("ROI GetSummary: %v", err)
	}
	// Reorder: 50 * 20,000 * 20% = 200,000; Collections: 450,000; Expiry: 30 * 10,000 = 300,000 -> total 950,000!
	if sum.TotalSavingsMinor != 950000 {
		t.Fatalf("expected 950000 total ROI savings from 3 agents, got %d", sum.TotalSavingsMinor)
	}
}
