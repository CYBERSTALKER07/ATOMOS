package approvals

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestCEL_ValidExpressionsAndEvaluation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		expr string
		ctx  map[string]any
		want bool
	}{
		{
			name: "amount_and_nested_retailer",
			expr: `amount_minor > 50000000 && retailer.risk_tier == "HIGH"`,
			ctx: map[string]any{
				"amount_minor": int64(75000000),
				"retailer":     map[string]any{"risk_tier": "HIGH"},
			},
			want: true,
		},
		{
			name: "amount_below_threshold",
			expr: `amount_minor > 50000000 && retailer.risk_tier == "HIGH"`,
			ctx: map[string]any{
				"amount_minor": int64(50000000),
				"retailer":     map[string]any{"risk_tier": "HIGH"},
			},
			want: false,
		},
		{
			name: "in_list_and_or_precedence",
			expr: `currency in ["UZS", "USD"] && (amount_minor >= 10000000 || !is_routine)`,
			ctx: map[string]any{
				"currency":     "UZS",
				"amount_minor": int64(500),
				"is_routine":   false,
			},
			want: true,
		},
		{
			name: "not_in_and_negation",
			expr: `!(actor.role in ["AUDITOR", "AGENT"]) && status not_in ["CANCELLED", "EXPIRED"]`,
			ctx: map[string]any{
				"actor":  map[string]any{"role": "FINANCE_MANAGER"},
				"status": "PENDING",
			},
			want: true,
		},
		{
			name: "comparisons_lte_neq_negative",
			expr: `delta_minor <= -10000 && account_code != "REV:SALES"`,
			ctx: map[string]any{
				"delta_minor":  int64(-15000),
				"account_code": "EXP:SHRINK",
			},
			want: true,
		},
		{
			name: "string_helpers_and_size",
			expr: `memo.startsWith("BREAK:") && account_code.contains("VAT") && size(tags) >= 2`,
			ctx: map[string]any{
				"memo":         "BREAK: manual correction",
				"account_code": "TAX:VAT:OUTPUT",
				"tags":         []string{"vat", "manual"},
			},
			want: true,
		},
		{
			name: "in_field_slice",
			expr: `"FINANCE_MANAGER" in actor.roles`,
			ctx: map[string]any{
				"actor": map[string]any{"roles": []string{"ACCOUNTANT", "FINANCE_MANAGER"}},
			},
			want: true,
		},
		{
			name: "json_number_int64",
			expr: `amount_minor >= 1000`,
			ctx: map[string]any{
				"amount_minor": json.Number("1000"),
			},
			want: true,
		},
		{
			name: "missing_nested_field_is_safe_false",
			expr: `amount_minor > 100 && retailer.risk_tier == "HIGH"`,
			ctx: map[string]any{
				"amount_minor": int64(500),
			},
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := EvaluateCEL(tc.expr, tc.ctx)
			if err != nil {
				t.Fatalf("EvaluateCEL(%q) error: %v", tc.expr, err)
			}
			if got != tc.want {
				t.Fatalf("EvaluateCEL(%q) = %v, want %v", tc.expr, got, tc.want)
			}
		})
	}
}

func TestCEL_MalformedAndFloatRejected(t *testing.T) {
	t.Parallel()

	badExprs := []string{
		"",
		"   ",
		"amount_minor > &&",
		`(amount_minor > 10`,
		`amount_minor = 10`,
		`amount_minor > 10.5`,
		`amount_minor > .5`,
		`amount_minor > 1e5`,
		`"unterminated`,
		`amount_minor in [1, 2`,
		`12345`,
		`"just_a_string"`,
		`unknownFunc(amount_minor)`,
		`amount_minor > 10 extra`,
		`amount_minor & 1`,
		`amount_minor | 1`,
	}

	for _, expr := range badExprs {
		if err := ValidateCEL(expr); !errors.Is(err, ErrInvalidCEL) {
			t.Fatalf("ValidateCEL(%q) err=%v, want ErrInvalidCEL", expr, err)
		}
	}

	// Runtime float in context must be rejected cleanly.
	_, err := EvaluateCEL(`amount_minor > 100`, map[string]any{
		"amount_minor": float64(150.25),
	})
	if !errors.Is(err, ErrFloatForbidden) {
		t.Fatalf("runtime float err=%v, want ErrFloatForbidden", err)
	}

	// Deeply nested or oversized CEL expressions must fail cleanly instead of exhausting stack.
	deep := strings.Repeat("(", 80) + "amount_minor > 1" + strings.Repeat(")", 80)
	if err := ValidateCEL(deep); !errors.Is(err, ErrInvalidCEL) {
		t.Fatalf("deeply nested CEL err=%v, want ErrInvalidCEL", err)
	}
	huge := "amount_minor == 1" + strings.Repeat(" && amount_minor == 1", 600)
	if err := ValidateCEL(huge); !errors.Is(err, ErrInvalidCEL) {
		t.Fatalf("oversized CEL err=%v, want ErrInvalidCEL", err)
	}
}
