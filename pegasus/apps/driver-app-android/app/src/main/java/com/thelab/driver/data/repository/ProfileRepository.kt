package com.thelab.driver.data.repository

import com.thelab.driver.data.model.DriverProfileResponse
import com.thelab.driver.data.remote.DriverApi
import com.thelab.driver.data.remote.TokenHolder
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.flow
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Polls GET /v1/driver/profile every 60s.
 * When the backend returns a different vehicleId, updates TokenHolder
 * which drives Compose recomposition.
 */
@Singleton
class ProfileRepository @Inject constructor(
    private val api: DriverApi
) {
    companion object {
        private const val POLL_INTERVAL_MS = 60_000L
    }

    /** Emits profile snapshots every 60s. Collect in a lifecycle-aware scope. */
    fun pollProfile(): Flow<DriverProfileResponse> = flow {
        while (true) {
            if (TokenHolder.token != null) {
                try {
                    val profile = api.getProfile()
                    applyIfChanged(profile)
                    emit(profile)
                } catch (_: Exception) {
                    // Silently ignore — next tick retries in 60s
                }
            }
            delay(POLL_INTERVAL_MS)
        }
    }

    private fun applyIfChanged(profile: DriverProfileResponse) {
        if (TokenHolder.vehicleId != profile.vehicleId
            || TokenHolder.vehicleClass != profile.vehicleClass
            || TokenHolder.maxVolumeVU != profile.maxVolumeVU
            || TokenHolder.vehicleType != profile.vehicleType
            || TokenHolder.licensePlate != profile.licensePlate
            || TokenHolder.warehouseId != profile.warehouseId
            || TokenHolder.warehouseName != profile.warehouseName
            || TokenHolder.warehouseLat != profile.warehouseLat
            || TokenHolder.warehouseLng != profile.warehouseLng
        ) {
            TokenHolder.vehicleId = profile.vehicleId
            TokenHolder.vehicleClass = profile.vehicleClass
            TokenHolder.maxVolumeVU = profile.maxVolumeVU
            TokenHolder.vehicleType = profile.vehicleType
            TokenHolder.licensePlate = profile.licensePlate
            TokenHolder.warehouseId = profile.warehouseId
            TokenHolder.warehouseName = profile.warehouseName
            TokenHolder.warehouseLat = profile.warehouseLat
            TokenHolder.warehouseLng = profile.warehouseLng
        }
    }
}
