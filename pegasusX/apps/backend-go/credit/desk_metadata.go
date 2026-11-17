package credit

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// SupplierDeskMetadata holds enriched retailer identity and pending credit-note
// counts for GET /v1/supplier/credit-profiles.
type SupplierDeskMetadata struct {
	RetailerName           string `json:"retailer_name,omitempty"`
	DisplayName            string `json:"display_name,omitempty"`
	TIN                    string `json:"tin,omitempty"`
	Phone                  string `json:"phone,omitempty"`
	PendingCreditNoteCount int64  `json:"pending_credit_note_count"`
}

// DeskMetadataProvider enriches supplier collections profiles with retailer metadata
// and pending credit note counts.
type DeskMetadataProvider interface {
	LoadSupplierDeskMetadata(ctx context.Context, supplierID string, retailerIDs []string) (map[string]SupplierDeskMetadata, int64, error)
}

// DeskMetadataLookup is an injectable function hook for unit tests without Spanner.
type DeskMetadataLookup func(ctx context.Context, supplierID string, retailerIDs []string) (map[string]SupplierDeskMetadata, int64, error)

// SetDeskMetadataLookup overrides desk metadata loading (tests / optional wiring).
func (s *Service) SetDeskMetadataLookup(fn DeskMetadataLookup) {
	if s == nil {
		return
	}
	s.mutMu.Lock()
	s.deskMetadataFn = fn
	s.mutMu.Unlock()
}

func (s *Service) loadDeskMetadata(ctx context.Context, supplierID string, retailerIDs []string) (map[string]SupplierDeskMetadata, int64, error) {
	s.mutMu.Lock()
	fn := s.deskMetadataFn
	s.mutMu.Unlock()
	if fn != nil {
		return fn(ctx, supplierID, retailerIDs)
	}
	if provider, ok := s.repo.(DeskMetadataProvider); ok {
		return provider.LoadSupplierDeskMetadata(ctx, supplierID, retailerIDs)
	}
	return nil, 0, nil
}

// LoadSupplierDeskMetadata implements DeskMetadataProvider directly on SpannerRepository,
// avoiding extra wiring changes in bootstrap.
func (r *SpannerRepository) LoadSupplierDeskMetadata(ctx context.Context, supplierID string, retailerIDs []string) (map[string]SupplierDeskMetadata, int64, error) {
	out := make(map[string]SupplierDeskMetadata)
	if r == nil || r.client == nil {
		return out, 0, nil
	}

	ro := r.client.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second))

	if len(retailerIDs) > 0 {
		if strings.TrimSpace(supplierID) != "" {
			linkIter := ro.Query(ctx, spanner.Statement{
				SQL: `SELECT RetailerId, COALESCE(LegalName, ''), COALESCE(NormalizedPhone, ''), COALESCE(TaxIdTIN, '')
					FROM SupplierRetailerLinks
					WHERE SupplierId = @sid AND RetailerId IN UNNEST(@rids)`,
				Params: map[string]any{"sid": supplierID, "rids": retailerIDs},
			})
			for {
				row, err := linkIter.Next()
				if err == iterator.Done || err != nil {
					break
				}
				var rid, name, phone, tin string
				if err := row.Columns(&rid, &name, &phone, &tin); err == nil {
					m := out[rid]
					m.RetailerName = firstNonEmptyCreditStr(name, m.RetailerName)
					m.DisplayName = firstNonEmptyCreditStr(name, m.DisplayName)
					m.Phone = firstNonEmptyCreditStr(phone, m.Phone)
					m.TIN = firstNonEmptyCreditStr(tin, m.TIN)
					out[rid] = m
				}
			}
			linkIter.Stop()
		}

		retIter := ro.Query(ctx, spanner.Statement{
			SQL: `SELECT RetailerId, COALESCE(Name, ''), COALESCE(Phone, '')
				FROM Retailers
				WHERE RetailerId IN UNNEST(@rids)`,
			Params: map[string]any{"rids": retailerIDs},
		})
		for {
			row, err := retIter.Next()
			if err == iterator.Done || err != nil {
				break
			}
			var rid, name, phone string
			if err := row.Columns(&rid, &name, &phone); err == nil {
				m := out[rid]
				m.RetailerName = firstNonEmptyCreditStr(m.RetailerName, name)
				m.DisplayName = firstNonEmptyCreditStr(m.DisplayName, name)
				m.Phone = firstNonEmptyCreditStr(m.Phone, phone)
				out[rid] = m
			}
		}
		retIter.Stop()
	}

	var totalPending int64
	if strings.TrimSpace(supplierID) != "" {
		draftIter := ro.Query(ctx, spanner.Statement{
			SQL: `SELECT RetailerId, COUNT(*)
				FROM DraftCreditNotes
				WHERE SupplierId = @sid AND Status = 'PENDING_APPROVAL'
				GROUP BY RetailerId`,
			Params: map[string]any{"sid": supplierID},
		})
		for {
			row, err := draftIter.Next()
			if err == iterator.Done || err != nil {
				break
			}
			var rid string
			var cnt int64
			if err := row.Columns(&rid, &cnt); err == nil && cnt > 0 {
				m := out[rid]
				m.PendingCreditNoteCount += cnt
				out[rid] = m
				totalPending += cnt
			}
		}
		draftIter.Stop()

		cnIter := ro.Query(ctx, spanner.Statement{
			SQL: `SELECT o.RetailerId, COUNT(*)
				FROM CreditNotes cn
				JOIN Orders o ON o.OrderId = cn.OrderId
				WHERE o.SupplierId = @sid AND cn.Status = 'DRAFT'
				GROUP BY o.RetailerId`,
			Params: map[string]any{"sid": supplierID},
		})
		for {
			row, err := cnIter.Next()
			if err == iterator.Done || err != nil {
				break
			}
			var rid string
			var cnt int64
			if err := row.Columns(&rid, &cnt); err == nil && cnt > 0 {
				m := out[rid]
				m.PendingCreditNoteCount += cnt
				out[rid] = m
				totalPending += cnt
			}
		}
		cnIter.Stop()
	}

	return out, totalPending, nil
}

func firstNonEmptyCreditStr(values ...string) string {
	for _, v := range values {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return ""
}
