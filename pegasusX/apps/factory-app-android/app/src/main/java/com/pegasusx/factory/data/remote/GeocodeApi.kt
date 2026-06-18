package com.pegasusx.factory.data.remote

import com.pegasusx.factory.data.model.GeocodeAutocompleteResponse
import com.pegasusx.factory.data.model.ResolvedLocation
import retrofit2.http.GET
import retrofit2.http.Query

interface GeocodeApi {
    @GET("v1/platform/geocode/autocomplete")
    suspend fun autocomplete(@Query("input") input: String): GeocodeAutocompleteResponse

    @GET("v1/platform/geocode/place")
    suspend fun resolvePlace(@Query("place_id") placeId: String): ResolvedLocation

    @GET("v1/platform/geocode/reverse")
    suspend fun reverse(@Query("lat") lat: Double, @Query("lng") lng: Double): ResolvedLocation
}
