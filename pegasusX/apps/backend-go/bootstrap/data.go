package bootstrap

import "github.com/pegasusx/pegasusx/apps/backend-go/supplier"

const (
	defaultDeliveryZoneCenterLat = 41.2995
	defaultDeliveryZoneCenterLng = 69.2401
	defaultDeliveryZoneRadiusKm  = 20.0
)

type deliveryZoneSeed struct {
	CenterLat float64
	CenterLng float64
	RadiusKm  float64
}

func resolveDeliveryZoneSeed(cfg *Config) deliveryZoneSeed {
	seed := deliveryZoneSeed{
		CenterLat: defaultDeliveryZoneCenterLat,
		CenterLng: defaultDeliveryZoneCenterLng,
		RadiusKm:  defaultDeliveryZoneRadiusKm,
	}
	if cfg == nil {
		return seed
	}
	if cfg.DeliveryZoneCenterLat != 0 || cfg.DeliveryZoneCenterLng != 0 {
		seed.CenterLat = cfg.DeliveryZoneCenterLat
		seed.CenterLng = cfg.DeliveryZoneCenterLng
	}
	if cfg.DeliveryZoneRadiusKm > 0 {
		seed.RadiusKm = cfg.DeliveryZoneRadiusKm
	}
	return seed
}

func (d deliveryZoneSeed) withSupplierProfile(profile supplier.Profile) deliveryZoneSeed {
	if profile.WarehouseLat != 0 || profile.WarehouseLng != 0 {
		d.CenterLat = profile.WarehouseLat
		d.CenterLng = profile.WarehouseLng
	}
	return d
}
