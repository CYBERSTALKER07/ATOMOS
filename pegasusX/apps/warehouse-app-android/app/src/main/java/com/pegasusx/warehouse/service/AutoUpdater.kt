package com.pegasusx.warehouse.service

import android.app.DownloadManager
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.database.Cursor
import android.net.Uri
import android.os.Environment
import android.util.Log
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONObject
import java.io.File
import java.net.URL
import java.security.MessageDigest

class AutoUpdater(private val context: Context) {
    private val downloadManager = context.getSystemService(Context.DOWNLOAD_SERVICE) as DownloadManager
    private val TAG = "AutoUpdater"
    private val UPDATE_MANIFEST_URL = "https://storage.googleapis.com/pegasusx-ssmr-app-updates/android/warehouse/updater.json"

    private val onDownloadComplete = object : BroadcastReceiver() {
        override fun onReceive(context: Context, intent: Intent) {
            val id = intent.getLongExtra(DownloadManager.EXTRA_DOWNLOAD_ID, -1)
            val prefs = context.getSharedPreferences("updater_prefs", Context.MODE_PRIVATE)
            val expectedId = prefs.getLong("download_id", -1)

            if (id == expectedId) {
                verifyAndInstall(id, prefs.getString("expected_hash", "") ?: "")
            }
        }
    }

    init {
        context.registerReceiver(
            onDownloadComplete,
            IntentFilter(DownloadManager.ACTION_DOWNLOAD_COMPLETE),
            Context.RECEIVER_EXPORTED
        )
    }

    suspend fun checkForUpdates(currentVersion: Int) {
        withContext(Dispatchers.IO) {
            try {
                val response = URL(UPDATE_MANIFEST_URL).readText()
                val manifest = JSONObject(response)

                val latestVersion = manifest.getInt("version_code")
                val downloadUrl = manifest.getString("apk_url")
                val expectedHash = manifest.getString("sha256")

                if (latestVersion > currentVersion) {
                    Log.i(TAG, "Update found! Downloading version $latestVersion...")
                    startDownload(downloadUrl, expectedHash)
                }
            } catch (e: Exception) {
                Log.e(TAG, "Failed to check for updates: ${e.message}")
            }
        }
    }

    private fun startDownload(url: String, expectedHash: String) {
        val request = DownloadManager.Request(Uri.parse(url))
            .setTitle("Warehouse Update")
            .setNotificationVisibility(DownloadManager.Request.VISIBILITY_VISIBLE_NOTIFY_COMPLETED)
            .setDestinationInExternalFilesDir(context, Environment.DIRECTORY_DOWNLOADS, "update.apk")

        val downloadId = downloadManager.enqueue(request)
        context.getSharedPreferences("updater_prefs", Context.MODE_PRIVATE).edit()
            .putLong("download_id", downloadId)
            .putString("expected_hash", expectedHash)
            .apply()
    }

    private fun verifyAndInstall(downloadId: Long, expectedHash: String) {
        val cursor: Cursor = downloadManager.query(DownloadManager.Query().setFilterById(downloadId))
        if (cursor.moveToFirst() && cursor.getInt(cursor.getColumnIndexOrThrow(DownloadManager.COLUMN_STATUS)) == DownloadManager.STATUS_SUCCESSFUL) {
            val file = File(Uri.parse(cursor.getString(cursor.getColumnIndexOrThrow(DownloadManager.COLUMN_LOCAL_URI))).path ?: "")
            if (file.exists() && verifyHash(file, expectedHash)) installApk(file) else file.delete()
        }
        cursor.close()
    }

    private fun verifyHash(file: File, expectedHash: String): Boolean {
        try {
            val bytes = file.readBytes()
            val md = MessageDigest.getInstance("SHA-256")
            return md.digest(bytes).joinToString("") { "%02x".format(it) }.equals(expectedHash, ignoreCase = true)
        } catch (e: Exception) { return false }
    }

    private fun installApk(file: File) {
        val intent = Intent(Intent.ACTION_VIEW)
        intent.setDataAndType(Uri.fromFile(file), "application/vnd.android.package-archive")
        intent.flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_GRANT_READ_URI_PERMISSION
        context.startActivity(intent)
    }

    fun cleanup() {
        try { context.unregisterReceiver(onDownloadComplete) } catch (e: Exception) {}
    }
}
