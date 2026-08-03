package claims

import (
	"context"

	"cloud.google.com/go/spanner"
)

type Repository interface {
	SaveClaim(ctx context.Context, claim *Claim, evidences []*ClaimEvidence) error
}

type spannerRepository struct {
	client *spanner.Client
}

func NewSpannerRepository(client *spanner.Client) Repository {
	return &spannerRepository{client: client}
}

func (r *spannerRepository) SaveClaim(ctx context.Context, claim *Claim, evidences []*ClaimEvidence) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{}
		
		claimMut, err := spanner.InsertOrUpdateStruct("Claims", claim)
		if err != nil {
			return err
		}
		mutations = append(mutations, claimMut)

		for _, ev := range evidences {
			evMut, err := spanner.InsertOrUpdateStruct("ClaimEvidences", ev)
			if err != nil {
				return err
			}
			mutations = append(mutations, evMut)
		}

		return txn.BufferWrite(mutations)
	})
	return err
}

