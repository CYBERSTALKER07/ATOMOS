package controltower

import (
	"sort"
	"strings"
	"time"
)

const (
	scoreAmountCapMinor   = 10_000_000 // 10M tiyin
	scoreAmountWeight     = 40
	scoreAgeWeightPerHour = 2
	scoreAgeCap           = 48 // hours
	scoreSeverityWeight   = 30
	scoreSegmentBoost     = 15
)

var severityRank = map[string]int64{
	"CRITICAL": 4,
	"HIGH":     3,
	"MEDIUM":   2,
	"LOW":      1,
}

// ScoredException is the wire model for ranked open exceptions.
type ScoredException struct {
	ExceptionID            string    `json:"exception_id"`
	Type                   string    `json:"type"`
	Severity               string    `json:"severity"`
	OrderID                string    `json:"order_id,omitempty"`
	RetailerID             string    `json:"retailer_id,omitempty"`
	AmountMinor            int64     `json:"amount_minor,omitempty"`
	Score                  int64     `json:"score"`
	SeverityRank           int64     `json:"severity_rank"`
	AgeMinutes             int64     `json:"age_minutes"`
	RetailerSegment        string    `json:"retailer_segment,omitempty"`
	RecommendedPlaybookIDs []string  `json:"recommended_playbook_ids,omitempty"`
	TopPlaybookName        string    `json:"top_playbook_name,omitempty"`
	CreatedAt              time.Time `json:"created_at"`
}

// ComputeExceptionScore returns a deterministic priority score for ranking.
func ComputeExceptionScore(ex Exception, now time.Time) (score int64, severityRankVal int64, ageMinutes int64) {
	amount := ex.AmountMinor
	if amount > scoreAmountCapMinor {
		amount = scoreAmountCapMinor
	}
	amountPart := (amount * scoreAmountWeight) / scoreAmountCapMinor

	sev := strings.ToUpper(strings.TrimSpace(ex.Severity))
	rank := severityRank[sev]
	if rank == 0 {
		rank = 1
	}
	severityPart := rank * scoreSeverityWeight

	ageMin := int64(now.Sub(ex.CreatedAt).Minutes())
	if ageMin < 0 {
		ageMin = 0
	}
	ageHours := ageMin / 60
	if ageHours > scoreAgeCap {
		ageHours = scoreAgeCap
	}
	agePart := ageHours * scoreAgeWeightPerHour

	segmentPart := int64(0)
	if strings.EqualFold(strings.TrimSpace(ex.RetailerSegment), "A") {
		segmentPart = scoreSegmentBoost
	}

	total := amountPart + severityPart + agePart + segmentPart
	return total, rank, ageMin
}

// MatchPlaybooks returns playbooks that match the exception, sorted by priority desc.
func MatchPlaybooks(ex Exception, playbooks []Playbook, now time.Time) []Playbook {
	matched := make([]Playbook, 0, len(playbooks))
	for _, pb := range playbooks {
		if !pb.IsActive {
			continue
		}
		if pb.MatchRules.MatchesException(ex, now) {
			matched = append(matched, pb)
		}
	}
	sort.Slice(matched, func(i, j int) bool {
		if matched[i].Priority != matched[j].Priority {
			return matched[i].Priority > matched[j].Priority
		}
		return matched[i].PlaybookID < matched[j].PlaybookID
	})
	return matched
}

func buildScoredException(ex Exception, playbooks []Playbook, now time.Time) ScoredException {
	score, rank, ageMin := ComputeExceptionScore(ex, now)
	matched := MatchPlaybooks(ex, playbooks, now)
	ids := make([]string, 0, len(matched))
	topName := ""
	for i, pb := range matched {
		ids = append(ids, pb.PlaybookID)
		if i == 0 {
			topName = pb.Name
		}
	}
	return ScoredException{
		ExceptionID:            ex.ExceptionID,
		Type:                   ex.Type,
		Severity:               ex.Severity,
		OrderID:                ex.OrderID,
		RetailerID:             ex.RetailerID,
		AmountMinor:            ex.AmountMinor,
		Score:                  score,
		SeverityRank:           rank,
		AgeMinutes:             ageMin,
		RetailerSegment:        ex.RetailerSegment,
		RecommendedPlaybookIDs: ids,
		TopPlaybookName:        topName,
		CreatedAt:              ex.CreatedAt,
	}
}
