#!/bin/bash
# scripts/build-update-manifest.sh
# 
# Generates the `updater.json` (Android/Tauri) and `manifest.plist` (iOS) 
# for Enterprise Over-The-Air (OTA) updates.
# This script should be run by the CI/CD pipeline after building the binaries.

set -e

VERSION_CODE=$1
APK_PATH=$2
IPA_PATH=$3
BASE_URL="https://storage.googleapis.com/pegasusx-ssmr-app-updates"

if [ -z "$VERSION_CODE" ] || [ -z "$APK_PATH" ]; then
  echo "Usage: ./build-update-manifest.sh <version_code> <path_to_apk> [path_to_ipa]"
  exit 1
fi

echo "Generating SHA-256 hash for APK..."
APK_HASH=$(shasum -a 256 "$APK_PATH" | awk '{ print $1 }')
APK_FILENAME=$(basename "$APK_PATH")

echo "Building updater.json..."
cat <<EOF > updater.json
{
  "version_code": $VERSION_CODE,
  "apk_url": "$BASE_URL/android/retailer/$APK_FILENAME",
  "sha256": "$APK_HASH",
  "manifest_url": "$BASE_URL/ios/retailer/manifest.plist",
  "release_notes": "Enterprise update applied via CI/CD."
}
EOF

# Build iOS manifest.plist if IPA is provided
if [ -n "$IPA_PATH" ]; then
  IPA_FILENAME=$(basename "$IPA_PATH")
  echo "Building manifest.plist for iOS Enterprise OTA..."
  cat <<EOF > manifest.plist
<?xml version="1.0" encoding="UTF-8"?>
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
                    <string>$BASE_URL/ios/retailer/$IPA_FILENAME</string>
                </dict>
            </array>
            <key>metadata</key>
            <dict>
                <key>bundle-identifier</key>
                <string>com.pegasusx.retailer</string>
                <key>bundle-version</key>
                <string>$VERSION_CODE</string>
                <key>kind</key>
                <string>software</string>
                <key>title</key>
                <string>Pegasus Retailer</string>
            </dict>
        </dict>
    </array>
</dict>
</plist>
EOF
fi

echo "Done! Upload updater.json, manifest.plist, and the binaries to your GCS bucket."
