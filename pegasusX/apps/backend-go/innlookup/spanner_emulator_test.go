package innlookup

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"golang.org/x/crypto/bcrypt"
)

func envOrDefaultTest(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func requireSpannerEmulatorForINNTest(t *testing.T) *spanner.Client {
	t.Helper()
	if strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST")) == "" {
		if os.Getenv("PARITY_REQUIRE_SPANNER") == "1" {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping Spanner emulator integration test")
	}
	project := envOrDefaultTest("SPANNER_PROJECT", "pegasusx-local")
	instance := envOrDefaultTest("SPANNER_INSTANCE", "pegasusx-instance")
	database := envOrDefaultTest("SPANNER_DATABASE", "pegasusx-db")
	dsn := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := spanner.NewClient(ctx, dsn)
	if err != nil {
		t.Fatalf("spanner.NewClient(%s): %v", dsn, err)
	}
	t.Cleanup(client.Close)
	return client
}

func random7Digits() uint32 {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return uint32(time.Now().UnixNano() % 10000000)
	}
	return binary.LittleEndian.Uint32(b[:]) % 10000000
}

func cleanupINNTestData(t *testing.T, client *spanner.Client, homeCell, normINN, supID, retID string, reqIDs []string, phones []string) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		var muts []*spanner.Mutation
		// Delete any SupplierUsers created for test supplier
		iter := client.Single().Query(ctx, spanner.Statement{
			SQL:    `SELECT UserId FROM SupplierUsers WHERE SupplierId = @sid`,
			Params: map[string]any{"sid": supID},
		})
		for {
			row, err := iter.Next()
			if err != nil {
				break
			}
			var uid string
			if row.Columns(&uid) == nil && uid != "" {
				muts = append(muts, spanner.Delete("SupplierUsers", spanner.Key{uid}))
			}
		}
		iter.Stop()

		// Delete OutboxEvents matching test request IDs or supplier ID
		obIter := client.Single().Query(ctx, spanner.Statement{
			SQL:    `SELECT EventId FROM OutboxEvents WHERE SupplierId = @sid OR AggregateId IN UNNEST(@aggIds)`,
			Params: map[string]any{"sid": supID, "aggIds": reqIDs},
		})
		for {
			row, err := obIter.Next()
			if err != nil {
				break
			}
			var eid string
			if row.Columns(&eid) == nil && eid != "" {
				muts = append(muts, spanner.Delete("OutboxEvents", spanner.Key{eid}))
			}
		}
		obIter.Stop()

		// Delete any RetailerUsers/RetailerUserMemberships created for test retailer/phones
		var retUserIDs []string
		rIter := client.Single().Query(ctx, spanner.Statement{
			SQL:    `SELECT UserId FROM RetailerUsers WHERE RetailerId = @rid OR Phone IN UNNEST(@phones)`,
			Params: map[string]any{"rid": retID, "phones": phones},
		})
		for {
			row, err := rIter.Next()
			if err != nil {
				break
			}
			var uid string
			if row.Columns(&uid) == nil && uid != "" {
				retUserIDs = append(retUserIDs, uid)
				muts = append(muts, spanner.Delete("RetailerUsers", spanner.Key{uid}))
			}
		}
		rIter.Stop()
		if len(retUserIDs) > 0 {
			mIter := client.Single().Query(ctx, spanner.Statement{
				SQL:    `SELECT UserId, RetailerId FROM RetailerUserMemberships WHERE UserId IN UNNEST(@uids)`,
				Params: map[string]any{"uids": retUserIDs},
			})
			for {
				row, err := mIter.Next()
				if err != nil {
					break
				}
				var uid, mrid string
				if row.Columns(&uid, &mrid) == nil && uid != "" {
					muts = append(muts, spanner.Delete("RetailerUserMemberships", spanner.Key{uid, mrid}))
				}
			}
			mIter.Stop()
		}

		for _, rid := range reqIDs {
			muts = append(muts, spanner.Delete("JoinRequests", spanner.Key{rid}))
		}
		if homeCell != "" && normINN != "" {
			muts = append(muts, spanner.Delete("InnRegistryCache", spanner.Key{homeCell, normINN}))
		}
		if supID != "" {
			muts = append(muts, spanner.Delete("Suppliers", spanner.Key{supID}))
		}
		if retID != "" {
			muts = append(muts, spanner.Delete("Retailers", spanner.Key{retID}))
		}
		if len(muts) > 0 {
			_, _ = client.Apply(ctx, muts)
		}
	})
}

func TestSpannerStore_CacheTenantMatchAndJoinRequests(t *testing.T) {
	client := requireSpannerEmulatorForINNTest(t)
	store := NewSpannerStore(client)
	ctx := context.Background()

	shortID := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	homeCell := "cell-" + shortID
	otherCell := "other-" + shortID
	normINN := fmt.Sprintf("30%07d", random7Digits())
	retINN := fmt.Sprintf("31%07d", random7Digits())
	supID := "sup_inn_" + shortID
	retID := "ret_inn_" + shortID
	supPhone := fmt.Sprintf("+99890%07d", random7Digits())
	reqPhone1 := fmt.Sprintf("+99891%07d", random7Digits())
	reqPhone2 := fmt.Sprintf("+99893%07d", random7Digits())
	retPhone := fmt.Sprintf("+99894%07d", random7Digits())
	reqID1 := "jr1_" + shortID
	reqID2 := "jr2_" + shortID

	cleanupINNTestData(t, client, homeCell, normINN, supID, retID, []string{reqID1, reqID2}, []string{supPhone, reqPhone1, reqPhone2, retPhone})

	// 1. PutCachedINN / GetCachedINN including TTL expiry and HomeCell isolation.
	// Keep explicit timestamps in the past because JoinRequests.decided_at has allow_commit_timestamp=true.
	now := time.Now().UTC().Add(-15 * time.Minute).Truncate(time.Microsecond)
	cached := LookupResult{
		Status:        StatusFound,
		NormalizedINN: normINN,
		EntityType:    EntityTypeYuridik,
		LegalName:     "OOO Baraka Logistics",
		Address:       "Tashkent, Sergeli 12",
		RegistryState: "ACTIVE",
		RawJSON:       json.RawMessage(`{"tin":"` + normINN + `","state":"ACTIVE"}`),
		FetchedAt:     now,
		ExpiresAt:     now.Add(1 * time.Hour),
	}
	if err := store.PutCachedINN(ctx, homeCell, cached); err != nil {
		t.Fatalf("PutCachedINN: %v", err)
	}

	got, found, err := store.GetCachedINN(ctx, homeCell, normINN, now.Add(5*time.Minute))
	if err != nil {
		t.Fatalf("GetCachedINN: %v", err)
	}
	if !found || !got.Cached || got.LegalName != cached.LegalName || got.EntityType != EntityTypeYuridik {
		t.Fatalf("unexpected cached lookup hit: found=%v got=%+v", found, got)
	}

	// Expired read returns miss.
	_, foundExpired, err := store.GetCachedINN(ctx, homeCell, normINN, now.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("GetCachedINN expired: %v", err)
	}
	if foundExpired {
		t.Fatalf("expected expired cache entry to miss")
	}

	// Cross-cell read returns miss.
	_, foundOtherCell, err := store.GetCachedINN(ctx, otherCell, normINN, now.Add(5*time.Minute))
	if err != nil {
		t.Fatalf("GetCachedINN other cell: %v", err)
	}
	if foundOtherCell {
		t.Fatalf("expected cross-cell cache lookup to miss")
	}

	// 2. Seed Supplier + Retailer rows and verify FindTenantByNormalizedINN.
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("Suppliers", map[string]any{
			"SupplierId":      supID,
			"Name":            "Baraka Trade LLC",
			"CountryCode":     "UZ",
			"Currency":        "UZS",
			"IsConfigured":    true,
			"HomeCell":        homeCell,
			"NormalizedTaxId": normINN,
			"CreatedAt":       spanner.CommitTimestamp,
			"UpdatedAt":       spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("Retailers", map[string]any{
			"RetailerId":      retID,
			"Phone":           retPhone,
			"Name":            "Yangi Bozor Shop",
			"CountryCode":     "UZ",
			"HomeCell":        homeCell,
			"NormalizedTaxId": retINN,
			"CreatedAt":       spanner.CommitTimestamp,
		}),
	}); err != nil {
		t.Fatalf("seed tenant rows: %v", err)
	}

	matchSup, ok, err := store.FindTenantByNormalizedINN(ctx, homeCell, TenantTypeSupplier, normINN)
	if err != nil {
		t.Fatalf("FindTenantByNormalizedINN(SUPPLIER): %v", err)
	}
	if !ok || matchSup.TenantID != supID || matchSup.LegalName != "Baraka Trade LLC" {
		t.Fatalf("unexpected supplier match: ok=%v match=%+v", ok, matchSup)
	}

	matchRet, ok, err := store.FindTenantByNormalizedINN(ctx, homeCell, TenantTypeRetailer, retINN)
	if err != nil {
		t.Fatalf("FindTenantByNormalizedINN(RETAILER): %v", err)
	}
	if !ok || matchRet.TenantID != retID || matchRet.LegalName != "Yangi Bozor Shop" {
		t.Fatalf("unexpected retailer match: ok=%v match=%+v", ok, matchRet)
	}

	_, okOther, err := store.FindTenantByNormalizedINN(ctx, otherCell, TenantTypeSupplier, normINN)
	if err != nil {
		t.Fatalf("FindTenantByNormalizedINN(other cell): %v", err)
	}
	if okOther {
		t.Fatalf("expected no match in other HomeCell")
	}

	// 3. CreateJoinRequest, idempotency, ListJoinRequests, GetJoinRequest.
	emitPending := func(buf outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, buf, "JoinRequest", reqID1, "identity.join.requests", map[string]string{
			"supplier_id": supID,
			"request_id":  reqID1,
			"status":      JoinStatusPending,
		})
	}
	supHash, err := bcrypt.GenerateFromPassword([]byte("ColleaguePass!99"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt hash: %v", err)
	}
	jr1, err := store.CreateJoinRequest(ctx, JoinRequest{
		RequestID:             reqID1,
		HomeCell:              homeCell,
		TargetTenantType:      TenantTypeSupplier,
		TargetTenantID:        supID,
		NormalizedINN:         normINN,
		RequesterPhone:        reqPhone1,
		RequesterName:         "Azizbek",
		RequesterEmail:        "aziz@" + shortID + ".example",
		RoleSought:            "WAREHOUSE_ADMIN",
		RequesterPasswordHash: string(supHash),
		Status:                JoinStatusPending,
		CreatedAt:             now,
	}, emitPending)
	if err != nil {
		t.Fatalf("CreateJoinRequest(1): %v", err)
	}
	if jr1.RequestID != reqID1 || jr1.Status != JoinStatusPending || jr1.RequesterPasswordHash != "" {
		t.Fatalf("unexpected created join request: %+v", jr1)
	}

	// Idempotent duplicate PENDING request returns existing reqID1 instead of inserting duplicate.
	jr1Dup, err := store.CreateJoinRequest(ctx, JoinRequest{
		RequestID:        "jr_unused_" + shortID,
		HomeCell:         homeCell,
		TargetTenantType: TenantTypeSupplier,
		TargetTenantID:   supID,
		NormalizedINN:    normINN,
		RequesterPhone:   reqPhone1,
		RoleSought:       "WAREHOUSE_ADMIN",
		Status:           JoinStatusPending,
		CreatedAt:        now,
	}, nil)
	if err != nil {
		t.Fatalf("CreateJoinRequest(dup): %v", err)
	}
	if jr1Dup.RequestID != reqID1 {
		t.Fatalf("expected idempotent return of %s, got %s", reqID1, jr1Dup.RequestID)
	}

	listed, err := store.ListJoinRequests(ctx, TenantTypeSupplier, supID, JoinStatusPending, 10)
	if err != nil {
		t.Fatalf("ListJoinRequests: %v", err)
	}
	if len(listed) != 1 || listed[0].RequestID != reqID1 {
		t.Fatalf("unexpected ListJoinRequests output: %+v", listed)
	}

	gotJR, foundJR, err := store.GetJoinRequest(ctx, TenantTypeSupplier, supID, reqID1)
	if err != nil {
		t.Fatalf("GetJoinRequest: %v", err)
	}
	if !foundJR || gotJR.RequesterPhone != reqPhone1 || gotJR.RoleSought != "WAREHOUSE_ADMIN" {
		t.Fatalf("unexpected GetJoinRequest output: found=%v jr=%+v", foundJR, gotJR)
	}

	// 4. DecideJoinRequest APPROVED inserts SupplierUsers row atomically + outbox event.
	decidedAt := now.Add(10 * time.Minute)
	approvedJR, err := store.DecideJoinRequest(
		ctx, TenantTypeSupplier, supID, reqID1, JoinStatusApproved, "admin_"+shortID, "Welcome aboard", decidedAt,
		func(buf outbox.TxnBuffer) error {
			return outbox.EmitJSON(ctx, buf, "JoinRequest", reqID1, "identity.join.requests", map[string]string{
				"supplier_id": supID,
				"request_id":  reqID1,
				"status":      JoinStatusApproved,
			})
		},
	)
	if err != nil {
		t.Fatalf("DecideJoinRequest(APPROVED): %v", err)
	}
	if approvedJR.Status != JoinStatusApproved || approvedJR.DecidedBy != "admin_"+shortID {
		t.Fatalf("unexpected DecideJoinRequest(APPROVED): %+v", approvedJR)
	}

	// Verify SupplierUsers row created with matching phone, role, and login password hash; hash cleared from JoinRequests.
	uIter := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT SupplierRole, IsActive, IFNULL(PasswordHash, '') FROM SupplierUsers WHERE SupplierId = @sid AND Phone = @phone`,
		Params: map[string]any{"sid": supID, "phone": reqPhone1},
	})
	defer uIter.Stop()
	uRow, err := uIter.Next()
	if err != nil {
		t.Fatalf("expected SupplierUsers row created for %s: %v", reqPhone1, err)
	}
	var role, storedHash string
	var isActive bool
	if err := uRow.Columns(&role, &isActive, &storedHash); err != nil {
		t.Fatalf("scan SupplierUsers row: %v", err)
	}
	if role != "WAREHOUSE_ADMIN" || !isActive {
		t.Fatalf("unexpected SupplierUsers row: role=%s active=%v", role, isActive)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte("ColleaguePass!99")); err != nil {
		t.Fatalf("expected provisioned SupplierUsers.PasswordHash to authenticate password: %v", err)
	}
	jrRow, err := client.Single().ReadRow(ctx, "JoinRequests", spanner.Key{reqID1}, []string{"requester_password_hash"})
	if err != nil {
		t.Fatalf("read JoinRequests after decision: %v", err)
	}
	var clearedHash spanner.NullString
	if err := jrRow.Columns(&clearedHash); err != nil {
		t.Fatalf("scan JoinRequests hash: %v", err)
	}
	if clearedHash.Valid {
		t.Fatalf("expected requester_password_hash cleared on decided JoinRequest, got %q", clearedHash.StringVal)
	}

	// Subsequent DecideJoinRequest calls (repeat approve or flip to reject) fail closed with ErrJoinRequestDecided.
	if _, err := store.DecideJoinRequest(ctx, TenantTypeSupplier, supID, reqID1, JoinStatusApproved, "admin_"+shortID, "repeat", decidedAt, nil); !errors.Is(err, ErrJoinRequestDecided) {
		t.Fatalf("expected ErrJoinRequestDecided on repeat approve, got %v", err)
	}
	if _, err := store.DecideJoinRequest(ctx, TenantTypeSupplier, supID, reqID1, JoinStatusRejected, "admin_"+shortID, "flip", decidedAt, nil); !errors.Is(err, ErrJoinRequestDecided) {
		t.Fatalf("expected ErrJoinRequestDecided when rejecting already approved request, got %v", err)
	}

	// Verify OutboxEvents recorded both PENDING and APPROVED transitions for reqID1.
	var obCount int64
	if err := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT COUNT(*) FROM OutboxEvents WHERE AggregateId = @aid`,
		Params: map[string]any{"aid": reqID1},
	}).Do(func(r *spanner.Row) error {
		return r.Columns(&obCount)
	}); err != nil {
		t.Fatalf("count OutboxEvents: %v", err)
	}
	if obCount != 2 {
		t.Fatalf("expected 2 outbox events for join request %s, got %d", reqID1, obCount)
	}

	// 5. Create and REJECT a second join request; verify no SupplierUsers row added for reqPhone2.
	if _, err := store.CreateJoinRequest(ctx, JoinRequest{
		RequestID:        reqID2,
		HomeCell:         homeCell,
		TargetTenantType: TenantTypeSupplier,
		TargetTenantID:   supID,
		NormalizedINN:    normINN,
		RequesterPhone:   reqPhone2,
		RequesterName:    "Botir",
		RoleSought:       "WAREHOUSE",
		Status:           JoinStatusPending,
		CreatedAt:        now,
	}, nil); err != nil {
		t.Fatalf("CreateJoinRequest(2): %v", err)
	}
	rejectedJR, err := store.DecideJoinRequest(ctx, TenantTypeSupplier, supID, reqID2, JoinStatusRejected, "admin_"+shortID, "Not recognized", decidedAt, nil)
	if err != nil {
		t.Fatalf("DecideJoinRequest(REJECTED): %v", err)
	}
	if rejectedJR.Status != JoinStatusRejected || rejectedJR.DecisionReason != "Not recognized" {
		t.Fatalf("unexpected DecideJoinRequest(REJECTED): %+v", rejectedJR)
	}

	// 6. Create and APPROVE a RETAILER join request from an existing shop user; verify RetailerUsers & RetailerUserMemberships provisioned.
	retHash, err := bcrypt.GenerateFromPassword([]byte("RetailJoin!88"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt ret hash: %v", err)
	}
	existingShopBUserID := "retu_b_" + shortID
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("RetailerUsers", map[string]any{
			"UserId":       existingShopBUserID,
			"RetailerId":   "ret_shop_b_" + shortID,
			"Phone":        retPhone,
			"Name":         "Shoira Shop B",
			"PasswordHash": string(retHash),
			"FirebaseUid":  "fb-ret-" + shortID,
			"RetailerRole": "OWNER",
			"IsOwner":      true,
			"IsActive":     true,
			"CreatedAt":    now,
			"UpdatedAt":    now,
		}),
	}); err != nil {
		t.Fatalf("seed existing RetailerUsers row: %v", err)
	}
	retReqID := "jr_ret_" + shortID
	cleanupINNTestData(t, client, "", "", "", "", []string{retReqID}, []string{retPhone})
	if _, err := store.CreateJoinRequest(ctx, JoinRequest{
		RequestID:        retReqID,
		HomeCell:         homeCell,
		TargetTenantType: TenantTypeRetailer,
		TargetTenantID:   retID,
		NormalizedINN:    retINN,
		RequesterPhone:   retPhone,
		RequesterName:    "Shoira",
		RoleSought:       "RETAILER",
		Status:           JoinStatusPending,
		CreatedAt:        now,
	}, nil); err != nil {
		t.Fatalf("CreateJoinRequest(RETAILER): %v", err)
	}
	approvedRetJR, err := store.DecideJoinRequest(ctx, TenantTypeRetailer, retID, retReqID, JoinStatusApproved, "owner_"+shortID, "Verified branch", decidedAt, nil)
	if err != nil || approvedRetJR.Status != JoinStatusApproved {
		t.Fatalf("DecideJoinRequest(RETAILER): %+v err=%v", approvedRetJR, err)
	}
	ruIter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT ru.UserId, ru.RetailerRole, ru.IsActive, IFNULL(ru.PasswordHash, ''), IFNULL(ru.FirebaseUid, ''), m.RetailerRole
		      FROM RetailerUsers ru
		      JOIN RetailerUserMemberships m ON m.UserId = ru.UserId AND m.RetailerId = ru.RetailerId
		      WHERE ru.RetailerId = @rid AND ru.Phone = @phone`,
		Params: map[string]any{"rid": retID, "phone": retPhone},
	})
	defer ruIter.Stop()
	ruRow, err := ruIter.Next()
	if err != nil {
		t.Fatalf("expected RetailerUsers + RetailerUserMemberships created for %s on %s: %v", retPhone, retID, err)
	}
	var newRetUID, ruRole, ruPwd, ruFB, mRole string
	var ruActive bool
	if err := ruRow.Columns(&newRetUID, &ruRole, &ruActive, &ruPwd, &ruFB, &mRole); err != nil {
		t.Fatalf("scan RetailerUsers + membership: %v", err)
	}
	if !ruActive || ruRole != "MANAGER" || mRole != "MANAGER" || ruFB != "fb-ret-"+shortID {
		t.Fatalf("unexpected retailer approval provisioning: uid=%s role=%s mRole=%s fb=%s active=%v", newRetUID, ruRole, mRole, ruFB, ruActive)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(ruPwd), []byte("RetailJoin!88")); err != nil {
		t.Fatalf("expected carried-forward PasswordHash to authenticate: %v", err)
	}
}
