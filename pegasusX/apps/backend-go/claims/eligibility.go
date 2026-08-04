package claims

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// PhotoRequiredClaimTypes matches FileRetailerClaim evidence rules (G2 advisory list).
var PhotoRequiredClaimTypes = []string{
	string(ClaimTypeDamaged),
	string(ClaimTypeConcealedDamage),
	string(ClaimTypeTamper),
	string(ClaimTypeTemperature),
}

// ClaimEligibility is the GET /v1/orders/{id}/claim-eligibility payload.
type ClaimEligibility struct {
	Eligible           bool     `json:"eligible"`
	EndsAt             *string  `json:"ends_at"`
	WindowHours        int      `json:"window_hours"`
	HoursRemaining     float64  `json:"hours_remaining"`
	PolicySource       string   `json:"policy_source"`
	PhotoRequiredTypes []string `json:"photo_required_types"`
	OrderStatus        string   `json:"order_status"`
	Reason             string   `json:"reason,omitempty"`
}

// claimWindowPolicySource reports how the service fallback window was chosen.
func claimWindowPolicySource() string {
	if strings.TrimSpace(os.Getenv("CLAIM_WINDOW_HOURS")) != "" {
		return "ENV"
	}
	return "DEFAULT"
}

// EvaluateClaimEligibility computes window eligibility from an order snapshot.
// Prefer immutable ClaimWindowEndsAt (G3); else legacy completedAt + window (G2).
func EvaluateClaimEligibility(o OrderSnapshot, now time.Time, window time.Duration) ClaimEligibility {
	if window <= 0 {
		window = DefaultPostDeliveryClaimWindow
	}
	status := strings.ToUpper(strings.TrimSpace(o.Status))
	hours := int(math.Round(window.Hours()))
	if hours <= 0 {
		hours = int(DefaultPostDeliveryClaimWindow.Hours())
	}
	policySource := claimWindowPolicySource()
	out := ClaimEligibility{
		Eligible:           false,
		EndsAt:             nil,
		WindowHours:        hours,
		HoursRemaining:     0,
		PolicySource:       policySource,
		PhotoRequiredTypes: append([]string(nil), PhotoRequiredClaimTypes...),
		OrderStatus:        status,
	}
	if status != OrderStatusCompleted {
		out.Reason = "order_not_completed"
		return out
	}

	var ends time.Time
	if o.ClaimWindowEndsAt != nil && !o.ClaimWindowEndsAt.IsZero() {
		ends = o.ClaimWindowEndsAt.UTC()
		if o.ClaimWindowHours > 0 {
			out.WindowHours = int(o.ClaimWindowHours)
		} else {
			// Derive hours from completedAt if present.
			completedAt := o.UpdatedAt
			if completedAt.IsZero() {
				completedAt = o.CreatedAt
			}
			if !completedAt.IsZero() {
				out.WindowHours = int(math.Round(ends.Sub(completedAt.UTC()).Hours()))
				if out.WindowHours < 1 {
					out.WindowHours = hours
				}
			}
		}
		if src := strings.TrimSpace(o.ClaimWindowPolicySource); src != "" {
			out.PolicySource = src
		}
	} else {
		completedAt := o.UpdatedAt
		if completedAt.IsZero() {
			completedAt = o.CreatedAt
		}
		if completedAt.IsZero() {
			out.Reason = "order_not_completed"
			return out
		}
		ends = completedAt.UTC().Add(window)
	}

	endsStr := ends.Format(time.RFC3339)
	out.EndsAt = &endsStr
	now = now.UTC()
	if now.After(ends) {
		out.Reason = "claim_window_expired"
		out.HoursRemaining = 0
		return out
	}
	out.Eligible = true
	rem := ends.Sub(now).Hours()
	if rem < 0 {
		rem = 0
	}
	out.HoursRemaining = math.Round(rem*100) / 100
	return out
}

// GetClaimEligibility loads the order, authorizes the actor, and evaluates the window.
func (s *Service) GetClaimEligibility(ctx context.Context, actor auth.Claims, orderID string) (ClaimEligibility, error) {
	if s == nil || s.orders == nil {
		return ClaimEligibility{}, fmt.Errorf("claims service unavailable")
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return ClaimEligibility{}, ErrOrderNotFound
	}
	if actor.Role != auth.RoleRetailer && actor.Role != auth.RoleAdmin {
		return ClaimEligibility{}, ErrForbidden
	}
	o, ok, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return ClaimEligibility{}, err
	}
	if !ok {
		return ClaimEligibility{}, ErrOrderNotFound
	}
	if actor.Role == auth.RoleRetailer {
		org := auth.ResolveRetailerOrgID(actor)
		if strings.TrimSpace(o.RetailerID) != org {
			return ClaimEligibility{}, ErrForbidden
		}
	}
	now := time.Now().UTC()
	if s.now != nil {
		now = s.now().UTC()
	}
	return EvaluateClaimEligibility(o, now, s.window), nil
}
