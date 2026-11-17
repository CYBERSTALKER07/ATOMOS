package migration

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/migration/parser"
)

const (
	// MinAutoSortConfidence is the minimum certainty threshold below which a stream triggers review.
	MinAutoSortConfidence = 0.65
)

var (
	reINN9      = regexp.MustCompile(`^\d{9}$`)
	rePhoneUZ   = regexp.MustCompile(`^\+?998[\s\-]?\d{2}[\s\-]?\d{3}[\s\-]?\d{2}[\s\-]?\d{2}$`)
	reDateValue = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}|\d{2}[./]\d{2}[./]\d{4})`)
)

// ArtifactStreamsInput bundles an uploaded artifact row with the physical streams extracted by `migration/parser`.
type ArtifactStreamsInput struct {
	Artifact Artifact
	Streams  []parser.ParsedStream
}

// StreamOverrideItem allows users to override auto-detected packs, encoding, or ignore a stream via PATCH /sort.
type StreamOverrideItem struct {
	ArtifactID string `json:"artifact_id"`
	StreamKey  string `json:"stream_key"`
	Packs      []Pack `json:"packs,omitempty"`
	Ignore     bool   `json:"ignore,omitempty"`
	Encoding   string `json:"encoding,omitempty"`
	Confirmed  bool   `json:"confirmed,omitempty"`
}

// BundleSortOverrideRequest represents the body for PATCH /v1/supplier/migrations/projects/{projectId}/bundles/{bundleId}/sort.
type BundleSortOverrideRequest struct {
	Overrides              []StreamOverrideItem `json:"overrides,omitempty"`
	ResolvedInventoryMode  Pack                 `json:"resolved_inventory_mode,omitempty"` // stock or lots when resolving conflict
	ConfirmReview          bool                 `json:"confirm_review,omitempty"`
}

// SortedStreamDecision captures classification and split decisions for one `(artifact_id, stream_key)`.
type SortedStreamDecision struct {
	ArtifactID    string   `json:"artifact_id"`
	FileName      string   `json:"file_name"`
	StreamKey     string   `json:"stream_key"`
	Encoding      string   `json:"encoding"`
	TotalRows     int64    `json:"total_rows"`
	Headers       []string `json:"headers"`
	PreviewRaw    []string `json:"preview_raw,omitempty"`
	DetectedPacks []Pack   `json:"detected_packs"`
	SplitKeys     []string `json:"split_keys"`
	Confidence    float64  `json:"confidence"`
	IsComposite   bool     `json:"is_composite"`
	Ignored       bool     `json:"ignored,omitempty"`
	NeedsReview   bool     `json:"needs_review"`
	ReviewReason  string   `json:"review_reason,omitempty"`
}

// BundleSortSummary is persisted on `MigrationBundles.sort_summary_json` and returned by tree/sort APIs.
type BundleSortSummary struct {
	Streams                  []SortedStreamDecision `json:"streams"`
	ConflictingInventoryMode bool                   `json:"conflicting_inventory_mode"`
	NeedsReview              bool                   `json:"needs_review"`
	ReviewReasons            []string               `json:"review_reasons,omitempty"`
	ArtifactEncodings        map[string]string      `json:"artifact_encodings,omitempty"`
}

// SortDecision holds the full computed output ready for persistence by `Repository.UpsertSortedBundleSessions`.
type SortDecision struct {
	BundleStatus    BundleStatus
	Summary         BundleSortSummary
	SummaryJSON     json.RawMessage
	ArtifactStreams map[string]json.RawMessage
	Sessions        []Session
}

type streamTarget struct {
	Pack     Pack
	SplitKey string
}

// ClassifyAndSplitBundle evaluates all physical streams across a bundle's artifacts,
// splits composite sheets/structures into virtual DAG-ordered child sessions, and flags ambiguities.
func ClassifyAndSplitBundle(
	inputs []ArtifactStreamsInput,
	overrides *BundleSortOverrideRequest,
	existingSessions []Session,
	newID func() string,
) (SortDecision, error) {
	if newID == nil {
		return SortDecision{}, fmt.Errorf("newID generator is required")
	}

	overrideByStream := make(map[string]StreamOverrideItem)
	var resolvedMode Pack
	confirmReview := false
	if overrides != nil {
		for _, ov := range overrides.Overrides {
			overrideByStream[ov.ArtifactID+"::"+ov.StreamKey] = ov
		}
		if overrides.ResolvedInventoryMode == PackStock || overrides.ResolvedInventoryMode == PackLots {
			resolvedMode = overrides.ResolvedInventoryMode
		}
		confirmReview = overrides.ConfirmReview
	}

	existingByCompositeKey := make(map[string]Session)
	for _, s := range existingSessions {
		key := fmt.Sprintf("%s::%s::%s::%s", s.ArtifactID, s.StreamKey, s.Pack, s.StreamSplitKey)
		existingByCompositeKey[key] = s
	}

	summary := BundleSortSummary{
		ArtifactEncodings: make(map[string]string),
	}
	artifactStreams := make(map[string]json.RawMessage, len(inputs))
	reviewReasonSet := make(map[string]bool)

	type pendingSession struct {
		decision SortedStreamDecision
		art      Artifact
		stream   parser.ParsedStream
		target   streamTarget
	}
	var pending []pendingSession

	for _, in := range inputs {
		perArtifactList := make([]SortedStreamDecision, 0, len(in.Streams))
		for _, st := range in.Streams {
			dec, targets := classifySingleStream(in.Artifact, st)
			if ov, hasOv := overrideByStream[in.Artifact.ArtifactID+"::"+st.StreamName]; hasOv {
				if ov.Encoding != "" {
					dec.Encoding = strings.TrimSpace(ov.Encoding)
					summary.ArtifactEncodings[in.Artifact.ArtifactID] = dec.Encoding
				}
				if ov.Ignore {
					dec.Ignored = true
					dec.DetectedPacks = nil
					dec.SplitKeys = nil
					dec.NeedsReview = false
					dec.ReviewReason = ""
					targets = nil
				} else if len(ov.Packs) > 0 {
					targets = nil
					dec.DetectedPacks = nil
					dec.SplitKeys = nil
					for _, p := range ov.Packs {
						parsedPack, err := ParsePack(string(p))
						if err != nil {
							return SortDecision{}, err
						}
						targets = append(targets, streamTarget{Pack: parsedPack, SplitKey: string(parsedPack)})
						dec.DetectedPacks = append(dec.DetectedPacks, parsedPack)
						dec.SplitKeys = append(dec.SplitKeys, string(parsedPack))
					}
					dec.IsComposite = len(targets) > 1
					dec.Confidence = 1.0
					if !st.NeedsEncode || ov.Encoding != "" {
						dec.NeedsReview = false
						dec.ReviewReason = ""
					}
				}
				if ov.Confirmed || confirmReview {
					dec.NeedsReview = false
					dec.ReviewReason = ""
				}
			} else if confirmReview && dec.ReviewReason != "ambiguous_encoding" {
				dec.NeedsReview = false
				dec.ReviewReason = ""
			}

			// Apply explicit bundle-wide stock vs lots choice if provided.
			if resolvedMode != "" && !dec.Ignored {
				targets, dec = applyResolvedInventoryMode(targets, dec, resolvedMode)
			}

			if dec.NeedsReview {
				summary.NeedsReview = true
				if dec.ReviewReason != "" {
					reviewReasonSet[dec.ReviewReason] = true
				}
			}

			summary.Streams = append(summary.Streams, dec)
			perArtifactList = append(perArtifactList, dec)

			for _, tgt := range targets {
				pending = append(pending, pendingSession{
					decision: dec,
					art:      in.Artifact,
					stream:   st,
					target:   tgt,
				})
			}
		}

		rawArtStreams, err := json.Marshal(perArtifactList)
		if err != nil {
			return SortDecision{}, err
		}
		artifactStreams[in.Artifact.ArtifactID] = rawArtStreams
	}

	// Verify bundle-wide stock XOR lots invariant.
	hasStock := false
	hasLots := false
	for _, p := range pending {
		if p.target.Pack == PackStock {
			hasStock = true
		} else if p.target.Pack == PackLots {
			hasLots = true
		}
	}
	if hasStock && hasLots {
		summary.ConflictingInventoryMode = true
		summary.NeedsReview = true
		reviewReasonSet["conflicting_stock_and_lots"] = true
	}

	for r := range reviewReasonSet {
		summary.ReviewReasons = append(summary.ReviewReasons, r)
	}
	sort.Strings(summary.ReviewReasons)

	// Sort child sessions in canonical DAG topological order for operator clarity.
	packRank := make(map[Pack]int, len(TopologicalPackOrder))
	for idx, p := range TopologicalPackOrder {
		packRank[p] = idx
	}
	sort.SliceStable(pending, func(i, j int) bool {
		ri, rj := packRank[pending[i].target.Pack], packRank[pending[j].target.Pack]
		if ri != rj {
			return ri < rj
		}
		if pending[i].art.FileName != pending[j].art.FileName {
			return pending[i].art.FileName < pending[j].art.FileName
		}
		if pending[i].stream.StreamName != pending[j].stream.StreamName {
			return pending[i].stream.StreamName < pending[j].stream.StreamName
		}
		return pending[i].target.SplitKey < pending[j].target.SplitKey
	})

	sessions := make([]Session, 0, len(pending))
	for _, item := range pending {
		compKey := fmt.Sprintf("%s::%s::%s::%s", item.art.ArtifactID, item.stream.StreamName, item.target.Pack, item.target.SplitKey)
		sessID := newID()
		status := SessionStatusMappingRequired
		if prev, ok := existingByCompositeKey[compKey]; ok {
			sessID = prev.SessionID
			if prev.Status != "" && prev.Status != SessionStatusInitialized && prev.Status != SessionStatusUploaded {
				status = prev.Status
			}
		}
		sessions = append(sessions, Session{
			SupplierID:     item.art.SupplierID,
			SessionID:      sessID,
			Status:         status,
			FileName:       item.art.FileName,
			TotalRows:      item.stream.TotalRows,
			ProjectID:      item.art.ProjectID,
			BundleID:       item.art.BundleID,
			ArtifactID:     item.art.ArtifactID,
			StreamKey:      item.stream.StreamName,
			StreamSplitKey: item.target.SplitKey,
			Pack:           item.target.Pack,
			SheetName:      item.stream.StreamName,
			GCSPath:        item.art.GCSPath,
			ContentSHA256:  item.art.ContentSHA256,
			DetectedMIME:   string(item.stream.Format),
			Encoding:       item.decision.Encoding,
		})
	}

	rawSummary, err := json.Marshal(summary)
	if err != nil {
		return SortDecision{}, err
	}

	bundleStatus := BundleStatusSorted
	if summary.NeedsReview {
		bundleStatus = BundleStatusNeedsReview
	}

	return SortDecision{
		BundleStatus:    bundleStatus,
		Summary:         summary,
		SummaryJSON:     rawSummary,
		ArtifactStreams: artifactStreams,
		Sessions:        sessions,
	}, nil
}

func applyResolvedInventoryMode(targets []streamTarget, dec SortedStreamDecision, resolved Pack) ([]streamTarget, SortedStreamDecision) {
	other := PackLots
	if resolved == PackLots {
		other = PackStock
	}
	var nextTargets []streamTarget
	var nextPacks []Pack
	var nextSplits []string
	seen := make(map[string]bool)
	for _, t := range targets {
		p := t.Pack
		sk := t.SplitKey
		if p == other {
			p = resolved
			sk = string(resolved)
		}
		k := string(p) + "::" + sk
		if seen[k] {
			continue
		}
		seen[k] = true
		nextTargets = append(nextTargets, streamTarget{Pack: p, SplitKey: sk})
		nextPacks = append(nextPacks, p)
		nextSplits = append(nextSplits, sk)
	}
	dec.DetectedPacks = nextPacks
	dec.SplitKeys = nextSplits
	if dec.ReviewReason == "conflicting_stock_and_lots" {
		dec.NeedsReview = false
		dec.ReviewReason = ""
	}
	return nextTargets, dec
}

type packSignal struct {
	pack        Pack
	score       float64
	matchedKeys map[string]bool
	anchorHits  int
}

func classifySingleStream(art Artifact, st parser.ParsedStream) (SortedStreamDecision, []streamTarget) {
	dec := SortedStreamDecision{
		ArtifactID: art.ArtifactID,
		FileName:   art.FileName,
		StreamKey:  st.StreamName,
		Encoding:   st.Encoding,
		TotalRows:  st.TotalRows,
		Headers:    st.Headers,
		PreviewRaw: st.PreviewRaw,
	}

	// 1. CommerceML stream shortcuts (strong structural XML tags already separated streams).
	if st.Format == parser.FormatCommerceML {
		targets, conf := classifyCommerceMLStream(st)
		if len(targets) > 0 {
			for _, t := range targets {
				dec.DetectedPacks = append(dec.DetectedPacks, t.Pack)
				dec.SplitKeys = append(dec.SplitKeys, t.SplitKey)
			}
			dec.Confidence = conf
			dec.IsComposite = len(targets) > 1
			if st.NeedsEncode {
				dec.NeedsReview = true
				dec.ReviewReason = "ambiguous_encoding"
			}
			return dec, targets
		}
	}

	// 2. Score headers + sample value patterns against every pack taxonomy.
	signals := make(map[Pack]*packSignal, len(TopologicalPackOrder))
	for _, p := range TopologicalPackOrder {
		sig := scoreStreamForPack(p, st)
		signals[p] = &sig
	}

	// 3. Detect composite multi-domain tables cleanly before falling back to top pure pack.
	if targets, conf := detectCompositeTargets(st, signals); len(targets) >= 2 {
		for _, t := range targets {
			dec.DetectedPacks = append(dec.DetectedPacks, t.Pack)
			dec.SplitKeys = append(dec.SplitKeys, t.SplitKey)
		}
		dec.Confidence = conf
		dec.IsComposite = true
		if st.NeedsEncode {
			dec.NeedsReview = true
			dec.ReviewReason = "ambiguous_encoding"
		} else if conf < MinAutoSortConfidence {
			dec.NeedsReview = true
			dec.ReviewReason = "low_classification_confidence"
		}
		return dec, targets
	}

	// 4. Single pure pack classification.
	bestPack := Pack("")
	bestScore := -1.0
	for _, p := range TopologicalPackOrder {
		sig := signals[p]
		if sig.score > bestScore {
			bestScore = sig.score
			bestPack = p
		}
	}
	if bestScore < 0 {
		bestScore = 0
	}
	if bestScore > 1 {
		bestScore = 1
	}

	var targets []streamTarget
	if bestPack != "" && bestScore >= 0.20 {
		splitKey := string(bestPack)
		targets = []streamTarget{{Pack: bestPack, SplitKey: splitKey}}
		dec.DetectedPacks = []Pack{bestPack}
		dec.SplitKeys = []string{splitKey}
	}
	dec.Confidence = bestScore

	if st.NeedsEncode {
		dec.NeedsReview = true
		dec.ReviewReason = "ambiguous_encoding"
	} else if dec.Confidence < MinAutoSortConfidence || len(targets) == 0 {
		dec.NeedsReview = true
		dec.ReviewReason = "low_classification_confidence"
	}
	return dec, targets
}

func classifyCommerceMLStream(st parser.ParsedStream) ([]streamTarget, float64) {
	switch st.StreamName {
	case "commerceml_products":
		return []streamTarget{{Pack: PackProductsPrices, SplitKey: "products_prices"}}, 0.98
	case "commerceml_offers_stock":
		hasPrice := sampleHasNonEmptyKey(st.SampleRows, "price")
		hasQty := sampleHasNonEmptyKey(st.SampleRows, "quantity_on_hand") || sampleHasNonEmptyKey(st.SampleRows, "warehouse_id")
		switch {
		case hasPrice && hasQty:
			return []streamTarget{
				{Pack: PackProductsPrices, SplitKey: "offers_prices"},
				{Pack: PackStock, SplitKey: "offers_stock"},
			}, 0.96
		case hasQty:
			return []streamTarget{{Pack: PackStock, SplitKey: "offers_stock"}}, 0.96
		case hasPrice:
			return []streamTarget{{Pack: PackProductsPrices, SplitKey: "offers_prices"}}, 0.95
		default:
			return []streamTarget{{Pack: PackStock, SplitKey: "offers_stock"}}, 0.85
		}
	case "commerceml_customers":
		return []streamTarget{{Pack: PackCustomers, SplitKey: "customers"}}, 0.98
	case "commerceml_documents":
		hasOpen, hasClosed := inspectOrderStatuses(st.SampleRows, []string{"status", "operation"})
		switch {
		case hasOpen && hasClosed:
			return []streamTarget{
				{Pack: PackHistoricalOrders, SplitKey: "orders_historical"},
				{Pack: PackOpenOrders, SplitKey: "orders_open"},
			}, 0.90
		case hasOpen && !hasClosed:
			return []streamTarget{{Pack: PackOpenOrders, SplitKey: "orders_open"}}, 0.92
		default:
			return []streamTarget{{Pack: PackHistoricalOrders, SplitKey: "orders_historical"}}, 0.93
		}
	default:
		return nil, 0
	}
}

func scoreStreamForPack(p Pack, st parser.ParsedStream) packSignal {
	sig := packSignal{
		pack:        p,
		matchedKeys: make(map[string]bool),
	}

	var scoreSum float64
	for _, h := range st.Headers {
		cf, s := MatchHeaderScore(p, h)
		if s >= 0.70 {
			if !sig.matchedKeys[cf.Key] {
				sig.matchedKeys[cf.Key] = true
				scoreSum += s
				if cf.StrongAnchor {
					sig.anchorHits++
				}
			}
		}
	}

	// Required group satisfaction bonus (heavily separates packs that only share generic names like `sku` or `name`).
	groups := PackRequiredGroups(p)
	groupsSatisfied := 0
	for _, g := range groups {
		for _, k := range g.Fields {
			if sig.matchedKeys[k] {
				groupsSatisfied++
				break
			}
		}
	}

	coverage := 0.0
	if len(groups) > 0 {
		coverage = float64(groupsSatisfied) / float64(len(groups))
	}

	// Disambiguate open_orders vs historical_orders using delivery_date vs order_date & status values.
	if p == PackOpenOrders {
		if !sig.matchedKeys["delivery_date"] && !containsOrderStatusSignal(st, true) {
			coverage *= 0.65
		}
	}
	if p == PackHistoricalOrders && sig.matchedKeys["delivery_date"] && containsOrderStatusSignal(st, true) && !containsOrderStatusSignal(st, false) {
		coverage *= 0.65
	}

	// Sample value heuristics boost (9-digit INN, +998 phone, dates).
	valBonus := computeSampleValueBonus(p, st.SampleRows)

	// Stream name / sheet title signal.
	nameBonus := computeStreamNameBonus(p, st.StreamName)

	rawScore := coverage*0.58 + float64(sig.anchorHits)*0.11 + (scoreSum/float64(maxInt(len(st.Headers), 1)))*0.16 + valBonus + nameBonus
	if groupsSatisfied < len(groups) && nameBonus == 0 {
		rawScore *= 0.72
	}
	// Extra penalty if inventory stream has lot/expiry anchors when scoring `stock` (prefer `lots`).
	if p == PackStock && hasLotAnchors(st.Headers) {
		rawScore *= 0.40
	}
	if p == PackLots && !hasLotAnchors(st.Headers) {
		rawScore *= 0.35
	}
	if rawScore > 1.0 {
		rawScore = 1.0
	}
	sig.score = rawScore
	return sig
}

func detectCompositeTargets(st parser.ParsedStream, sigs map[Pack]*packSignal) ([]streamTarget, float64) {
	var targets []streamTarget
	var confidences []float64

	addTarget := func(p Pack, splitKey string, conf float64) {
		for _, existing := range targets {
			if existing.Pack == p && existing.SplitKey == splitKey {
				return
			}
		}
		targets = append(targets, streamTarget{Pack: p, SplitKey: splitKey})
		confidences = append(confidences, conf)
	}

	whSig := sigs[PackWarehouses]
	prodSig := sigs[PackProductsPrices]
	custSig := sigs[PackCustomers]
	stockSig := sigs[PackStock]
	lotSig := sigs[PackLots]
	histSig := sigs[PackHistoricalOrders]
	nasiyaSig := sigs[PackNasiyaAR]
	openSig := sigs[PackOpenOrders]

	hasProductCols := (prodSig.matchedKeys["sku"] || prodSig.matchedKeys["barcode"]) && prodSig.matchedKeys["name"] &&
		(prodSig.matchedKeys["price"] || prodSig.matchedKeys["retail_price"] || prodSig.matchedKeys["category"] || prodSig.matchedKeys["brand"] || prodSig.matchedKeys["unit"])
	hasStockCols := stockSig.matchedKeys["quantity_on_hand"] && (stockSig.matchedKeys["warehouse_id"] || !histSig.matchedKeys["external_order_no"])
	hasLotCols := (lotSig.matchedKeys["lot_number"] || lotSig.matchedKeys["expiry_date"]) && lotSig.matchedKeys["quantity_on_hand"]
	hasCustCols := custSig.matchedKeys["customer_name"] && (custSig.matchedKeys["inn"] || custSig.matchedKeys["phone"] || custSig.matchedKeys["credit_limit"])
	hasCustProfileCols := custSig.matchedKeys["phone"] || custSig.matchedKeys["address"] || custSig.matchedKeys["credit_limit"] || custSig.matchedKeys["district"] || custSig.matchedKeys["email"] || strings.Contains(NormalizeHeaderToken(st.StreamName), "client") || strings.Contains(NormalizeHeaderToken(st.StreamName), "customer") || strings.Contains(NormalizeHeaderToken(st.StreamName), "mijoz")
	hasNasiyaCols := nasiyaSig.matchedKeys["balance_due"] && (nasiyaSig.matchedKeys["customer_name"] || nasiyaSig.matchedKeys["customer_inn"] || nasiyaSig.matchedKeys["customer_phone"])
	hasHistOrderCols := histSig.matchedKeys["external_order_no"] && histSig.matchedKeys["total_amount"]
	hasOpenOrderCols := openSig.matchedKeys["external_order_no"] && openSig.matchedKeys["delivery_date"]
	hasWhSetupCols := whSig.matchedKeys["name"] && (whSig.matchedKeys["address"] || whSig.matchedKeys["city"] || whSig.matchedKeys["zone"] || whSig.matchedKeys["temperature_mode"])

	if hasWhSetupCols && (hasStockCols || hasLotCols) {
		addTarget(PackWarehouses, "warehouses", 0.90)
	}
	if hasProductCols && (hasStockCols || hasLotCols) {
		addTarget(PackProductsPrices, "products_prices", 0.92)
		if hasLotCols {
			addTarget(PackLots, "lots", 0.92)
		} else {
			addTarget(PackStock, "stock", 0.92)
		}
	}
	if hasCustCols && hasCustProfileCols && hasNasiyaCols {
		addTarget(PackCustomers, "customers", 0.90)
		addTarget(PackNasiyaAR, "nasiya_ar", 0.91)
	}
	if hasCustCols && (hasHistOrderCols || hasOpenOrderCols) {
		addTarget(PackCustomers, "customers", 0.88)
	}

	// Mixed status split on a single orders table (e.g. `Prodaji` containing DELIVERED + PENDING rows).
	if hasHistOrderCols || hasOpenOrderCols {
		hasOpen, hasClosed := inspectOrderStatuses(st.SampleRows, findStatusHeaderNames(st.Headers))
		if hasHistOrderCols && hasOpenOrderCols && hasOpen && hasClosed {
			addTarget(PackHistoricalOrders, "orders_historical", 0.87)
			addTarget(PackOpenOrders, "orders_open", 0.87)
		} else if len(targets) >= 1 {
			if hasOpenOrderCols || (hasOpen && !hasClosed) {
				addTarget(PackOpenOrders, "orders_open", 0.86)
			} else if hasHistOrderCols {
				addTarget(PackHistoricalOrders, "orders_historical", 0.88)
			}
		}
	}

	if len(targets) < 2 {
		return nil, 0
	}
	sum := 0.0
	for _, c := range confidences {
		sum += c
	}
	return targets, sum / float64(len(confidences))
}

func hasLotAnchors(headers []string) bool {
	for _, h := range headers {
		cf, s := MatchHeaderScore(PackLots, h)
		if s >= 0.72 && (cf.Key == "lot_number" || cf.Key == "expiry_date" || cf.Key == "production_date") {
			return true
		}
	}
	return false
}

func computeSampleValueBonus(p Pack, rows []parser.ParsedRow) float64 {
	if len(rows) == 0 {
		return 0
	}
	bonus := 0.0
	for _, r := range rows[:minInt(len(rows), 5)] {
		for k, raw := range r.Fields {
			val := strings.TrimSpace(fmt.Sprint(raw))
			if val == "" || val == "<nil>" {
				continue
			}
			keyNorm := NormalizeHeaderToken(k)
			if p == PackCustomers && reINN9.MatchString(val) && (strings.Contains(keyNorm, "inn") || strings.Contains(keyNorm, "stir") || strings.Contains(keyNorm, "инн")) {
				bonus += 0.04
			}
			if p == PackCustomers && rePhoneUZ.MatchString(val) {
				bonus += 0.03
			}
			if (p == PackLots || p == PackHistoricalOrders || p == PackOpenOrders || p == PackNasiyaAR) && reDateValue.MatchString(val) {
				bonus += 0.02
			}
		}
	}
	if bonus > 0.14 {
		bonus = 0.14
	}
	return bonus
}

func computeStreamNameBonus(p Pack, streamName string) float64 {
	n := NormalizeHeaderToken(streamName)
	if n == "" {
		return 0
	}
	hints := map[Pack][]string{
		PackWarehouses:       {"warehouse", "ombor", "sklad", "склад", "омбор"},
		PackProductsPrices:   {"product", "catalog", "price", "tovar", "nomenklatura", "товар", "номенклатур", "прайс", "narx"},
		PackCustomers:        {"customer", "client", "kontragent", "mijoz", "buyer", "клиент", "контрагент", "мижоз"},
		PackStock:            {"stock", "ostatok", "inventory", "qoldiq", "остат", "қолдиқ"},
		PackLots:             {"lot", "batch", "expiry", "partiya", "seriya", "парти", "сери"},
		PackHistoricalOrders: {"histor", "prodaj", "sales", "invoice", "realization", "продаж", "реализац", "savdo"},
		PackNasiyaAR:         {"nasiya", "qarz", "debt", "receivable", "debitor", "akt sverki", "долг", "дебитор", "қарз", "насия"},
		PackOpenOrders:       {"open order", "pending", "preorder", "ochiq", "заказ", "буюртма"},
	}
	for _, h := range hints[p] {
		if strings.Contains(n, NormalizeHeaderToken(h)) {
			return 0.14
		}
	}
	return 0
}

func findStatusHeaderNames(headers []string) []string {
	var out []string
	for _, h := range headers {
		n := NormalizeHeaderToken(h)
		if strings.Contains(n, "status") || strings.Contains(n, "state") || strings.Contains(n, "статус") || strings.Contains(n, "holat") {
			out = append(out, h)
		}
	}
	return out
}

func containsOrderStatusSignal(st parser.ParsedStream, wantOpen bool) bool {
	openHit, closedHit := inspectOrderStatuses(st.SampleRows, findStatusHeaderNames(st.Headers))
	if wantOpen {
		return openHit
	}
	return closedHit
}

func inspectOrderStatuses(rows []parser.ParsedRow, statusCols []string) (hasOpen, hasClosed bool) {
	if len(statusCols) == 0 {
		return false, false
	}
	for _, r := range rows {
		for _, col := range statusCols {
			raw, ok := r.Fields[col]
			if !ok {
				continue
			}
			v := NormalizeHeaderToken(fmt.Sprint(raw))
			switch {
			case strings.Contains(v, "open"), strings.Contains(v, "pending"), strings.Contains(v, "new"),
				strings.Contains(v, "scheduled"), strings.Contains(v, "draft"), strings.Contains(v, "kutilmoqda"),
				strings.Contains(v, "ochiq"), strings.Contains(v, "открыт"), strings.Contains(v, "новый"):
				hasOpen = true
			case strings.Contains(v, "deliver"), strings.Contains(v, "complet"), strings.Contains(v, "closed"),
				strings.Contains(v, "paid"), strings.Contains(v, "done"), strings.Contains(v, "cancel"),
				strings.Contains(v, "отгруж"), strings.Contains(v, "выполн"), strings.Contains(v, "закрыт"), strings.Contains(v, "yetkaz"):
				hasClosed = true
			}
		}
	}
	return hasOpen, hasClosed
}

func sampleHasNonEmptyKey(rows []parser.ParsedRow, key string) bool {
	for _, r := range rows {
		if v, ok := r.Fields[key]; ok && strings.TrimSpace(fmt.Sprint(v)) != "" {
			return true
		}
	}
	return false
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
