package com.pegasusx.driver.service

import com.pegasusx.driver.BuildConfig

/**
 * Distribution config for native Android.
 *
 * - **enterprise** flavor: website CDN OTA (`channel=enterprise`, ENABLE_CDN_OTA=true)
 * - **store** flavor: Play Store (`channel=production`, ENABLE_CDN_OTA=false)
 *
 * Build with: `./gradlew :app:assembleStoreRelease` or `assembleEnterpriseRelease`.
 */
object EnterpriseUpdateConfig {
    /** Client-policy channel — production for Play, enterprise for website. */
    val CHANNEL: String
        get() = BuildConfig.DISTRIBUTION_CHANNEL

    /** When false, never download/install APK from CDN; open Play listing instead. */
    val enableCdnOta: Boolean
        get() = BuildConfig.ENABLE_CDN_OTA

    /** Policy role for client-policy API. */
    const val POLICY_ROLE = "DRIVER"

    /**
     * Fallback CDN manifest for enterprise only.
     * Store builds ignore this and use [storeListingUrl].
     */
    const val DEFAULT_MANIFEST_URL =
        "https://storage.googleapis.com/pegasusx-ssmr-app-updates/android/driver/updater.json"

    /** Play Store listing (store flavor); empty on enterprise flavor. */
    val storeListingUrl: String
        get() = BuildConfig.STORE_LISTING_URL.ifBlank {
            "https://play.google.com/store/apps/details?id=${BuildConfig.APPLICATION_ID}"
        }

    val isStoreBuild: Boolean
        get() = !enableCdnOta
}
