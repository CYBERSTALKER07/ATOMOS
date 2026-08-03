package pricing

import (
	"time"
)

type PriceList struct {
	PriceListId   string     `json:"price_list_id" spanner:"PriceListId"`
	SupplierId    string     `json:"supplier_id" spanner:"SupplierId"`
	Name          string     `json:"name" spanner:"Name"`
	EffectiveFrom time.Time  `json:"effective_from" spanner:"EffectiveFrom"`
	EffectiveTo   *time.Time `json:"effective_to" spanner:"EffectiveTo"`
}

type PriceListItem struct {
	PriceListId    string `json:"price_list_id" spanner:"PriceListId"`
	Sku            string `json:"sku" spanner:"Sku"`
	UnitPriceMinor int64  `json:"unit_price_minor" spanner:"UnitPriceMinor"`
	MinQty         *int64 `json:"min_qty" spanner:"MinQty"`
}
