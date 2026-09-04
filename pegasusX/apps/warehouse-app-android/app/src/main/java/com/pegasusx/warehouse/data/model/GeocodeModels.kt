package com.pegasusx.warehouse.data.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class ForwardGeocodeRequest(
    val address: String,
)

@Serializable
data class WarehouseSetupRequest(
    val name: String = "",
    val address: String,
    @SerialName("place_id") val placeId: String? = null,
    val lat: Double,
    val lng: Double,
)

@Serializable
data class WarehouseSetupResponse(
    @SerialName("warehouse_id") val warehouseId: String = "",
    val token: String? = null,
    @SerialName("refresh_token") val refreshToken: String? = null,
    @SerialName("is_configured") val isConfigured: Boolean = false,
)

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
    @SerialName("country_code") val countryCode: String = "",
    @SerialName("pack_country_code") val packCountryCode: String = "",
    @SerialName("currency_code") val currencyCode: String = "",
)

@Serializable
data class WarehouseLocationPatchRequest(
    val address: String,
    @SerialName("place_id") val placeId: String? = null,
    val lat: Double,
    val lng: Double,
)
