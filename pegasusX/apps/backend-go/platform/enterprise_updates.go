package platform

import (
	"fmt"
	"os"
	"strings"
)

// EnterpriseCDNBase is the public HTTPS origin for website-only enterprise
// install packages and updater manifests. Override with UPDATES_BASE_URL
// (same env as desktop OTA) — no trailing slash.
const defaultEnterpriseCDNBase = "https://storage.googleapis.com/pegasusx-ssmr-app-updates"

// EnterpriseChannel is the client-policy channel for website/sideload builds.
// Desktop website Tauri shells and mobile website APK/IPA use this channel
// (CDN OTA). Microsoft Store / Mac App Store / Play / App Store use production.
const EnterpriseChannel = "enterprise"

// IsEnterpriseChannel reports whether channel is website-only enterprise OTA.
func IsEnterpriseChannel(channel string) bool {
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case EnterpriseChannel, "production-web", "website", "web-enterprise", "sideload":
		return true
	default:
		return false
	}
}

// StoreChannel is the client-policy channel for public store builds
// (Play, App Store, Microsoft Store, Mac App Store).
// Store clients open the store listing; they must not use website CDN OTA.
const StoreChannel = "production"

// IsStoreChannel reports whether channel is public store distribution.
func IsStoreChannel(channel string) bool {
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case StoreChannel, "store", "appstore", "app-store", "play", "playstore",
		"google-play", "ms-store", "microsoft-store", "mac-app-store", "macstore":
		return true
	default:
		return false
	}
}

// NormalizeChannel maps aliases; empty → production (store-oriented default).
// Website / sideload clients MUST pass channel=enterprise.
// App Store / Play clients MUST pass channel=production (or store aliases).
func NormalizeChannel(channel string) string {
	c := strings.ToLower(strings.TrimSpace(channel))
	if c == "" {
		return StoreChannel
	}
	if IsEnterpriseChannel(c) {
		return EnterpriseChannel
	}
	if IsStoreChannel(c) {
		return StoreChannel
	}
	return c
}

// EnterpriseCDNBase returns the configured CDN origin for manifests/binaries.
func EnterpriseCDNBase() string {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("UPDATES_BASE_URL")), "/")
	if base == "" {
		base = defaultEnterpriseCDNBase
	}
	return base
}

// DefaultEnterpriseManifestURL returns the canonical updater.json path for a
// role/platform on the enterprise CDN. Empty when not applicable.
//
// Layout:
//
//	{base}/android/supplier/updater.json
//	{base}/ios/supplier/updater.json
//	{base}/supplier-desktop/windows/x86_64/updater.json   (Tauri 2 {{target}}/{{arch}})
//	{base}/retailer-desktop/darwin/aarch64/updater.json
//	…
func DefaultEnterpriseManifestURL(role, platform string) string {
	platform = normalizePlatform(platform)
	role = normalizeRole(role)
	appSlug := enterpriseAppSlug(role)
	if appSlug == "" {
		return ""
	}
	base := EnterpriseCDNBase()
	switch platform {
	case "ios", "android":
		return fmt.Sprintf("%s/%s/%s/updater.json", base, platform, appSlug)
	case "desktop":
		// Tauri shells poll OS/arch-specific manifests. Policy default points at
		// the primary enterprise Windows fleet path; macOS/linux clients still
		// use plugins.updater endpoints baked into tauri.conf.json.
		return fmt.Sprintf("%s/%s-desktop/windows/x86_64/updater.json", base, appSlug)
	default:
		return ""
	}
}

func enterpriseAppSlug(role string) string {
	switch normalizeRole(role) {
	case "ADMIN", "SUPPLIER":
		return "supplier"
	case "DRIVER":
		return "driver"
	case "RETAILER":
		return "retailer"
	case "WAREHOUSE":
		return "warehouse"
	case "FACTORY":
		return "factory"
	case "PAYLOAD":
		return "payload"
	default:
		return ""
	}
}

// androidPackageID is the Play applicationId for each role.
func androidPackageID(role string) string {
	switch normalizeRole(role) {
	case "ADMIN", "SUPPLIER":
		return "com.pegasusx.supplier"
	case "DRIVER":
		return "com.pegasusx.driver"
	case "RETAILER":
		return "com.pegasusx.retailer"
	case "WAREHOUSE":
		return "com.pegasusx.warehouse"
	case "FACTORY":
		return "com.pegasusx.factory"
	case "PAYLOAD":
		return "com.pegasus.payload"
	default:
		return ""
	}
}

// DefaultStoreUpdateURL returns the public store listing URL for production
// channel clients when ClientVersionPolicies.update_url is empty.
//
// Overrides (optional):
//
//	STORE_URL_ANDROID_<slug>       e.g. STORE_URL_ANDROID_SUPPLIER
//	STORE_URL_IOS_<slug>           e.g. STORE_URL_IOS_DRIVER
//	APP_STORE_ID_<slug>            e.g. APP_STORE_ID_SUPPLIER=1234567890
//	STORE_URL_DESKTOP_<slug>       generic desktop store landing
//	MS_STORE_PRODUCT_ID_<slug>     Microsoft Store ProductId (desktop)
//	MS_STORE_URL_<slug>            full ms-windows-store:// or https URL
//	MAC_APP_STORE_ID_DESKTOP_<slug> Mac App Store id for desktop shells
//	MAC_APP_STORE_URL_DESKTOP_<slug>
//
// Android falls back to Play details?id=<applicationId>.
// iOS requires APP_STORE_ID_* or STORE_URL_IOS_* (no fake App Store URL).
// Desktop prefers STORE_URL_DESKTOP_*, else MS Store product deep link,
// else Mac App Store desktop id (empty if unset).
func DefaultStoreUpdateURL(role, platform string) string {
	platform = normalizePlatform(platform)
	role = normalizeRole(role)
	slug := enterpriseAppSlug(role)
	if slug == "" {
		return ""
	}
	envSlug := strings.ToUpper(slug)
	switch platform {
	case "android":
		if u := strings.TrimSpace(os.Getenv("STORE_URL_ANDROID_" + envSlug)); u != "" {
			return u
		}
		pkg := androidPackageID(role)
		if pkg == "" {
			return ""
		}
		return "https://play.google.com/store/apps/details?id=" + pkg
	case "ios":
		if u := strings.TrimSpace(os.Getenv("STORE_URL_IOS_" + envSlug)); u != "" {
			return u
		}
		if id := strings.TrimSpace(os.Getenv("APP_STORE_ID_" + envSlug)); id != "" {
			id = strings.TrimPrefix(id, "id")
			return "https://apps.apple.com/app/id" + id
		}
		return ""
	case "desktop":
		if u := strings.TrimSpace(os.Getenv("STORE_URL_DESKTOP_" + envSlug)); u != "" {
			return u
		}
		if u := strings.TrimSpace(os.Getenv("MS_STORE_URL_" + envSlug)); u != "" {
			return u
		}
		if id := strings.TrimSpace(os.Getenv("MS_STORE_PRODUCT_ID_" + envSlug)); id != "" {
			return "ms-windows-store://pdp/?ProductId=" + id
		}
		if u := strings.TrimSpace(os.Getenv("MAC_APP_STORE_URL_DESKTOP_" + envSlug)); u != "" {
			return u
		}
		if id := strings.TrimSpace(os.Getenv("MAC_APP_STORE_ID_DESKTOP_" + envSlug)); id != "" {
			id = strings.TrimPrefix(id, "id")
			return "https://apps.apple.com/app/id" + id
		}
		return ""
	default:
		return ""
	}
}
