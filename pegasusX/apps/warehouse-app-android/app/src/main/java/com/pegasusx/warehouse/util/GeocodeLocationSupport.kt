package com.pegasusx.warehouse.util

import com.pegasusx.warehouse.data.remote.GeocodeApi
import com.pegasusx.warehouse.ui.components.AddressLocationValue

object GeocodeLocationSupport {
    fun hasValidCoordinates(lat: Double, lng: Double): Boolean {
        if (!lat.isFinite() || !lng.isFinite()) return false
        if (lat == 0.0 && lng == 0.0) return false
        return lat in -90.0..90.0 && lng in -180.0..180.0
    }

    suspend fun resolveLocationValue(
        geocodeApi: GeocodeApi,
        value: AddressLocationValue,
    ): AddressLocationValue? {
        val address = value.address.trim()
        if (address.isEmpty()) return null
        if (hasValidCoordinates(value.lat, value.lng)) return value

        val top = runCatching { geocodeApi.autocomplete(address).predictions.firstOrNull() }.getOrNull()
        if (!top?.placeId.isNullOrBlank()) {
            val byPlace = runCatching { geocodeApi.resolvePlace(top!!.placeId) }.getOrNull()
            if (byPlace != null && hasValidCoordinates(byPlace.lat, byPlace.lng)) {
                return AddressLocationValue(
                    address = byPlace.address.ifBlank { address },
                    lat = byPlace.lat,
                    lng = byPlace.lng,
                    placeId = byPlace.placeId,
                )
            }
        }

        val byAddress = runCatching { geocodeApi.forward(com.pegasusx.warehouse.data.model.ForwardGeocodeRequest(address = address)) }.getOrNull()
        if (byAddress != null && hasValidCoordinates(byAddress.lat, byAddress.lng)) {
            return AddressLocationValue(
                address = byAddress.address.ifBlank { address },
                lat = byAddress.lat,
                lng = byAddress.lng,
                placeId = byAddress.placeId,
            )
        }
        return null
    }
}
