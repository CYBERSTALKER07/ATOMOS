package com.pegasusx.warehouse.data.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class GeocodeAutocompleteResponse(
    val predictions: List<GeocodePrediction> = emptyList(),
)

@Serializable
data class GeocodePrediction(
    @SerialName("place_id") val placeId: String = "",
    val description: String = "",
)

@Serializable
data class ResolvedLocation(
    val address: String = "",
    val lat: Double = 0.0,
    val lng: Double = 0.0,
    @SerialName("place_id") val placeId: String? = null,
)

@Serializable
data class WarehouseLocationResponse(
    @SerialName("warehouse_id") val warehouseId: String = "",
    val name: String = "",
    val address: String = "",
    @SerialName("place_id") val placeId: String? = null,
    val lat: Double = 0.0,
    val lng: Double = 0.0,
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class WarehouseLocationPatchRequest(
    val address: String,
    @SerialName("place_id") val placeId: String? = null,
    val lat: Double,
    val lng: Double,
)
