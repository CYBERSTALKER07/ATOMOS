package updateroutes

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// Deps configures OTA manifest generation for mobile/desktop clients.
type Deps struct {
	// BaseURL is the public HTTPS origin for signed artifacts (no trailing slash).
	// Sourced from UPDATES_BASE_URL in production.
	BaseURL string
	// DefaultVersion is advertised when per-app version env is unset.
	DefaultVersion string
}

// RegisterRoutes mounts OTA discovery endpoints.
func RegisterRoutes(r chi.Router, deps Deps) {
	r.Get("/v1/updates/ios/{app_id}/manifest.plist", handleIOSManifest(deps))
	r.Get("/v1/updates/desktop/{app_id}/updater.json", handleDesktopUpdater(deps))
}

func resolveVersion(appID, fallback string) string {
	appID = strings.TrimSpace(appID)
	if appID != "" {
		// UPDATES_VERSION_<APP_ID> e.g. UPDATES_VERSION_SUPPLIER_PORTAL=1.2.3
		key := "UPDATES_VERSION_" + strings.ToUpper(strings.ReplaceAll(appID, "-", "_"))
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	if v := strings.TrimSpace(os.Getenv("UPDATES_DEFAULT_VERSION")); v != "" {
		return v
	}
	if strings.TrimSpace(fallback) != "" {
		return strings.TrimSpace(fallback)
	}
	return "1.0.0"
}

func resolveSignature(appID, platform string) string {
	appID = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(appID), "-", "_"))
	platform = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(platform), "-", "_"))
	if appID != "" && platform != "" {
		if v := strings.TrimSpace(os.Getenv("UPDATES_SIG_" + appID + "_" + platform)); v != "" {
			return v
		}
	}
	if v := strings.TrimSpace(os.Getenv("UPDATES_SIG_" + platform)); v != "" {
		return v
	}
	return ""
}

func handleIOSManifest(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appID := chi.URLParam(r, "app_id")
		base := strings.TrimRight(strings.TrimSpace(deps.BaseURL), "/")
		if base == "" {
			http.Error(w, `{"error":"updates_base_url_unconfigured"}`, http.StatusServiceUnavailable)
			return
		}

		version := resolveVersion(appID, deps.DefaultVersion)
		bundleID := fmt.Sprintf("com.void.%s", appID)
		if override := strings.TrimSpace(os.Getenv("UPDATES_IOS_BUNDLE_" + strings.ToUpper(strings.ReplaceAll(appID, "-", "_")))); override != "" {
			bundleID = override
		}
		ipaURL := fmt.Sprintf("%s/downloads/ios/%s.ipa", base, appID)
		imageURL := fmt.Sprintf("%s/downloads/ios/icon.png", base)

		manifest := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
<key>items</key>
<array>
<dict>
<key>assets</key>
<array>
<dict>
<key>kind</key>
<string>software-package</string>
<key>url</key>
<string>%s</string>
</dict>
<dict>
<key>kind</key>
<string>display-image</string>
<key>url</key>
<string>%s</string>
</dict>
<dict>
<key>kind</key>
<string>full-size-image</string>
<key>url</key>
<string>%s</string>
</dict>
</array>
<key>metadata</key>
<dict>
<key>bundle-identifier</key>
<string>%s</string>
<key>bundle-version</key>
<string>%s</string>
<key>kind</key>
<string>software</string>
<key>title</key>
<string>%s</string>
</dict>
</dict>
</array>
</dict>
</plist>`, ipaURL, imageURL, imageURL, bundleID, version, appID)

		w.Header().Set("Content-Type", "application/x-apple-aspen-metadata")
		w.Header().Set("Cache-Control", "public, max-age=60")
		_, _ = w.Write([]byte(manifest))
	}
}

func handleDesktopUpdater(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appID := chi.URLParam(r, "app_id")
		base := strings.TrimRight(strings.TrimSpace(deps.BaseURL), "/")
		if base == "" {
			http.Error(w, `{"error":"updates_base_url_unconfigured"}`, http.StatusServiceUnavailable)
			return
		}

		version := resolveVersion(appID, deps.DefaultVersion)
		macURL := fmt.Sprintf("%s/downloads/desktop/%s.app.tar.gz", base, appID)
		winURL := fmt.Sprintf("%s/downloads/desktop/%s.msi.zip", base, appID)
		macSig := resolveSignature(appID, "darwin")
		winSig := resolveSignature(appID, "windows")
		if macSig == "" {
			macSig = "UNSIGNED"
		}
		if winSig == "" {
			winSig = "UNSIGNED"
		}

		updaterJSON := fmt.Sprintf(`{
  "version": %q,
  "notes": "Latest performance and stability improvements.",
  "pub_date": %q,
  "platforms": {
    "darwin-x86_64": {
      "signature": %q,
      "url": %q
    },
    "darwin-aarch64": {
      "signature": %q,
      "url": %q
    },
    "windows-x86_64": {
      "signature": %q,
      "url": %q
    }
  }
}`, version, time.Now().UTC().Format(time.RFC3339), macSig, macURL, macSig, macURL, winSig, winURL)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=60")
		_, _ = w.Write([]byte(updaterJSON))
	}
}
