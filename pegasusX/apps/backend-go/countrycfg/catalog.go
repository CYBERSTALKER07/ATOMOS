package countrycfg

import "strings"

// Catalog is the signup-picker country set. Checkout gateways still do not read this.
func Catalog() map[string]Config {
	uz := UZDefault()
	uz.OpsReadsThis = true
	return map[string]Config{
		"UZ": uz,
		"KZ": {
			CountryCode: "KZ", CountryName: "Kazakhstan", Timezone: "Asia/Almaty",
			CurrencyCode: "KZT", CurrencyDecimalPlaces: 2, DistanceUnit: "km", GridSystem: "H3",
			BreachRadiusMeters: 150, ShopClosedGraceMinutes: 10, ShopClosedEscalationMinutes: 30,
			OfflineModeDurationMinutes: 120, CashCustodyAlertHours: 24,
			PaymentGatewaysListed: []string{"GLOBAL_PAY", "CASH"}, CheckoutReadsThis: false, OpsReadsThis: true, Source: "catalog",
		},
		"KG": catalogRow("KG", "Kyrgyzstan", "Asia/Bishkek", "KGS"),
		"TJ": catalogRow("TJ", "Tajikistan", "Asia/Dushanbe", "TJS"),
		"TM": catalogRow("TM", "Turkmenistan", "Asia/Ashgabat", "TMT"),
		"AE": catalogRow("AE", "United Arab Emirates", "Asia/Dubai", "AED"),
		"TR": catalogRow("TR", "Türkiye", "Europe/Istanbul", "TRY"),
		"RU": catalogRow("RU", "Russia", "Europe/Moscow", "RUB"),
		"US": {
			CountryCode: "US", CountryName: "United States", Timezone: "America/New_York",
			CurrencyCode: "USD", CurrencyDecimalPlaces: 2, DistanceUnit: "mi", GridSystem: "H3",
			BreachRadiusMeters: 150, ShopClosedGraceMinutes: 10, ShopClosedEscalationMinutes: 30,
			OfflineModeDurationMinutes: 120, CashCustodyAlertHours: 24,
			PaymentGatewaysListed: []string{"GLOBAL_PAY", "CASH"}, CheckoutReadsThis: false, OpsReadsThis: true, Source: "catalog",
		},
		"GB": catalogRow("GB", "United Kingdom", "Europe/London", "GBP"),
	}
}

func catalogRow(code, name, tz, currency string) Config {
	return Config{
		CountryCode: code, CountryName: name, Timezone: tz,
		CurrencyCode: currency, CurrencyDecimalPlaces: 2, DistanceUnit: "km", GridSystem: "H3",
		BreachRadiusMeters: 150, ShopClosedGraceMinutes: 10, ShopClosedEscalationMinutes: 30,
		OfflineModeDurationMinutes: 120, CashCustodyAlertHours: 24,
		PaymentGatewaysListed: []string{"GLOBAL_PAY", "CASH"}, CheckoutReadsThis: false, OpsReadsThis: true, Source: "catalog",
	}
}

func Lookup(code string) (Config, bool) {
	code = strings.ToUpper(strings.TrimSpace(code))
	cfg, ok := Catalog()[code]
	return cfg, ok
}
