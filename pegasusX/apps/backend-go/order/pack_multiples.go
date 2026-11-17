package order

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

var (
	// ErrPackMultipleRequired indicates one or more CASE/BLOCK lines violate integer pack multiples.
	ErrPackMultipleRequired = errors.New("PACK_MULTIPLE_REQUIRED")
	// ErrPackMetadataUnavailable indicates Spanner/catalog pack metadata could not be read (fail-closed).
	ErrPackMetadataUnavailable = errors.New("pack_metadata_unavailable")
)

// SnapMode determines how arbitrary quantities snap to case pack multiples.
type SnapMode string

const (
	SnapModeNearest SnapMode = "NEAREST"
	SnapModeCeil    SnapMode = "CEIL"
	SnapModeFloor   SnapMode = "FLOOR"
)

// PackRule captures pack constraints for a catalog SKU.
type PackRule struct {
	SaleUnit     string
	UnitsPerPack int64
}

type productPackRule = PackRule

// PackRuleReader loads pack rules for the requested SKUs (defaults to Spanner Products).
type PackRuleReader func(ctx context.Context, skus []string) (map[string]PackRule, error)

// PackLineSuggestion provides structured quantity snap suggestions for a failing line.
type PackLineSuggestion struct {
	SKU              string `json:"sku"`
	RequestedQty     int64  `json:"requested_qty"`
	PackSize         int64  `json:"pack_size"`
	SuggestedFloor   int64  `json:"suggested_floor"`
	SuggestedCeil    int64  `json:"suggested_ceil"`
	SuggestedNearest int64  `json:"suggested_nearest"`
	SaleUnit         string `json:"sale_unit"`
	Message          string `json:"message,omitempty"`
}

// PackValidationError carries machine-readable PACK_MULTIPLE_REQUIRED details and suggestions.
type PackValidationError struct {
	Code        string               `json:"code"`
	Message     string               `json:"message"`
	LineErrors  map[string]string    `json:"line_errors,omitempty"`
	Suggestions []PackLineSuggestion `json:"suggestions"`
}

func (e *PackValidationError) Error() string {
	if e == nil {
		return ErrPackMultipleRequired.Error()
	}
	if len(e.LineErrors) > 0 {
		return fmt.Sprintf("%s: %v", e.Code, e.LineErrors)
	}
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return ErrPackMultipleRequired.Error()
}

func (e *PackValidationError) Is(target error) bool {
	return target == ErrPackMultipleRequired || target == ErrLineQuantityOutOfRange
}

// SnapToPackMultiple snaps a raw order quantity to the nearest or directed multiple of pack size.
// If pack <= 1 or qty <= 0, returns qty (or 0 if negative).
// E.g. with pack=6, qty=7, mode="NEAREST" -> 6; qty=10 -> 12; mode="CEIL" -> 12.
func SnapToPackMultiple(qty int64, pack int64, mode SnapMode) int64 {
	if qty <= 0 {
		return 0
	}
	if pack <= 1 {
		return qty
	}

	remainder := qty % pack
	if remainder == 0 {
		return qty
	}

	lower := qty - remainder
	upper := lower + pack

	switch mode {
	case SnapModeFloor:
		return lower
	case SnapModeCeil:
		return upper
	case SnapModeNearest:
		fallthrough
	default:
		if remainder*2 >= pack {
			return upper
		}
		if lower == 0 {
			// Do not drop to 0 if retailer requested at least one item under nearest
			return upper
		}
		return lower
	}
}

func collectPackValidationSKUs(items []LineItem) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	skus := make([]string, 0, len(items))
	for _, item := range items {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" || item.Quantity <= 0 {
			continue
		}
		if _, exists := seen[sku]; !exists {
			seen[sku] = struct{}{}
			skus = append(skus, sku)
		}
	}
	return skus
}

func loadPackRulesFromSpanner(ctx context.Context, client *spanner.Client, skus []string) (map[string]PackRule, error) {
	if client == nil || len(skus) == 0 {
		return nil, nil
	}
	keySets := make([]spanner.KeySet, 0, len(skus))
	for _, sku := range skus {
		keySets = append(keySets, spanner.Key{sku})
	}
	rules := make(map[string]PackRule, len(keySets))
	iter := client.Single().Read(ctx, "Products", spanner.KeySets(keySets...), []string{"ProductId", "SaleUnit", "UnitsPerPack"})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrPackMetadataUnavailable, err)
		}
		var sku string
		var saleUnit spanner.NullString
		var unitsPerPack spanner.NullInt64
		if err := row.Columns(&sku, &saleUnit, &unitsPerPack); err != nil {
			return nil, fmt.Errorf("%w: decode product pack rule: %w", ErrPackMetadataUnavailable, err)
		}
		if !unitsPerPack.Valid {
			continue
		}
		resolvedSaleUnit := ""
		if saleUnit.Valid {
			resolvedSaleUnit = saleUnit.StringVal
		}
		rules[strings.TrimSpace(sku)] = PackRule{
			SaleUnit:     resolvedSaleUnit,
			UnitsPerPack: unitsPerPack.Int64,
		}
	}
	return rules, nil
}

func validatePackMultiples(ctx context.Context, client *spanner.Client, items []LineItem) error {
	return validatePackMultiplesWithReader(ctx, client, nil, items)
}

func validatePackMultiplesWithReader(ctx context.Context, client *spanner.Client, reader PackRuleReader, items []LineItem) error {
	skus := collectPackValidationSKUs(items)
	if len(skus) == 0 {
		return nil
	}
	var (
		rules map[string]PackRule
		err   error
	)
	switch {
	case reader != nil:
		rules, err = reader(ctx, skus)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrPackMetadataUnavailable, err)
		}
	case client != nil:
		rules, err = loadPackRulesFromSpanner(ctx, client, skus)
		if err != nil {
			return err
		}
	default:
		return nil
	}
	if packErr := evaluatePackMultiplesDetailed(items, rules); packErr != nil {
		return packErr
	}
	return nil
}

func evaluatePackMultiplesDetailed(items []LineItem, rules map[string]PackRule) *PackValidationError {
	if len(items) == 0 || len(rules) == 0 {
		return nil
	}
	lineErrs := make(map[string]string)
	suggestions := make([]PackLineSuggestion, 0)
	seenErrSKUs := make(map[string]struct{})
	for _, item := range items {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" || item.Quantity <= 0 {
			continue
		}
		rule, ok := rules[sku]
		if !ok {
			continue
		}
		saleUnit := strings.ToUpper(strings.TrimSpace(rule.SaleUnit))
		if saleUnit != "CASE" && saleUnit != "BLOCK" {
			continue
		}
		if rule.UnitsPerPack <= 1 {
			continue
		}
		if item.Quantity%rule.UnitsPerPack != 0 {
			msg := fmt.Sprintf("quantity must be a multiple of %d (%s)", rule.UnitsPerPack, saleUnit)
			lineErrs[sku] = msg
			if _, exists := seenErrSKUs[sku]; !exists {
				seenErrSKUs[sku] = struct{}{}
				suggestions = append(suggestions, PackLineSuggestion{
					SKU:              sku,
					RequestedQty:     item.Quantity,
					PackSize:         rule.UnitsPerPack,
					SuggestedFloor:   SnapToPackMultiple(item.Quantity, rule.UnitsPerPack, SnapModeFloor),
					SuggestedCeil:    SnapToPackMultiple(item.Quantity, rule.UnitsPerPack, SnapModeCeil),
					SuggestedNearest: SnapToPackMultiple(item.Quantity, rule.UnitsPerPack, SnapModeNearest),
					SaleUnit:         saleUnit,
					Message:          msg,
				})
			}
		}
	}
	if len(lineErrs) == 0 {
		return nil
	}
	return &PackValidationError{
		Code:        ErrPackMultipleRequired.Error(),
		Message:     "one or more lines require integer pack multiples",
		LineErrors:  lineErrs,
		Suggestions: suggestions,
	}
}

func evaluatePackMultipleRules(items []LineItem, rules map[string]productPackRule) map[string]string {
	detailed := evaluatePackMultiplesDetailed(items, rules)
	if detailed == nil || len(detailed.LineErrors) == 0 {
		return nil
	}
	return detailed.LineErrors
}
