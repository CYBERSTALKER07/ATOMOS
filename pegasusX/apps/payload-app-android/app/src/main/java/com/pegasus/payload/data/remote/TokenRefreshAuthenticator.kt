package com.pegasus.payload.data.remote

import com.pegasus.payload.BuildConfig
import com.pegasus.payload.data.local.SecureStore
import com.pegasus.payload.data.model.RefreshTokenRequest
import com.pegasus.payload.data.model.RefreshTokenResponse
import kotlinx.serialization.json.Json
import okhttp3.Authenticator
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.Response
import okhttp3.Route
import java.util.concurrent.TimeUnit
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class TokenRefreshAuthenticator @Inject constructor(
    private val secureStore: SecureStore,
    private val json: Json,
) : Authenticator {
    override fun authenticate(route: Route?, response: Response): Request? {
        if (response.code != 401) return null
        if (response.request.header("X-Refresh-Attempted") != null) {
            secureStore.clear()
            return null
        }
        val refresh = secureStore.refreshToken ?: return null
        val baseUrl = BuildConfig.API_BASE_URL.trimEnd('/') + "/"
        val body = json.encodeToString(RefreshTokenRequest.serializer(), RefreshTokenRequest(refresh))
            .toRequestBody("application/json".toMediaType())
        val refreshRequest = Request.Builder()
            .url("${baseUrl}v1/auth/payloader/refresh")
            .post(body)
            .build()
        val refreshClient = OkHttpClient.Builder()
            .connectTimeout(10, TimeUnit.SECONDS)
            .readTimeout(10, TimeUnit.SECONDS)
            .build()
        return try {
            val refreshResponse = refreshClient.newCall(refreshRequest).execute()
            if (!refreshResponse.isSuccessful) {
                secureStore.clear()
                return null
            }
            val responseBody = refreshResponse.body?.string() ?: return null
            val auth = json.decodeFromString(RefreshTokenResponse.serializer(), responseBody)
            secureStore.token = auth.token
            if (auth.refreshToken.isNotBlank()) {
                secureStore.refreshToken = auth.refreshToken
            }
            response.request.newBuilder()
                .header("Authorization", "Bearer ${auth.token}")
                .header("X-Refresh-Attempted", "true")
                .build()
        } catch (_: Exception) {
            secureStore.clear()
            null
        }
    }
}
