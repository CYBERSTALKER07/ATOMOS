package migration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/migration/parser"
)

const (
	// MinRequiredMappingConfidence is the minimum confidence threshold required for every
	// mandatory canonical field before dry-run/apply can proceed.
	MinRequiredMappingConfidence = 0.80

	defaultGeminiModel       = "gemini-2.5-flash"
	defaultTargetTimeout     = 8 * time.Second
	defaultHardTimeout       = 20 * time.Second
	defaultDailyTokenBudget  = 100000
)

var (
	reRedactPhone = regexp.MustCompile(`(\+?\d[\d\s\-()]{7,16}\d)`)
	reRedactINN   = regexp.MustCompile(`\b\d{9}(\d{5})?\b`)
	reRedactEmail = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)

	// ErrAIMapperDisabled indicates AI mapping is disabled for the supplier's HomeCell.
	ErrAIMapperDisabled = errors.New("ai_mapper_disabled")
	// ErrAITokenBudgetExceeded indicates the supplier reached its daily Gemini token budget.
	ErrAITokenBudgetExceeded = errors.New("ai_mapper_daily_token_budget_exceeded")
)

// GeminiMapperConfig configures the PII-redacted Gemini schema-mapping client.
type GeminiMapperConfig struct {
	Enabled          bool
	DisabledCells    map[string]bool
	Endpoint         string
	APIKey           string
	BearerToken      string
	Model            string
	TargetTimeout    time.Duration
	HardTimeout      time.Duration
	DailyTokenBudget int
	HTTPClient       *http.Client
	Now              func() time.Time
}

// GeminiMapperConfigFromEnv builds GeminiMapperConfig from environment variables.
func GeminiMapperConfigFromEnv() GeminiMapperConfig {
	enabled := true
	if raw := strings.TrimSpace(os.Getenv("AI_MAPPER_ENABLED")); raw != "" {
		enabled = strings.EqualFold(raw, "true") || raw == "1"
	}
	disabledCells := make(map[string]bool)
	for _, part := range strings.Split(os.Getenv("AI_MAPPER_DISABLED_CELLS"), ",") {
		if c := strings.TrimSpace(part); c != "" {
			disabledCells[strings.ToLower(c)] = true
		}
	}
	model := strings.TrimSpace(os.Getenv("GEMINI_MODEL"))
	if model == "" {
		model = defaultGeminiModel
	}
	budget := defaultDailyTokenBudget
	if raw := strings.TrimSpace(os.Getenv("AI_MAPPER_DAILY_TOKEN_BUDGET")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
			budget = n
		}
	}
	apiKey := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	endpoint := strings.TrimSpace(os.Getenv("GEMINI_ENDPOINT"))
	if endpoint == "" && apiKey != "" {
		endpoint = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", model)
	}
	return GeminiMapperConfig{
		Enabled:          enabled,
		DisabledCells:    disabledCells,
		Endpoint:         endpoint,
		APIKey:           apiKey,
		BearerToken:      strings.TrimSpace(os.Getenv("VERTEX_ACCESS_TOKEN")),
		Model:            model,
		TargetTimeout:    defaultTargetTimeout,
		HardTimeout:      defaultHardTimeout,
		DailyTokenBudget: budget,
	}
}

// AIMapper proposes canonical schema mappings using mapping library cache -> deterministic+heuristic ->
// PII-redacted Gemini (when enabled and within daily tenant token budget), falling back safely on errors.
type AIMapper struct {
	cfg      GeminiMapperConfig
	client   *http.Client
	now      func() time.Time
	mu       sync.Mutex
	usageDay string
	usedByID map[string]int
}

// NewAIMapper constructs a bounded AIMapper with per-tenant daily quota tracking.
func NewAIMapper(cfg GeminiMapperConfig) *AIMapper {
	if cfg.Model == "" {
		cfg.Model = defaultGeminiModel
	}
	if cfg.TargetTimeout <= 0 {
		cfg.TargetTimeout = defaultTargetTimeout
	}
	if cfg.HardTimeout <= 0 {
		cfg.HardTimeout = defaultHardTimeout
	}
	if cfg.HardTimeout < cfg.TargetTimeout {
		cfg.HardTimeout = cfg.TargetTimeout
	}
	if cfg.DailyTokenBudget <= 0 {
		cfg.DailyTokenBudget = defaultDailyTokenBudget
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.HardTimeout}
	}
	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = func() time.Time { return time.Now().UTC() }
	}
	return &AIMapper{
		cfg:      cfg,
		client:   httpClient,
		now:      nowFn,
		usedByID: make(map[string]int),
	}
}

// IsEnabledForCell checks the AI_MAPPER_ENABLED switch and HomeCell blocklist.
func (m *AIMapper) IsEnabledForCell(homeCell string) bool {
	if m == nil || !m.cfg.Enabled {
		return false
	}
	if m.cfg.DisabledCells[strings.ToLower(strings.TrimSpace(homeCell))] {
		return false
	}
	return m.cfg.Endpoint != "" && (m.cfg.APIKey != "" || m.cfg.BearerToken != "")
}

// CheckAndConsumeBudget enforces per-tenant daily token quota before issuing an LLM request.
func (m *AIMapper) CheckAndConsumeBudget(supplierID string, estTokens int) error {
	if m == nil {
		return ErrAIMapperDisabled
	}
	today := m.now().UTC().Format("2006-01-02")
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.usageDay != today {
		m.usageDay = today
		m.usedByID = make(map[string]int)
	}
	cur := m.usedByID[supplierID]
	if cur+estTokens > m.cfg.DailyTokenBudget {
		return fmt.Errorf("%w: used=%d limit=%d", ErrAITokenBudgetExceeded, cur, m.cfg.DailyTokenBudget)
	}
	m.usedByID[supplierID] = cur + estTokens
	return nil
}

// ComputeHeaderFingerprint computes a stable 64-char hex digest of a pack's normalized header set.
func ComputeHeaderFingerprint(pack Pack, headers []string) string {
	seen := make(map[string]bool, len(headers))
	parts := make([]string, 0, len(headers))
	for _, h := range headers {
		n := NormalizeHeaderToken(h)
		if n != "" && !seen[n] {
			seen[n] = true
			parts = append(parts, n)
		}
	}
	sort.Strings(parts)
	sum := sha256.Sum256([]byte(string(pack) + "|" + strings.Join(parts, "\x1f")))
	return hex.EncodeToString(sum[:])
}

// RedactSampleRows hashes phone numbers, 9/14-digit tax IDs/PINFLs, and emails in up to 20 sample rows
// so raw tenant customer PII never leaves PegasusX boundaries for Gemini prompts.
func RedactSampleRows(rows []parser.ParsedRow) []map[string]string {
	limit := len(rows)
	if limit > 20 {
		limit = 20
	}
	out := make([]map[string]string, 0, limit)
	for i := 0; i < limit; i++ {
		redacted := make(map[string]string, len(rows[i].Fields))
		for k, v := range rows[i].Fields {
			redacted[k] = RedactPIIString(fmt.Sprint(v))
		}
		out = append(out, redacted)
	}
	return out
}

// RedactPIIString masks and deterministically hashes emails, 9/14-digit INNs/PINFLs, and phone numbers.
func RedactPIIString(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" || s == "<nil>" {
		return ""
	}
	s = reRedactEmail.ReplaceAllStringFunc(s, func(match string) string {
		return "REDACTED_EMAIL:" + shortHash8(match)
	})
	s = reRedactINN.ReplaceAllStringFunc(s, func(match string) string {
		return "REDACTED_INN:" + shortHash8(match)
	})
	s = reRedactPhone.ReplaceAllStringFunc(s, func(match string) string {
		digits := digitsOnly(match)
		if len(digits) < 7 {
			return match
		}
		return "REDACTED_PHONE:" + shortHash8(digits)
	})
	return s
}

func shortHash8(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])[:8]
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ProposeMappings resolves column mappings using:
// 1. Confirmed SupplierImportMappingLibrary hit (instant 0.99 match)
// 2. Deterministic/multilingual header matcher
// 3. PII-redacted Gemini schema mapper (if enabled + budget available), with safe heuristic fallback on timeout/error.
func (m *AIMapper) ProposeMappings(
	ctx context.Context,
	supplierID, homeCell, sessionID string,
	pack Pack,
	headers []string,
	sampleRows []parser.ParsedRow,
	libraryEntry *MappingLibraryEntry,
) SessionMappingDocument {
	fingerprint := ComputeHeaderFingerprint(pack, headers)
	now := m.now()

	doc := SessionMappingDocument{
		SessionID:         sessionID,
		Pack:              pack,
		HeaderFingerprint: fingerprint,
		UpdatedAt:         now,
	}

	// Step 1: Reusable confirmed mapping library lookup by (supplier_id, pack, header_fingerprint).
	if libraryEntry != nil && len(libraryEntry.Mappings) > 0 {
		matched := applyLibraryProposals(headers, libraryEntry.Mappings)
		if len(matched) > 0 {
			doc.Mappings = matched
			doc.AIModel = "mapping-library-v1"
			doc.AIStatus = "LIBRARY_HIT"
			doc.AllRequiredMapped, doc.MissingRequired, _ = EvaluateRequiredMappings(pack, doc.Mappings)
			doc.MappingHash = ComputeMappingHash(doc)
			return doc
		}
	}

	// Step 2: Deterministic + heuristic matching across all pack canonical fields.
	proposals := deterministicAndHeuristicProposals(pack, headers, sampleRows)
	usedTargets := make(map[string]bool)
	unresolved := make([]string, 0)
	for _, p := range proposals {
		if p.TargetField != "" && p.Confidence >= MinRequiredMappingConfidence {
			usedTargets[p.TargetField] = true
		} else {
			unresolved = append(unresolved, p.SourceColumn)
		}
	}

	doc.AIModel = "heuristic-v1"
	doc.AIStatus = "HEURISTIC"

	// Step 3: Call Gemini on unresolved columns (or when required fields remain missing) if enabled & budgeted.
	allReqDet, _, _ := EvaluateRequiredMappings(pack, proposals)
	if len(unresolved) > 0 || !allReqDet {
		if !m.IsEnabledForCell(homeCell) {
			doc.AIStatus = "FALLBACK_DISABLED"
			doc.FallbackReason = "ai_mapper_disabled"
		} else {
			redactedSamples := RedactSampleRows(sampleRows)
			estTokens := 350 + len(headers)*25 + len(redactedSamples)*30
			if budgetErr := m.CheckAndConsumeBudget(supplierID, estTokens); budgetErr != nil {
				doc.AIStatus = "FALLBACK_QUOTA"
				doc.FallbackReason = budgetErr.Error()
			} else {
				aiProposals, tokensUsed, aiErr := m.invokeGemini(ctx, pack, headers, unresolved, redactedSamples)
				doc.TokensUsed = tokensUsed
				if aiErr != nil {
					if errors.Is(aiErr, context.DeadlineExceeded) {
						doc.AIStatus = "FALLBACK_TIMEOUT"
					} else {
						doc.AIStatus = "FALLBACK_ERROR"
					}
					doc.FallbackReason = aiErr.Error()
				} else {
					doc.AIModel = m.cfg.Model
					doc.AIStatus = "GEMINI_PROPOSED"
					mergeGeminiProposals(pack, &proposals, aiProposals, usedTargets)
				}
			}
		}
	} else {
		doc.AIModel = "deterministic-v1"
		doc.AIStatus = "DETERMINISTIC"
	}

	doc.Mappings = proposals
	doc.AllRequiredMapped, doc.MissingRequired, _ = EvaluateRequiredMappings(pack, doc.Mappings)
	doc.MappingHash = ComputeMappingHash(doc)
	return doc
}

func applyLibraryProposals(headers []string, lib []ColumnMappingProposal) []ColumnMappingProposal {
	byColNorm := make(map[string]ColumnMappingProposal, len(lib))
	for _, m := range lib {
		if m.TargetField == "" {
			continue
		}
		byColNorm[NormalizeHeaderToken(m.SourceColumn)] = m
	}
	out := make([]ColumnMappingProposal, 0, len(headers))
	hitCount := 0
	for _, h := range headers {
		if found, ok := byColNorm[NormalizeHeaderToken(h)]; ok {
			hitCount++
			out = append(out, ColumnMappingProposal{
				SourceColumn: h,
				TargetField:  found.TargetField,
				Confidence:   0.99,
				Reason:       "Matched confirmed tenant mapping library",
				Source:       "library",
				Confirmed:    found.Confirmed,
				Transform:    found.Transform,
			})
		} else {
			out = append(out, ColumnMappingProposal{
				SourceColumn: h,
				Confidence:   0,
				Source:       "library",
			})
		}
	}
	if hitCount == 0 {
		return nil
	}
	return out
}

func deterministicAndHeuristicProposals(pack Pack, headers []string, sampleRows []parser.ParsedRow) []ColumnMappingProposal {
	taxonomy := PackTaxonomy(pack)
	validTargets := make(map[string]CanonicalField, len(taxonomy))
	for _, f := range taxonomy {
		validTargets[f.Key] = f
	}

	candidates := make([]ColumnMappingProposal, 0, len(headers))
	claimedTarget := make(map[string]float64)

	for _, h := range headers {
		cf, score := MatchHeaderScore(pack, h)
		reason := ""
		source := "deterministic"
		if score < 0.70 {
			// Value-shape heuristic fallback when header text itself is cryptic (e.g., Col_1, A, C_3).
			if inferredKey, infConf, infReason := inferFieldFromSamples(pack, h, sampleRows); infConf > score {
				if f, ok := validTargets[inferredKey]; ok {
					cf = f
					score = infConf
					reason = infReason
					source = "heuristic"
				}
			}
		} else {
			reason = fmt.Sprintf("Matched multilingual %s alias (%s)", pack, cf.Key)
		}

		// Avoid mapping two columns to the exact same canonical field unless the earlier match was weaker.
		targetKey := cf.Key
		if targetKey != "" {
			if prevScore, taken := claimedTarget[targetKey]; taken {
				if score <= prevScore {
					targetKey = ""
					score = 0
					reason = "Duplicate candidate suppressed"
				}
			}
			if targetKey != "" {
				claimedTarget[targetKey] = score
			}
		}

		candidates = append(candidates, ColumnMappingProposal{
			SourceColumn: h,
			TargetField:  targetKey,
			Confidence:   score,
			Reason:       reason,
			Source:       source,
		})
	}
	return candidates
}

func inferFieldFromSamples(pack Pack, header string, rows []parser.ParsedRow) (string, float64, string) {
	if len(rows) == 0 {
		return "", 0, ""
	}
	sample := rows
	if len(sample) > 10 {
		sample = sample[:10]
	}
	innHits, phoneHits, dateHits := 0, 0, 0
	nonEmptyCount := 0
	for _, r := range sample {
		raw, ok := r.Fields[header]
		if !ok {
			continue
		}
		v := strings.TrimSpace(fmt.Sprint(raw))
		if v == "" || v == "<nil>" {
			continue
		}
		nonEmptyCount++
		switch {
		case reINN9.MatchString(v):
			innHits++
		case rePhoneUZ.MatchString(v):
			phoneHits++
		case reDateValue.MatchString(v):
			dateHits++
		}
	}
	if nonEmptyCount == 0 {
		return "", 0, ""
	}
	if innHits*2 >= nonEmptyCount {
		switch pack {
		case PackCustomers:
			return "inn", 0.85, "Sample values match 9-digit INN"
		case PackHistoricalOrders, PackNasiyaAR, PackOpenOrders:
			return "customer_inn", 0.85, "Sample values match 9-digit customer INN"
		}
	}
	if phoneHits*2 >= nonEmptyCount {
		switch pack {
		case PackCustomers:
			return "phone", 0.86, "Sample values match Uzbekistan +998 phone format"
		case PackHistoricalOrders, PackNasiyaAR, PackOpenOrders:
			return "customer_phone", 0.86, "Sample values match Uzbekistan +998 phone format"
		}
	}
	if dateHits*2 >= nonEmptyCount {
		switch pack {
		case PackLots:
			return "expiry_date", 0.72, "Sample values match date pattern"
		case PackHistoricalOrders:
			return "order_date", 0.72, "Sample values match date pattern"
		case PackOpenOrders:
			return "delivery_date", 0.72, "Sample values match date pattern"
		case PackNasiyaAR:
			return "due_date", 0.72, "Sample values match date pattern"
		}
	}
	return "", 0, ""
}

type geminiPromptPayload struct {
	Pack           Pack                `json:"pack"`
	TargetSchema   []CanonicalField    `json:"target_schema"`
	Headers        []string            `json:"headers"`
	UnresolvedCols []string            `json:"unresolved_columns"`
	SampleRows     []map[string]string `json:"redacted_sample_rows"`
}

type geminiMappingResponse struct {
	Mappings []struct {
		SourceColumn string  `json:"source_column"`
		TargetField  string  `json:"target_field"`
		Confidence   float64 `json:"confidence"`
		Reason       string  `json:"reason"`
	} `json:"mappings"`
}

func (m *AIMapper) invokeGemini(
	ctx context.Context,
	pack Pack,
	headers []string,
	unresolved []string,
	redactedSamples []map[string]string,
) ([]ColumnMappingProposal, int, error) {
	reqCtx, cancel := context.WithTimeout(ctx, m.cfg.TargetTimeout)
	defer cancel()

	// Cap hard timeout defensively in case TargetTimeout was raised externally.
	hardCtx, cancelHard := context.WithTimeout(reqCtx, m.cfg.HardTimeout)
	defer cancelHard()

	promptObj := geminiPromptPayload{
		Pack:           pack,
		TargetSchema:   PackTaxonomy(pack),
		Headers:        headers,
		UnresolvedCols: unresolved,
		SampleRows:     redactedSamples,
	}
	promptBytes, err := json.Marshal(promptObj)
	if err != nil {
		return nil, 0, err
	}

	instruction := "Map CSV/Excel headers from an Uzbekistan/CIS distributor ERP export (Russian, Uzbek Latin/Cyrillic, English, 1C, Billz, MoySklad, Smartup) to target_schema keys. Respond ONLY with JSON matching {\"mappings\":[{\"source_column\":string,\"target_field\":string,\"confidence\":number,\"reason\":string}]}. Confidence must be 0.0 to 1.0."
	bodyObj := map[string]any{
		"contents": []map[string]any{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": instruction + "\nINPUT_JSON:\n" + string(promptBytes)},
				},
			},
		},
		"generationConfig": map[string]any{
			"temperature":      0.0,
			"responseMimeType": "application/json",
		},
	}
	reqBody, err := json.Marshal(bodyObj)
	if err != nil {
		return nil, 0, err
	}

	endpoint := m.cfg.Endpoint
	if m.cfg.APIKey != "" && !strings.Contains(endpoint, "key=") {
		sep := "?"
		if strings.Contains(endpoint, "?") {
			sep = "&"
		}
		endpoint += sep + "key=" + m.cfg.APIKey
	}

	httpReq, err := http.NewRequestWithContext(hardCtx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, 0, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if m.cfg.BearerToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+m.cfg.BearerToken)
	}

	resp, err := m.client.Do(httpReq)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	rawResp, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, 0, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, 0, fmt.Errorf("gemini http %d: %s", resp.StatusCode, strings.TrimSpace(string(rawResp)))
	}

	parsed, tokensUsed, err := decodeGeminiResponse(rawResp)
	if err != nil {
		return nil, tokensUsed, err
	}

	out := make([]ColumnMappingProposal, 0, len(parsed.Mappings))
	for _, item := range parsed.Mappings {
		conf := item.Confidence
		if conf < 0 {
			conf = 0
		}
		if conf > 1 {
			conf = 1
		}
		out = append(out, ColumnMappingProposal{
			SourceColumn: item.SourceColumn,
			TargetField:  strings.TrimSpace(item.TargetField),
			Confidence:   conf,
			Reason:       strings.TrimSpace(item.Reason),
			Source:       "gemini",
		})
	}
	return out, tokensUsed, nil
}

func decodeGeminiResponse(raw []byte) (geminiMappingResponse, int, error) {
	// Support both native Gemini GenerateContentResponse envelope and simplified mock JSON `{"mappings":[...]}`.
	var direct geminiMappingResponse
	if err := json.Unmarshal(raw, &direct); err == nil && len(direct.Mappings) > 0 {
		return direct, 250, nil
	}
	var env struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			TotalTokenCount int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return geminiMappingResponse{}, 0, fmt.Errorf("decode gemini envelope: %w", err)
	}
	if len(env.Candidates) == 0 || len(env.Candidates[0].Content.Parts) == 0 {
		return geminiMappingResponse{}, env.UsageMetadata.TotalTokenCount, fmt.Errorf("gemini returned no candidates")
	}
	text := strings.TrimSpace(env.Candidates[0].Content.Parts[0].Text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var mapped geminiMappingResponse
	if err := json.Unmarshal([]byte(text), &mapped); err != nil {
		return geminiMappingResponse{}, env.UsageMetadata.TotalTokenCount, fmt.Errorf("decode gemini candidate json: %w", err)
	}
	tokens := env.UsageMetadata.TotalTokenCount
	if tokens <= 0 {
		tokens = 250
	}
	return mapped, tokens, nil
}

func mergeGeminiProposals(pack Pack, base *[]ColumnMappingProposal, ai []ColumnMappingProposal, usedTargets map[string]bool) {
	validField := make(map[string]bool)
	for _, f := range PackTaxonomy(pack) {
		validField[f.Key] = true
	}
	aiByCol := make(map[string]ColumnMappingProposal, len(ai))
	for _, item := range ai {
		if validField[item.TargetField] {
			aiByCol[item.SourceColumn] = item
		}
	}
	// Sort column updates deterministically so duplicate suppression is reproducible.
	indices := make([]int, len(*base))
	for i := range indices {
		indices[i] = i
	}
	sort.Ints(indices)
	for _, i := range indices {
		cur := &(*base)[i]
		sug, ok := aiByCol[cur.SourceColumn]
		if !ok || sug.TargetField == "" {
			continue
		}
		if cur.Confidence >= sug.Confidence && cur.TargetField != "" {
			continue
		}
		if usedTargets[sug.TargetField] && cur.TargetField != sug.TargetField {
			continue
		}
		usedTargets[sug.TargetField] = true
		cur.TargetField = sug.TargetField
		cur.Confidence = sug.Confidence
		cur.Reason = sug.Reason
		cur.Source = "gemini"
	}
}
