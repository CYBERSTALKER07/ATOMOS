package warehouse

import (
	"context"
	"errors"
	"net/http"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

func requestPack(ctx context.Context) (auth.MarketPack, error) {
	return auth.CheckoutPackFromContext(ctx)
}

func stampWarehouseCoords(ctx context.Context, lat, lng float64, requestedCountry string) (proximity.NodeGeography, error) {
	pack, err := requestPack(ctx)
	if err != nil {
		return proximity.NodeGeography{}, err
	}
	return proximity.StampNodeGeography(pack, lat, lng, requestedCountry)
}

func stampWarehouseEntity(ctx context.Context, w *Warehouse) error {
	if w == nil {
		return auth.ErrGeographyIncomplete
	}
	if w.Lat != nil && w.Lng != nil {
		geo, err := stampWarehouseCoords(ctx, *w.Lat, *w.Lng, w.CountryCode)
		if err != nil {
			return err
		}
		w.CountryCode = geo.CountryCode
		cell := geo.H3Cell
		w.H3Cell = &cell
		return nil
	}
	pack, err := requestPack(ctx)
	if err != nil {
		return err
	}
	country, err := proximity.ResolveNodeCountry(pack, w.CountryCode)
	if err != nil {
		return err
	}
	w.CountryCode = country
	return nil
}

func writeMarketLaw(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, auth.ErrCrossMarketDeferred) || errors.Is(err, auth.ErrGeographyIncomplete) ||
		errors.Is(err, auth.ErrMarketPackUnknown) || errors.Is(err, auth.ErrMarketPackNotShipped) {
		status, code := auth.CheckoutPackHTTPStatus(err)
		writeJSON(w, status, map[string]string{"error": code})
		return true
	}
	return false
}

func packCurrencyDefault() string {
	pack, ok := auth.ResolveShippedMarketPack(auth.DefaultMarketCodeFromEnv())
	if !ok {
		return ""
	}
	cur, err := auth.PackCurrency(pack)
	if err != nil {
		return ""
	}
	return cur
}

func (s *Service) responseCurrency(ctx context.Context) string {
	cur, err := auth.CoalesceCurrency(ctx, s.analyticsSupplierID(ctx), s.currency)
	if err != nil {
		return ""
	}
	return cur
}
