package globalproducts

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/gs1"
)

// Service implements GlobalProducts matching and APIs.
type Service struct {
	repo Repository
	log  *slog.Logger
}

func NewService(repo Repository, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{repo: repo, log: log}
}

// EnsureBootstrap seeds UoM hierarchy when enabled.
func (s *Service) EnsureBootstrap(ctx context.Context) error {
	if !Enabled() {
		return nil
	}
	return s.repo.EnsureStandardUoM(ctx)
}

// OnProductUpserted is the catalog hook: match/link when flag on.
func (s *Service) OnProductUpserted(ctx context.Context, in ProductInput) error {
	if !Enabled() {
		return nil
	}
	_, err := s.MatchAndLink(ctx, in)
	return err
}

// MatchResult summarizes what MatchAndLink did.
type MatchResult struct {
	GlobalProductID string   `json:"global_product_id,omitempty"`
	Method          string   `json:"method,omitempty"`
	Queued          bool     `json:"queued,omitempty"`
	QueueIDs        []string `json:"queue_ids,omitempty"`
	Created         bool     `json:"created,omitempty"`
}

// MatchAndLink runs exact GTIN → fuzzy → create pipeline.
func (s *Service) MatchAndLink(ctx context.Context, in ProductInput) (MatchResult, error) {
	if err := s.repo.EnsureStandardUoM(ctx); err != nil {
		return MatchResult{}, err
	}
	pack := in.UnitsPerPack
	if pack <= 0 {
		pack = 1
	}
	uom := strings.ToUpper(strings.TrimSpace(in.UomCode))
	if uom == "" {
		uom = "EACH"
	}
	brand := strings.TrimSpace(in.Brand)
	if brand == "" {
		brand = firstToken(in.Name)
	}
	baseUom := UomEachID
	switch uom {
	case "INNER":
		baseUom = UomInnerID
	case "CASE":
		baseUom = UomCaseID
	case "PALLET":
		baseUom = UomPalletID
	}

	gtin := ""
	if raw := strings.TrimSpace(in.Barcode); raw != "" {
		if n, err := gs1.NormalizeGTIN(raw); err == nil {
			gtin = n
		}
		// Invalid GTIN: skip exact path but still allow fuzzy/create without GTIN.
	}

	if gtin != "" {
		if existing, err := s.repo.GetByGtin(ctx, gtin); err != nil {
			return MatchResult{}, err
		} else if existing != nil {
			if err := s.linkOffer(ctx, in, existing.GlobalProductID); err != nil {
				return MatchResult{}, err
			}
			return MatchResult{GlobalProductID: existing.GlobalProductID, Method: MethodExactGTIN}, nil
		}
	}

	// Fuzzy against existing masters (by normalized key + score scan).
	key := BuildNormalizedKey(brand, in.Name, pack, uom)
	candidates, err := s.collectFuzzyCandidates(ctx, brand, in.Name, pack, uom, key)
	if err != nil {
		return MatchResult{}, err
	}
	auto, queue := DecideFuzzy(candidates)
	if auto != nil {
		if err := s.linkOffer(ctx, in, auto.GlobalProductID); err != nil {
			return MatchResult{}, err
		}
		return MatchResult{GlobalProductID: auto.GlobalProductID, Method: MethodFuzzy}, nil
	}
	if len(queue) > 0 {
		var qids []string
		for _, c := range queue {
			qid := uuid.NewString()
			if err := s.repo.EnqueueMatch(ctx, MatchQueueItem{
				QueueID:                  qid,
				SupplierID:               in.SupplierID,
				ProductID:                in.ProductID,
				CandidateGlobalProductID: c.GlobalProductID,
				MatchMethod:              MethodFuzzy,
				Score:                    c.Score,
				Status:                   StatusPending,
				Reason:                   "ambiguous_fuzzy_match",
			}); err != nil {
				return MatchResult{}, err
			}
			qids = append(qids, qid)
		}
		return MatchResult{Queued: true, QueueIDs: qids, Method: MethodFuzzy}, nil
	}

	// Create new global product.
	gpID := uuid.NewString()
	gp := GlobalProduct{
		GlobalProductID: gpID,
		Gtin:            gtin,
		Brand:           brand,
		Name:            in.Name,
		PackQty:         pack,
		BaseUomID:       baseUom,
		NormalizedKey:   key,
		Version:         1,
	}
	if err := s.repo.UpsertGlobal(ctx, gp); err != nil {
		return MatchResult{}, err
	}
	if err := s.linkOffer(ctx, in, gpID); err != nil {
		return MatchResult{}, err
	}
	method := MethodManual
	if gtin != "" {
		method = MethodExactGTIN
	}
	return MatchResult{GlobalProductID: gpID, Method: method, Created: true}, nil
}

func (s *Service) collectFuzzyCandidates(ctx context.Context, brand, name string, pack int64, uom, key string) ([]scoredCandidate, error) {
	byKey, err := s.repo.ListByNormalizedKey(ctx, key)
	if err != nil {
		return nil, err
	}
	all, err := s.repo.ListAll(ctx, 500)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var scored []scoredCandidate
	add := func(gp GlobalProduct) {
		if seen[gp.GlobalProductID] {
			return
		}
		seen[gp.GlobalProductID] = true
		gpUom := "EACH"
		switch gp.BaseUomID {
		case UomInnerID:
			gpUom = "INNER"
		case UomCaseID:
			gpUom = "CASE"
		case UomPalletID:
			gpUom = "PALLET"
		}
		sc := FuzzyScore(brand, name, pack, uom, gp.Brand, gp.Name, gp.PackQty, gpUom)
		if sc > 0 {
			scored = append(scored, scoredCandidate{GlobalProductID: gp.GlobalProductID, Score: sc})
		}
	}
	for _, gp := range byKey {
		add(gp)
	}
	for _, gp := range all {
		add(gp)
	}
	// sort desc
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].Score > scored[i].Score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}
	return scored, nil
}

func (s *Service) linkOffer(ctx context.Context, in ProductInput, globalID string) error {
	return s.repo.UpsertOffer(ctx, Offer{
		SupplierID:      in.SupplierID,
		ProductID:       in.ProductID,
		GlobalProductID: globalID,
		PriceMinor:      in.PriceMinor,
		Currency:        offerCurrency(ctx, in.SupplierID, in.Currency),
		Moq:             1,
		LeadTimeDays:    0,
		Status:          StatusLinked,
		Version:         1,
	})
}

func firstToken(name string) string {
	parts := strings.Fields(strings.TrimSpace(name))
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func (s *Service) GetGlobal(ctx context.Context, id string) (*GlobalProduct, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListOffers(ctx context.Context, globalID string) ([]Offer, error) {
	return s.repo.ListOffersByGlobal(ctx, globalID)
}

func (s *Service) ListMatchQueue(ctx context.Context, status string, limit int) ([]MatchQueueItem, error) {
	if status == "" {
		status = StatusPending
	}
	return s.repo.ListMatchQueue(ctx, status, limit)
}

// ResolveMatch accepts or rejects a queue item; accept links the offer.
func (s *Service) ResolveMatch(ctx context.Context, queueID, decision, actorSupplierID string, forceGlobalID string) error {
	item, err := s.repo.GetMatchQueueItem(ctx, queueID)
	if err != nil {
		return err
	}
	if item == nil {
		return fmt.Errorf("queue item not found")
	}
	if actorSupplierID != "" && item.SupplierID != actorSupplierID {
		return fmt.Errorf("forbidden: not owner of queue item")
	}
	dec := strings.ToUpper(strings.TrimSpace(decision))
	switch dec {
	case "ACCEPT", "ACCEPTED":
		gid := forceGlobalID
		if gid == "" {
			gid = item.CandidateGlobalProductID
		}
		if gid == "" {
			return fmt.Errorf("candidate_global_product_id required")
		}
		if err := s.repo.UpsertOffer(ctx, Offer{
			SupplierID:      item.SupplierID,
			ProductID:       item.ProductID,
			GlobalProductID: gid,
			PriceMinor:      0,
			Currency:        offerCurrency(ctx, item.SupplierID, ""),
			Moq:             1,
			Status:          StatusLinked,
			Version:         1,
		}); err != nil {
			return err
		}
		item.Status = StatusAccepted
		item.CandidateGlobalProductID = gid
		return s.repo.UpdateMatchQueue(ctx, *item)
	case "REJECT", "REJECTED":
		item.Status = StatusRejected
		return s.repo.UpdateMatchQueue(ctx, *item)
	default:
		return fmt.Errorf("decision must be ACCEPT or REJECT")
	}
}

// LinkExplicit forces a product onto an existing or new global product (supplier API).
func (s *Service) LinkExplicit(ctx context.Context, in ProductInput, globalProductID string) (MatchResult, error) {
	if !Enabled() {
		return MatchResult{}, fmt.Errorf("global products disabled")
	}
	if globalProductID != "" {
		gp, err := s.repo.GetByID(ctx, globalProductID)
		if err != nil {
			return MatchResult{}, err
		}
		if gp == nil {
			return MatchResult{}, fmt.Errorf("global product not found")
		}
		if err := s.linkOffer(ctx, in, globalProductID); err != nil {
			return MatchResult{}, err
		}
		return MatchResult{GlobalProductID: globalProductID, Method: MethodManual}, nil
	}
	return s.MatchAndLink(ctx, in)
}

// offerCurrency is empty-currency law for catalog offers: stored ISO code, else
// the shipped pack. Planned/unknown packs stay empty — never invent UZS.
func offerCurrency(ctx context.Context, supplierID, stored string) string {
	c, err := auth.CoalesceCurrency(ctx, supplierID, stored)
	if err != nil {
		return strings.ToUpper(strings.TrimSpace(stored))
	}
	return c
}
