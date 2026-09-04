package retailer

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type KycDocument struct {
	DocumentID      string    `json:"document_id"`
	RetailerID      string    `json:"retailer_id"`
	Status          string    `json:"status"`
	DocumentType    string    `json:"document_type"`
	DocumentURL     string    `json:"document_url"`
	SubmittedAt     time.Time `json:"submitted_at"`
	ReviewedAt      time.Time `json:"reviewed_at,omitempty"`
	ReviewedBy      string    `json:"reviewed_by,omitempty"`
	RejectionReason string    `json:"rejection_reason,omitempty"`
}

func (r *SpannerRepository) InsertKycDocument(ctx context.Context, doc KycDocument) error {
	m := spanner.Insert("RetailerKycDocuments",
		[]string{"DocumentId", "RetailerId", "Status", "DocumentType", "DocumentUrl", "SubmittedAt", "ReviewedAt", "ReviewedBy", "RejectionReason"},
		[]interface{}{doc.DocumentID, doc.RetailerID, doc.Status, doc.DocumentType, doc.DocumentURL, spanner.CommitTimestamp, nil, nil, nil},
	)
	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	return err
}

func (r *SpannerRepository) UpdateKycDocumentStatus(ctx context.Context, docID string, status string, reviewedBy string, rejectionReason string) error {
	var reason interface{} = nil
	if rejectionReason != "" {
		reason = rejectionReason
	}
	m := spanner.Update("RetailerKycDocuments",
		[]string{"DocumentId", "Status", "ReviewedAt", "ReviewedBy", "RejectionReason"},
		[]interface{}{docID, status, spanner.CommitTimestamp, reviewedBy, reason},
	)
	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	return err
}

func (r *SpannerRepository) ListKycDocumentsByRetailer(ctx context.Context, retailerID string) ([]KycDocument, error) {
	stmt := spanner.Statement{
		SQL: `SELECT DocumentId, RetailerId, Status, DocumentType, DocumentUrl, SubmittedAt, ReviewedAt, ReviewedBy, RejectionReason 
		      FROM RetailerKycDocuments 
		      WHERE RetailerId = @retId ORDER BY SubmittedAt DESC`,
		Params: map[string]interface{}{"retId": retailerID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var docs []KycDocument
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var doc KycDocument
		var revAt spanner.NullTime
		var revBy spanner.NullString
		var rejReason spanner.NullString
		if err := row.Columns(
			&doc.DocumentID, &doc.RetailerID, &doc.Status, &doc.DocumentType, &doc.DocumentURL,
			&doc.SubmittedAt, &revAt, &revBy, &rejReason,
		); err != nil {
			return nil, err
		}
		if revAt.Valid {
			doc.ReviewedAt = revAt.Time
		}
		if revBy.Valid {
			doc.ReviewedBy = revBy.StringVal
		}
		if rejReason.Valid {
			doc.RejectionReason = rejReason.StringVal
		}
		docs = append(docs, doc)
	}
	return docs, nil
}
