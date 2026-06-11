package dispatch

import "context"

const fetchAllPageSize = 1000

// FetchAllDispatchable pages through the dispatch pool until exhausted.
// Used by execute paths that must consider every eligible order, not a preview slice.
func FetchAllDispatchable(ctx context.Context, repo *Repository, base FetchParams) ([]DispatchableOrder, error) {
	if repo == nil {
		return nil, nil
	}
	all := make([]DispatchableOrder, 0)
	params := base
	params.Limit = fetchAllPageSize
	params.Offset = 0
	for {
		batch, err := repo.FetchDispatchable(ctx, params)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < fetchAllPageSize {
			break
		}
		params.Offset += fetchAllPageSize
	}
	return all, nil
}
