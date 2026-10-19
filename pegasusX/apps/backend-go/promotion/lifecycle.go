package promotion

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

type SupplierPromotionCampaign struct {
	CampaignID       string    `json:"campaign_id"`
	SupplierID       string    `json:"supplier_id"`
	Name             string    `json:"name"`
	BudgetLimitMinor int64     `json:"budget_limit_minor"`
	BudgetUsedMinor  int64     `json:"budget_used_minor"`
	Status           string    `json:"status"` // ACTIVE, EXHAUSTED, PAUSED
	CreatedAt        time.Time `json:"created_at"`
}

type RetailerPromotionEnrollment struct {
	EnrollmentID string    `json:"enrollment_id"`
	CampaignID   string    `json:"campaign_id"`
	RetailerID   string    `json:"retailer_id"`
	Status       string    `json:"status"` // ENROLLED, OPTED_OUT
	EnrolledAt   time.Time `json:"enrolled_at"`
}

const (
	CampaignStatusActive    = "ACTIVE"
	CampaignStatusExhausted = "EXHAUSTED"
	CampaignStatusPaused    = "PAUSED"

	EnrollmentStatusEnrolled = "ENROLLED"
	EnrollmentStatusOptedOut = "OPTED_OUT"
)

// CreateCampaign initializes a new budget-capped supplier promotion campaign.
func CreateCampaign(ctx context.Context, client *spanner.Client, supplierID, name string, budgetLimit int64) (string, error) {
	if budgetLimit <= 0 {
		return "", errors.New("budget must be greater than 0")
	}

	campaignID := uuid.New().String()
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		m := spanner.Insert("SupplierPromotionCampaigns",
			[]string{"CampaignId", "SupplierId", "Name", "BudgetLimitMinor", "BudgetUsedMinor", "Status", "CreatedAt"},
			[]interface{}{campaignID, supplierID, name, budgetLimit, 0, CampaignStatusActive, spanner.CommitTimestamp},
		)
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	return campaignID, err
}

// EnrollRetailer allows a retailer to opt-in to an active campaign.
func EnrollRetailer(ctx context.Context, client *spanner.Client, campaignID, retailerID string) error {
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Check campaign status
		stmt := spanner.Statement{
			SQL:    `SELECT Status FROM SupplierPromotionCampaigns WHERE CampaignId = @c`,
			Params: map[string]interface{}{"c": campaignID},
		}
		iter := txn.Query(ctx, stmt)
		defer iter.Stop()
		row, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				return errors.New("campaign not found")
			}
			return err
		}
		var status string
		if err := row.Columns(&status); err != nil {
			return err
		}
		if status != CampaignStatusActive {
			return errors.New("cannot enroll in non-active campaign")
		}

		enrollmentID := uuid.New().String()
		m := spanner.InsertOrUpdate("RetailerPromotionEnrollments",
			[]string{"EnrollmentId", "CampaignId", "RetailerId", "Status", "EnrolledAt"},
			[]interface{}{enrollmentID, campaignID, retailerID, EnrollmentStatusEnrolled, spanner.CommitTimestamp},
		)
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	return err
}

// TrackCampaignUsage deducts the minor currency usage from the campaign budget and auto-exhausts if crossed.
func TrackCampaignUsage(ctx context.Context, txn *spanner.ReadWriteTransaction, campaignID string, usedMinor int64) error {
	stmt := spanner.Statement{
		SQL:    `SELECT BudgetLimitMinor, BudgetUsedMinor FROM SupplierPromotionCampaigns WHERE CampaignId = @c`,
		Params: map[string]interface{}{"c": campaignID},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return errors.New("campaign not found")
		}
		return err
	}
	var limit, used int64
	if err := row.Columns(&limit, &used); err != nil {
		return err
	}

	newUsed := used + usedMinor
	status := CampaignStatusActive
	if newUsed >= limit {
		status = CampaignStatusExhausted
	}

	m := spanner.Update("SupplierPromotionCampaigns",
		[]string{"CampaignId", "BudgetUsedMinor", "Status"},
		[]interface{}{campaignID, newUsed, status},
	)
	return txn.BufferWrite([]*spanner.Mutation{m})
}
