package com.pegasusx.driver.services

import android.content.Context
import android.util.Log
import androidx.hilt.work.HiltWorker
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import com.pegasusx.driver.data.local.TelemetryDao
import com.pegasusx.driver.data.model.TelemetryPayload
import com.pegasusx.driver.data.remote.TelemetrySocket
import com.pegasusx.driver.data.remote.TokenHolder
import dagger.assisted.Assisted
import dagger.assisted.AssistedInject

@HiltWorker
class TelemetrySyncWorker @AssistedInject constructor(
    @Assisted appContext: Context,
    @Assisted params: WorkerParameters,
    private val telemetryDao: TelemetryDao,
    private val telemetrySocket: TelemetrySocket
) : CoroutineWorker(appContext, params) {

    companion object {
        const val TAG = "TelemetrySyncWorker"
        const val WORK_NAME = "telemetry_sync"
    }

    override suspend fun doWork(): Result {
        val locations = telemetryDao.getAll()
        if (locations.isEmpty()) return Result.success()

        Log.d(TAG, "Draining ${locations.size} offline telemetry points")
        val driverId = TokenHolder.userId?.takeIf { it.isNotBlank() } ?: return Result.failure()

        // If the socket isn't connected, we can't send them this way.
        // We will just loop and send them via the socket if connected,
        // or we could add a REST endpoint for batch sync in the future.
        if (!telemetrySocket.isConnected()) {
            Log.w(TAG, "Socket not connected, deferring telemetry sync")
            return Result.retry()
        }

        val syncedIds = mutableListOf<String>()

        for (loc in locations) {
            val payload = TelemetryPayload(
                driverId = driverId,
                latitude = loc.latitude,
                longitude = loc.longitude,
                timestamp = loc.timestamp,
                speed = loc.speed,
                bearing = loc.bearing
            )
            val sent = telemetrySocket.send(payload)
            if (sent) {
                syncedIds.add(loc.id)
            } else {
                Log.w(TAG, "Socket disconnected midway through batch")
                break
            }
        }

        if (syncedIds.isNotEmpty()) {
            telemetryDao.deleteByIds(syncedIds)
            Log.d(TAG, "Successfully synced ${syncedIds.size} telemetry points")
        }

        val remaining = telemetryDao.getAll()
        return if (remaining.isNotEmpty()) Result.retry() else Result.success()
    }
}
