package updateroutes

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type Deps struct {
	BaseURL string // e.g. "https://void.example.com"
}

func RegisterRoutes(r chi.Router, deps Deps) {
	r.Get("/v1/updates/ios/{app_id}/manifest.plist", handleIOSManifest(deps))
	r.Get("/v1/updates/desktop/{app_id}/updater.json", handleDesktopUpdater(deps))
}

func handleIOSManifest(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appID := chi.URLParam(r, "app_id")
		
		// In a real system, version would be pulled from a DB or env
		version := "1.0.0"
		bundleID := fmt.Sprintf("com.void.%s", appID)
		ipaURL := fmt.Sprintf("%s/downloads/ios/%s.ipa", deps.BaseURL, appID)
		imageURL := fmt.Sprintf("%s/downloads/ios/icon.png", deps.BaseURL)
		
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
		w.Write([]byte(manifest))
	}
}

func handleDesktopUpdater(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appID := chi.URLParam(r, "app_id")
		
		version := "1.0.0"
		macURL := fmt.Sprintf("%s/downloads/desktop/%s.app.tar.gz", deps.BaseURL, appID)
		winURL := fmt.Sprintf("%s/downloads/desktop/%s.msi.zip", deps.BaseURL, appID)
		
		// In Tauri, signatures are required for secure updates. 
		// We'll return dummy signatures if they are not generated yet.
		updaterJSON := fmt.Sprintf(`{
  "version": "%s",
  "notes": "Latest performance and stability improvements.",
  "pub_date": "%s",
  "platforms": {
    "darwin-x86_64": {
      "signature": "PLACEHOLDER_MAC_SIG",
      "url": "%s"
    },
    "darwin-aarch64": {
      "signature": "PLACEHOLDER_MAC_SIG",
      "url": "%s"
    },
    "windows-x86_64": {
      "signature": "PLACEHOLDER_WIN_SIG",
      "url": "%s"
    }
  }
}`, version, time.Now().Format(time.RFC3339), macURL, macURL, winURL)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(updaterJSON))
	}
}
