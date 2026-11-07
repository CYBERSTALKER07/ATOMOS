package countrycfg

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/grpc/codes"
)

const DefaultCountry = "UZ"

// Config is the signup-picker / UZ seed surface. MarketPack is product law.
// Checkout gateway selection does not read this.
type Config struct {
	CountryCode                 string   `json:"country_code"`
	CountryName                 string   `json:"country_name"`
	Timezone                    string   `json:"timezone"`
	CurrencyCode                string   `json:"currency_code"`
	CurrencyDecimalPlaces       int64    `json:"currency_decimal_places"`
	DistanceUnit                string   `json:"distance_unit"`
	GridSystem                  string   `json:"grid_system"`
	BreachRadiusMeters          float64  `json:"breach_radius_meters"`
	ShopClosedGraceMinutes      int64    `json:"shop_closed_grace_minutes"`
	ShopClosedEscalationMinutes int64    `json:"shop_closed_escalation_minutes"`
	OfflineModeDurationMinutes  int64    `json:"offline_mode_duration_minutes"`
	CashCustodyAlertHours       int64    `json:"cash_custody_alert_hours"`
	PaymentGatewaysListed       []string `json:"payment_gateways_listed,omitempty"`
	CheckoutReadsThis           bool     `json:"checkout_reads_this"`
	OpsReadsThis                bool     `json:"ops_reads_this"`
	Source                      string   `json:"source"`
}

type Override struct {
	SupplierID                  string   `json:"supplier_id"`
	CountryCode                 string   `json:"country_code"`
	BreachRadiusMeters          *float64 `json:"breach_radius_meters,omitempty"`
	ShopClosedGraceMinutes      *int64   `json:"shop_closed_grace_minutes,omitempty"`
	ShopClosedEscalationMinutes *int64   `json:"shop_closed_escalation_minutes,omitempty"`
	OfflineModeDurationMinutes  *int64   `json:"offline_mode_duration_minutes,omitempty"`
	CashCustodyAlertHours       *int64   `json:"cash_custody_alert_hours,omitempty"`
	Reason                      string   `json:"reason,omitempty"`
	UpdatedBy                   string   `json:"updated_by,omitempty"`
	Source                      string   `json:"source"`
}

func UZDefault() Config {
	return Config{
		CountryCode:                 DefaultCountry,
		CountryName:                 "Uzbekistan",
		Timezone:                    "Asia/Tashkent",
		CurrencyCode:                "UZS",
		CurrencyDecimalPlaces:       2,
		DistanceUnit:                "km",
		GridSystem:                  "H3",
		BreachRadiusMeters:          150,
		ShopClosedGraceMinutes:      10,
		ShopClosedEscalationMinutes: 30,
		OfflineModeDurationMinutes:  120,
		CashCustodyAlertHours:       24,
		PaymentGatewaysListed:       nil,
		CheckoutReadsThis:           false,
		OpsReadsThis:                true,
		Source:                      "uz_seed",
	}
}

type Handlers struct {
	Spanner *spanner.Client
}

func scopeSupplier(r *http.Request) string {
	sid := strings.TrimSpace(auth.PreferTenantSupplierID(r.Context(), ""))
	if sid != "" {
		return sid
	}
	if id, ok := auth.ResolveSupplierID(r.Context()); ok {
		return strings.TrimSpace(id)
	}
	return ""
}

func (h *Handlers) HandleCountryConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		web.JSONError(w, "method_not_allowed", http.StatusMethodNotAllowed)
		return
	}
	code := strings.ToUpper(strings.TrimSpace(chi.URLParam(r, "code")))
	if code == "" {
		code = DefaultCountry
	}
	cfg, ok := Lookup(code)
	if !ok {
		web.JSONError(w, "country_not_supported", http.StatusNotFound)
		return
	}
	web.JSONResponse(w, http.StatusOK, cfg)
}

func (h *Handlers) HandleCountryOverride(w http.ResponseWriter, r *http.Request) {
	sid := scopeSupplier(r)
	if sid == "" {
		web.JSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	code := strings.ToUpper(strings.TrimSpace(chi.URLParam(r, "code")))
	if code == "" {
		code = DefaultCountry
	}
	if _, ok := Lookup(code); !ok {
		web.JSONError(w, "country_not_supported", http.StatusNotFound)
		return
	}
	switch r.Method {
	case http.MethodGet:
		ov, err := h.loadOverride(r.Context(), sid, code)
		if err != nil {
			web.JSONError(w, "query_failed", http.StatusInternalServerError)
			return
		}
		if ov == nil {
			web.JSONResponse(w, http.StatusOK, Override{SupplierID: sid, CountryCode: code, Source: "country_default"})
			return
		}
		web.JSONResponse(w, http.StatusOK, ov)
	case http.MethodPatch:
		claims, ok := auth.FromContext(r.Context())
		if !ok {
			web.JSONError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var req Override
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			web.JSONError(w, "invalid_json", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Reason) == "" {
			web.JSONError(w, "reason_required", http.StatusBadRequest)
			return
		}
		if h == nil || h.Spanner == nil {
			web.JSONError(w, "countrycfg_unavailable", http.StatusServiceUnavailable)
			return
		}
		req.SupplierID = sid
		req.CountryCode = code
		req.UpdatedBy = claims.Subject
		req.Source = "supplier_override"
		row := map[string]any{
			"SupplierId":    sid,
			"CountryCode":   code,
			"Reason":        req.Reason,
			"UpdatedBy":     claims.Subject,
			"UpdatedByType": "SUPPLIER",
			"CreatedAt":     spanner.CommitTimestamp,
			"UpdatedAt":     spanner.CommitTimestamp,
		}
		if req.BreachRadiusMeters != nil {
			row["BreachRadiusMeters"] = *req.BreachRadiusMeters
		}
		if req.ShopClosedGraceMinutes != nil {
			row["ShopClosedGraceMinutes"] = *req.ShopClosedGraceMinutes
		}
		if req.ShopClosedEscalationMinutes != nil {
			row["ShopClosedEscalationMinutes"] = *req.ShopClosedEscalationMinutes
		}
		if req.OfflineModeDurationMinutes != nil {
			row["OfflineModeDurationMinutes"] = *req.OfflineModeDurationMinutes
		}
		if req.CashCustodyAlertHours != nil {
			row["CashCustodyAlertHours"] = *req.CashCustodyAlertHours
		}
		err := spannerutils.RunReadWriteTransaction(r.Context(), h.Spanner, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			buf := outbox.NewSpannerTxnBuffer(txn)
			if err := outbox.EmitJSON(ctx, buf, events.AggregateCountryOverride, sid+":"+code, events.TopicMain, map[string]any{
				"type":         "COUNTRY_OVERRIDE_UPDATED",
				"supplier_id":  sid,
				"country_code": code,
				"reason":       req.Reason,
			}); err != nil {
				return err
			}
			if err := buf.Flush(ctx); err != nil {
				return err
			}
			return txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("SupplierCountryOverrides", row)})
		})
		if err != nil {
			web.JSONError(w, "update_failed", http.StatusInternalServerError)
			return
		}
		web.JSONResponse(w, http.StatusOK, req)
	default:
		web.JSONError(w, "method_not_allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handlers) loadOverride(ctx context.Context, supplierID, code string) (*Override, error) {
	if h == nil || h.Spanner == nil {
		return nil, errors.New("countrycfg_unavailable")
	}
	row, err := h.Spanner.Single().ReadRow(ctx, "SupplierCountryOverrides", spanner.Key{supplierID, code}, []string{
		"BreachRadiusMeters", "ShopClosedGraceMinutes", "ShopClosedEscalationMinutes",
		"OfflineModeDurationMinutes", "CashCustodyAlertHours", "Reason", "UpdatedBy",
	})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	ov := &Override{SupplierID: supplierID, CountryCode: code, Source: "supplier_override"}
	var radius spanner.NullFloat64
	var grace, esc, offline, cash spanner.NullInt64
	var reason, updated spanner.NullString
	if err := row.Columns(&radius, &grace, &esc, &offline, &cash, &reason, &updated); err != nil {
		return nil, err
	}
	if radius.Valid {
		v := radius.Float64
		ov.BreachRadiusMeters = &v
	}
	if grace.Valid {
		v := grace.Int64
		ov.ShopClosedGraceMinutes = &v
	}
	if esc.Valid {
		v := esc.Int64
		ov.ShopClosedEscalationMinutes = &v
	}
	if offline.Valid {
		v := offline.Int64
		ov.OfflineModeDurationMinutes = &v
	}
	if cash.Valid {
		v := cash.Int64
		ov.CashCustodyAlertHours = &v
	}
	if reason.Valid {
		ov.Reason = reason.StringVal
	}
	if updated.Valid {
		ov.UpdatedBy = updated.StringVal
	}
	return ov, nil
}
