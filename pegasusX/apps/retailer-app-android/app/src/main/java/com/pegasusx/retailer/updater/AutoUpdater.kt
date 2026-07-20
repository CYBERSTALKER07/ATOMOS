package com.pegasusx.retailer.updater

import android.content.Context
import com.pegasusx.retailer.service.AutoUpdater as EnterpriseAutoUpdater

/**
 * Deprecated package path — delegates to [com.pegasusx.retailer.service.AutoUpdater]
 * for website-only enterprise OTA.
 */
@Deprecated(
    message = "Use com.pegasusx.retailer.service.AutoUpdater",
    replaceWith = ReplaceWith("AutoUpdater", "com.pegasusx.retailer.service.AutoUpdater"),
)
class AutoUpdater(context: Context) {
    private val impl = EnterpriseAutoUpdater(context.applicationContext)

    fun checkForUpdates() {
        // No-op without policy; navigation ViewModel drives enterprise checks.
    }

    fun downloadAndInstallApk(apkUrl: String, version: String) {
        // Legacy helper retained for call sites; prefer startUpdate(manifest).
        checkForUpdates()
    }

    fun cleanup() {
        impl.cleanup()
    }
}
