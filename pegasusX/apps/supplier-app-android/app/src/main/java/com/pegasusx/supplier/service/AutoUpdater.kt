package com.pegasusx.supplier.service

import android.app.Activity
import android.app.DownloadManager
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.content.pm.PackageManager
import android.database.Cursor
import android.net.Uri
import android.os.Build
import android.os.Environment
import android.provider.Settings
import android.util.Log
import androidx.core.content.FileProvider
import com.pegasusx.supplier.BuildConfig
import com.pegasusx.supplier.data.model.ClientPolicyResponse
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.json.JSONObject
import java.io.File
import java.net.HttpURLConnection
import java.net.URL
import java.security.MessageDigest

/**
 * Website-only enterprise OTA for supplier Android.
 *
 * Flow:
 * 1. [checkFromPolicy] after client-policy (channel=enterprise)
 * 2. Fetch updater.json from policy.update_url (CDN)
 * 3. If version_code > installed → download APK, verify sha256, prompt install
 *
 * Not used for Play Store builds.
 */
class AutoUpdater(private val context: Context) {
    private val downloadManager =
        context.getSystemService(Context.DOWNLOAD_SERVICE) as DownloadManager
    private val tag = "SupplierAutoUpdater"
    private val prefs = context.getSharedPreferences("supplier_enterprise_updater", Context.MODE_PRIVATE)

    data class Manifest(
        val versionCode: Int,
        val versionName: String,
        val apkUrl: String,
        val sha256: String,
        val notes: String,
    )

    data class UpdateState(
        val available: Boolean,
        val force: Boolean,
        val deferred: Boolean,
        val message: String?,
        val manifest: Manifest?,
        val policy: ClientPolicyResponse?,
    )

    private val onDownloadComplete = object : BroadcastReceiver() {
        override fun onReceive(ctx: Context, intent: Intent) {
            val id = intent.getLongExtra(DownloadManager.EXTRA_DOWNLOAD_ID, -1)
            val expectedId = prefs.getLong("download_id", -1)
            if (id == expectedId && id != -1L) {
                CoroutineScope(Dispatchers.IO).launch {
                verifyAndInstall(id, prefs.getString("expected_hash", "") ?: "")
            }
            }
        }
    }

    private var receiverRegistered = false

    fun register() {
        if (receiverRegistered) return
        val filter = IntentFilter(DownloadManager.ACTION_DOWNLOAD_COMPLETE)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            context.registerReceiver(onDownloadComplete, filter, Context.RECEIVER_NOT_EXPORTED)
        } else {
            @Suppress("UnspecifiedRegisterReceiverFlag")
            context.registerReceiver(onDownloadComplete, filter)
        }
        receiverRegistered = true
    }

    fun cleanup() {
        if (!receiverRegistered) return
        try {
            context.unregisterReceiver(onDownloadComplete)
        } catch (_: Exception) {
        }
        receiverRegistered = false
    }

    /**
     * Evaluate policy + CDN manifest. Does not auto-download unless [autoDownload] is true.
     */
    suspend fun checkFromPolicy(
        policy: ClientPolicyResponse,
        autoDownload: Boolean = false,
    ): UpdateState = withContext(Dispatchers.IO) {
        val force = policy.forceUpdate && !policy.updateDeferred
        val message = when {
            policy.outdated || policy.forceUpdate -> buildString {
                append(if (force) "Update required" else "Update available")
                if (policy.minimumVersion.isNotBlank()) {
                    append(" — minimum version ${policy.minimumVersion}")
                }
                if (policy.recommendedVersion.isNotBlank() &&
                    policy.recommendedVersion != "0.0.0"
                ) {
                    append(" (recommended ${policy.recommendedVersion})")
                }
                policy.deferReason?.takeIf { it.isNotBlank() }?.let { append(". $it") }
            }
            else -> null
        }

        if (!policy.outdated && !policy.forceUpdate) {
            return@withContext UpdateState(
                available = false,
                force = false,
                deferred = policy.updateDeferred,
                message = null,
                manifest = null,
                policy = policy,
            )
        }

        // Play Store builds: never fetch/install CDN APKs — only surface policy + store URL.
        if (!EnterpriseUpdateConfig.enableCdnOta) {
            return@withContext UpdateState(
                available = true,
                force = force,
                deferred = policy.updateDeferred,
                message = message,
                manifest = null,
                policy = policy,
            )
        }

        val manifestUrl = policy.updateUrl?.takeIf { it.isNotBlank() }
            ?: EnterpriseUpdateConfig.DEFAULT_MANIFEST_URL
        val manifest = runCatching { fetchManifest(manifestUrl) }.getOrElse { err ->
            Log.e(tag, "manifest fetch failed: ${err.message}")
            null
        }

        val installedCode = installedVersionCode()
        val newer = manifest != null && manifest.versionCode > installedCode

        if (newer && autoDownload && manifest != null && !policy.updateDeferred) {
            startDownload(manifest)
        }

        UpdateState(
            available = newer || policy.outdated,
            force = force,
            deferred = policy.updateDeferred,
            message = message,
            manifest = manifest,
            policy = policy,
        )
    }

    /** User tapped Update — download + install from last known / fetched manifest. */
    suspend fun startUpdate(manifest: Manifest? = null) = withContext(Dispatchers.IO) {
        if (!EnterpriseUpdateConfig.enableCdnOta) {
            withContext(Dispatchers.Main) {
                openStoreListing()
            }
            return@withContext
        }

        register()
        val m = manifest ?: run {
            val url = prefs.getString("last_manifest_url", null)
                ?: EnterpriseUpdateConfig.DEFAULT_MANIFEST_URL
            fetchManifest(url)
        }
        startDownload(m)
    }

    fun ensureInstallPermission(activity: Activity): Boolean {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return true
        if (context.packageManager.canRequestPackageInstalls()) return true
        val intent = Intent(Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES).apply {
            data = Uri.parse("package:${context.packageName}")
        }
        activity.startActivity(intent)
        return false
    }


    /** Open Play Store listing (store builds). No-op URL falls back to package id. */
    fun openStoreListing() {
        val url = EnterpriseUpdateConfig.storeListingUrl
        val intent = Intent(Intent.ACTION_VIEW, Uri.parse(url)).apply {
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }
        try {
            context.startActivity(intent)
        } catch (e: Exception) {
            Log.e(tag, "open store failed: ${e.message}")
        }
    }

    private fun fetchManifest(url: String): Manifest {
        prefs.edit().putString("last_manifest_url", url).apply()
        val conn = (URL(url).openConnection() as HttpURLConnection).apply {
            connectTimeout = 15_000
            readTimeout = 15_000
            requestMethod = "GET"
            setRequestProperty("Accept", "application/json")
        }
        try {
            val code = conn.responseCode
            if (code !in 200..299) {
                throw IllegalStateException("manifest HTTP $code")
            }
            val body = conn.inputStream.bufferedReader().use { it.readText() }
            val json = JSONObject(body)
            return Manifest(
                versionCode = json.getInt("version_code"),
                versionName = json.optString("version_name", ""),
                apkUrl = json.getString("apk_url"),
                sha256 = json.getString("sha256"),
                notes = json.optString("notes", ""),
            )
        } finally {
            conn.disconnect()
        }
    }

    private fun installedVersionCode(): Int {
        return try {
            val info = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                context.packageManager.getPackageInfo(
                    context.packageName,
                    PackageManager.PackageInfoFlags.of(0),
                )
            } else {
                @Suppress("DEPRECATION")
                context.packageManager.getPackageInfo(context.packageName, 0)
            }
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
                info.longVersionCode.toInt()
            } else {
                @Suppress("DEPRECATION")
                info.versionCode
            }
        } catch (_: Exception) {
            BuildConfig.VERSION_CODE
        }
    }

    private fun startDownload(manifest: Manifest) {
        Log.i(tag, "Downloading supplier enterprise update ${manifest.versionCode} ${manifest.apkUrl}")
        val request = DownloadManager.Request(Uri.parse(manifest.apkUrl))
            .setTitle("Supplier Update ${manifest.versionName.ifBlank { manifest.versionCode.toString() }}")
            .setDescription(manifest.notes.ifBlank { "Enterprise website update" })
            .setNotificationVisibility(DownloadManager.Request.VISIBILITY_VISIBLE_NOTIFY_COMPLETED)
            .setDestinationInExternalFilesDir(context, Environment.DIRECTORY_DOWNLOADS, "supplier-update.apk")
            .setAllowedOverMetered(true)
            .setAllowedOverRoaming(true)

        val downloadId = downloadManager.enqueue(request)
        prefs.edit()
            .putLong("download_id", downloadId)
            .putString("expected_hash", manifest.sha256)
            .apply()
    }

    private fun verifyAndInstall(downloadId: Long, expectedHash: String) {
        val cursor: Cursor = downloadManager.query(DownloadManager.Query().setFilterById(downloadId))
        try {
            if (!cursor.moveToFirst()) return
            val status = cursor.getInt(cursor.getColumnIndexOrThrow(DownloadManager.COLUMN_STATUS))
            if (status != DownloadManager.STATUS_SUCCESSFUL) return
            val localUri = cursor.getString(cursor.getColumnIndexOrThrow(DownloadManager.COLUMN_LOCAL_URI))
                ?: return

            // Materialize into app cache so we can hash + FileProvider-install reliably.
            val dest = File(context.cacheDir, "supplier-update-install.apk")
            if (!copyUriToFile(Uri.parse(localUri), dest)) {
                Log.e(tag, "failed to copy download to cache")
                return
            }
            if (!verifyHash(dest, expectedHash)) {
                Log.e(tag, "sha256 mismatch — deleting package")
                dest.delete()
                return
            }
            installApk(dest)
        } finally {
            cursor.close()
        }
    }

    private fun copyUriToFile(uri: Uri, dest: File): Boolean {
        return try {
            context.contentResolver.openInputStream(uri)?.use { input ->
                dest.outputStream().use { output -> input.copyTo(output) }
            } ?: return false
            true
        } catch (e: Exception) {
            Log.e(tag, "copy failed: ${e.message}")
            false
        }
    }

    private fun verifyHash(file: File, expectedHash: String): Boolean {
        if (expectedHash.isBlank()) return false
        return try {
            val digest = MessageDigest.getInstance("SHA-256")
            file.inputStream().use { input ->
                val buf = ByteArray(8192)
                while (true) {
                    val n = input.read(buf)
                    if (n <= 0) break
                    digest.update(buf, 0, n)
                }
            }
            val hex = digest.digest().joinToString("") { "%02x".format(it) }
            hex.equals(expectedHash.trim(), ignoreCase = true)
        } catch (e: Exception) {
            Log.e(tag, "hash failed: ${e.message}")
            false
        }
    }

    private fun installApk(file: File) {
        val uri = FileProvider.getUriForFile(
            context,
            "${context.packageName}.fileprovider",
            file,
        )
        val intent = Intent(Intent.ACTION_VIEW).apply {
            setDataAndType(uri, "application/vnd.android.package-archive")
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
        }
        context.startActivity(intent)
    }
}
