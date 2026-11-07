package com.pegasus.payload.services

import android.content.Context
import android.util.Log
import androidx.hilt.work.HiltWorker
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import com.pegasus.payload.data.local.QueuedActionDao
import com.pegasus.payload.data.remote.PayloadApi
import com.pegasusx.mobilekit.offline.OfflineHttpSemantics
import dagger.assisted.Assisted
import dagger.assisted.AssistedInject
import okhttp3.MediaType.Companion.toMediaTypeOrNull
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject
import retrofit2.HttpException

@HiltWorker
class OfflineSyncWorker @AssistedInject constructor(
    @Assisted appContext: Context,
    @Assisted params: WorkerParameters,
    private val api: PayloadApi,
    private val queuedActionDao: QueuedActionDao
) : CoroutineWorker(appContext, params) {

    companion object {
        const val TAG = "OfflineSyncWorker"
        const val WORK_NAME = "payload_offline_sync"
    }

    override suspend fun doWork(): Result {
        val pending = queuedActionDao.getAll()
        if (pending.isEmpty()) return Result.success()

        Log.d(TAG, "Draining ${pending.size} queued action(s)")

        var failures = 0
        for (action in pending) {
            try {
                val payload = applyCapturedCoords(action.bodyJson, action.capturedLat, action.capturedLng, action.capturedAtMs)
                val body = payload?.toRequestBody("application/json".toMediaTypeOrNull())

                when (action.method.uppercase()) {
                    "POST" -> api.rawPost(action.endpoint, body, action.id)
                    "PUT" -> api.rawPut(action.endpoint, body, action.id)
                    "PATCH" -> api.rawPatch(action.endpoint, body, action.id)
                    "DELETE" -> api.rawDelete(action.endpoint, action.id)
                    else -> Log.w(TAG, "Unsupported method: ${action.method}")
                }

                queuedActionDao.deleteById(action.id)
                Log.d(TAG, "Successfully synced action ${action.id}")
            } catch (e: HttpException) {
                when (OfflineHttpSemantics.outcomeForHttp(e.code())) {
                    OfflineHttpSemantics.FlushOutcome.ACK -> {
                        queuedActionDao.deleteById(action.id)
                        Log.d(TAG, "409/2xx idempotent for ${action.id}, purged")
                    }
                    OfflineHttpSemantics.FlushOutcome.RETRY -> {
                        failures++
                        Log.w(TAG, "Server error ${e.code()} for ${action.id}, will retry")
                    }
                    OfflineHttpSemantics.FlushOutcome.DEAD -> {
                        queuedActionDao.deleteById(action.id)
                        Log.w(TAG, "Client error ${e.code()} for ${action.id}, discarded")
                    }
                }
            } catch (e: Exception) {
                failures++
                Log.e(TAG, "Failed to sync ${action.id}: ${e.message}")
            }
        }

        return if (failures > 0) Result.retry() else Result.success()
    }

    private fun applyCapturedCoords(
        bodyJson: String?,
        lat: Double?,
        lng: Double?,
        atMs: Long?,
    ): String? {
        if (bodyJson.isNullOrBlank()) return bodyJson
        if (lat == null || lng == null) return bodyJson
        return runCatching {
            val obj = JSONObject(bodyJson)
            obj.put("latitude", lat)
            obj.put("longitude", lng)
            atMs?.let { obj.put("captured_at_ms", it) }
            obj.toString()
        }.getOrDefault(bodyJson)
    }
}
