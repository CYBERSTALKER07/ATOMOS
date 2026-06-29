package platform

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// StatusExplain is human-readable guidance attached to API error bodies.
type StatusExplain struct {
	Code        string   `json:"code"`
	Title       string   `json:"title"`
	Summary     string   `json:"summary"`
	NextSteps   []string `json:"next_steps,omitempty"`
	DeepLink    string   `json:"deep_link,omitempty"`
	Recoverable bool     `json:"recoverable"`
}

var explainCatalog = map[string]StatusExplain{
	"zone_miss": {
		Code: "zone_miss", Title: "Outside delivery zone",
		Summary:     "This retailer location is outside the supplier's configured delivery zone.",
		NextSteps:   []string{"Verify the delivery address", "Ask the supplier to extend zone coverage"},
		Recoverable: false,
	},
	"geofence_violation": {
		Code: "geofence_violation", Title: "Geofence check failed",
		Summary:     "The driver is too far from the delivery location to complete this action.",
		NextSteps:   []string{"Move closer to the retailer location", "Retry when GPS accuracy improves"},
		Recoverable: true,
	},
	"dispatch_partial_commit": {
		Code: "dispatch_partial_commit", Title: "Dispatch partially committed",
		Summary:     "Some routes were committed before the dispatch batch failed. Review manifests before retrying.",
		NextSteps:   []string{"Open dispatch replay to inspect the last run", "Retry only unassigned orders"},
		DeepLink:    "/dispatch",
		Recoverable: true,
	},
	"dispatch_unavailable": {
		Code: "dispatch_unavailable", Title: "Dispatch unavailable",
		Summary:     "Smart dispatch requires a live database connection.",
		NextSteps:   []string{"Retry when Spanner is reachable", "Use manual assignment if urgent"},
		Recoverable: true,
	},
	"warehouse_scope_required": {
		Code: "warehouse_scope_required", Title: "Warehouse scope required",
		Summary:     "This action requires an authenticated warehouse operator with a home node.",
		NextSteps:   []string{"Sign in with a warehouse account", "Ensure your profile has a warehouse assigned"},
		Recoverable: true,
	},
	"target_manifest_capacity_exceeded": {
		Code: "target_manifest_capacity_exceeded", Title: "Manifest capacity exceeded",
		Summary:     "Adding this order would exceed the truck volume limit.",
		NextSteps:   []string{"Remove lower-priority stops", "Split orders across another manifest"},
		Recoverable: true,
	},
	"manifest_capacity_exceeded": {
		Code: "manifest_capacity_exceeded", Title: "Manifest capacity exceeded",
		Summary:     "The manifest cannot accept more volume at the loading gate.",
		NextSteps:   []string{"Seal and dispatch the current load", "Create a new manifest for overflow orders"},
		Recoverable: true,
	},
	"preorder_edit_locked": {
		Code: "preorder_edit_locked", Title: "Pre-order edit locked",
		Summary:     "This pre-order is within the edit lock window before its scheduled delivery date.",
		NextSteps:   []string{"Contact the warehouse for date changes", "Cancel and re-place if retailer agrees"},
		Recoverable: false,
	},
	"shop_closed": {
		Code: "shop_closed", Title: "Shop closed on arrival",
		Summary:     "The driver reported the retailer shop was closed during delivery.",
		NextSteps:   []string{"Respond in the shop-closed inbox", "Approve bypass or reschedule delivery"},
		DeepLink:    "/exceptions/shop-closed",
		Recoverable: true,
	},
	"AWAITING_PAYLOAD_SEAL": {
		Code: "AWAITING_PAYLOAD_SEAL", Title: "Awaiting payload seal",
		Summary:     "The manifest is still being loaded. The payloader must finish tap-check and seal before the driver can depart.",
		NextSteps:   []string{"Wait for the payloader to seal the manifest", "Contact the loading bay if loading is complete"},
		Recoverable: true,
	},
	"manifest_seal_failed": {
		Code: "manifest_seal_failed", Title: "Manifest seal failed",
		Summary:     "The server could not seal this manifest. Review manifest state and retry.",
		NextSteps:   []string{"Confirm every order is sealed", "Retry seal after refreshing the manifest"},
		Recoverable: true,
	},
	"manifest_not_sealable": {
		Code: "manifest_not_sealable", Title: "Manifest not sealable",
		Summary:     "This manifest is not in a state that allows sealing.",
		NextSteps:   []string{"Finish loading and per-order seals", "Refresh the manifest before retrying"},
		Recoverable: true,
	},
}

// ExplainForError maps an error or error code string to guidance copy.
func ExplainForError(err error) *StatusExplain {
	if err == nil {
		return nil
	}
	msg := strings.TrimSpace(err.Error())
	if ex := ExplainForCode(msg); ex != nil {
		return ex
	}
	if idx := strings.Index(msg, ":"); idx > 0 {
		if ex := ExplainForCode(strings.TrimSpace(msg[idx+1:])); ex != nil {
			return ex
		}
		return ExplainForCode(strings.TrimSpace(msg[:idx]))
	}
	return nil
}

// ExplainForCode returns catalog guidance for a canonical error code.
func ExplainForCode(code string) *StatusExplain {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil
	}
	if ex, ok := explainCatalog[code]; ok {
		copy := ex
		return &copy
	}
	lower := strings.ToLower(code)
	if strings.Contains(lower, "preorder edit locked") {
		return ExplainForCode("preorder_edit_locked")
	}
	return nil
}

// AttachExplain adds an `explain` field to a JSON error body map.
func AttachExplain(body map[string]any, err error) {
	if body == nil {
		return
	}
	code, _ := body["error"].(string)
	ex := ExplainForCode(code)
	if ex == nil {
		ex = ExplainForError(err)
	}
	if ex != nil {
		body["explain"] = ex
	}
}

// WriteErrorWithExplain writes a JSON error response with optional explain guidance.
func WriteErrorWithExplain(w http.ResponseWriter, status int, code string, err error) {
	body := map[string]any{"error": strings.TrimSpace(code)}
	if err != nil && err.Error() != code {
		body["detail"] = err.Error()
	}
	AttachExplain(body, err)
	if ex := ExplainForCode(code); ex == nil && err != nil {
		AttachExplain(body, err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// WriteErrorWithExplainFromErr derives the code from err when present.
func WriteErrorWithExplainFromErr(w http.ResponseWriter, status int, err error) {
	if err == nil {
		WriteErrorWithExplain(w, status, "internal_error", errors.New("internal_error"))
		return
	}
	WriteErrorWithExplain(w, status, err.Error(), err)
}
