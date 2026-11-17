package com.pegasusx.retailer.data.api


import com.pegasusx.retailer.BuildConfig
import com.pegasusx.retailer.data.local.TokenManager
import com.pegasusx.retailer.data.model.ProblemDetail
import com.pegasusx.retailer.data.model.ProblemDetailException
import dagger.Module
import dagger.Provides
import dagger.hilt.EntryPoint
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import com.jakewharton.retrofit2.converter.kotlinx.serialization.asConverterFactory
import okhttp3.Authenticator
import okhttp3.CertificatePinner
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.Response
import okhttp3.Route
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import java.util.concurrent.TimeUnit
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
object NetworkModule {

    @Provides
    @Singleton
    fun provideJson(): Json = Json {
        ignoreUnknownKeys = true
        coerceInputValues = true
        encodeDefaults = true
    }

    @Provides
    @Singleton
    fun provideOkHttpClient(tokenManager: TokenManager, json: Json): OkHttpClient {
        return OkHttpClient.Builder()
            .connectTimeout(15, TimeUnit.SECONDS)
            .readTimeout(30, TimeUnit.SECONDS)
            .writeTimeout(30, TimeUnit.SECONDS)
            .pingInterval(25, TimeUnit.SECONDS)
            .apply {
                configuredCertificatePinner(BuildConfig.BASE_URL, BuildConfig.API_TLS_SPKI_PINS)
                    ?.let { certificatePinner(it) }
            }
            .addInterceptor(com.pegasusx.retailer.ui.design.CellPinInterceptor(BuildConfig.BASE_URL) { tokenManager.getPreferredToken() })
            .addInterceptor { chain ->
                val token = tokenManager.getPreferredToken()
                val request = chain.request().newBuilder()
                    .addHeader("X-Trace-Id", java.util.UUID.randomUUID().toString())
                    .apply { if (token != null) addHeader("Authorization", "Bearer $token") }
                    .build()
                chain.proceed(request)
            }
            .authenticator(TokenRefreshAuthenticator(tokenManager, json))
            .addInterceptor(com.pegasusx.retailer.ui.design.PegasusHttpResilienceInterceptor())
            .addInterceptor(ProblemDetailInterceptor(json))
            .addInterceptor(HttpLoggingInterceptor().apply {
                level = if (BuildConfig.DEBUG) {
                    HttpLoggingInterceptor.Level.BODY
                } else {
                    HttpLoggingInterceptor.Level.NONE
                }
            })
            .build()
    }

    @Provides
    @Singleton
    fun provideRetrofit(okHttpClient: OkHttpClient, json: Json): Retrofit {
        return Retrofit.Builder()
            .baseUrl(BuildConfig.BASE_URL)
            .client(okHttpClient)
            .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
            .build()
    }

    @Provides
    @Singleton
    fun providePegasusApi(retrofit: Retrofit): PegasusApi {
        return retrofit.create(PegasusApi::class.java)
    }

    @Provides
    @Singleton
    fun provideRetailerWebSocket(tokenManager: TokenManager, json: Json): RetailerWebSocket {
        return RetailerWebSocket(tokenManager, json)
    }

    @Provides
    @Singleton
    fun provideCartRealtimeEventSource(webSocket: RetailerWebSocket): CartRealtimeEventSource = webSocket
}

internal fun configuredCertificatePinner(baseUrl: String, pinsCsv: String): CertificatePinner? {
    val parsed = baseUrl.trim().toHttpUrlOrNull() ?: return null
    if (!parsed.isHttps) return null
    val host = parsed.host.lowercase()
    if (host == "localhost" || host == "127.0.0.1" || host == "10.0.2.2" || host == "::1") {
        return null
    }
    val pins = pinsCsv.split(',').map { it.trim() }.filter { it.startsWith("sha256/") }
    if (pins.isEmpty()) return null
    val builder = CertificatePinner.Builder()
    for (pin in pins) {
        builder.add(host, pin)
    }
    return builder.build()
}

// ── Silent 401 → refresh → retry ──

private class TokenRefreshAuthenticator(
    private val tokenManager: TokenManager,
    private val json: Json,
) : Authenticator {
    private val refreshClient = OkHttpClient.Builder()
        .connectTimeout(10, TimeUnit.SECONDS)
        .readTimeout(10, TimeUnit.SECONDS)
        .apply {
            configuredCertificatePinner(BuildConfig.BASE_URL, BuildConfig.API_TLS_SPKI_PINS)
                ?.let { certificatePinner(it) }
        }
        .build()

    @Synchronized
    override fun authenticate(route: Route?, response: Response): Request? {
        // Prevent infinite loops
        if (response.request.header("X-Refresh-Attempted") != null) {
            tokenManager.clearToken()
            return null
        }
        val currentToken = tokenManager.getToken() ?: return null
        val requestToken = response.request.header("Authorization")
            ?.removePrefix("Bearer ")
            ?.trim()
        if (!requestToken.isNullOrBlank() && requestToken != currentToken) {
            return response.request.newBuilder()
                .header("Authorization", "Bearer $currentToken")
                .header("X-Refresh-Attempted", "true")
                .build()
        }

        val refreshUrl = "${BuildConfig.BASE_URL}v1/auth/retailer/refresh"
        val body = "{}".toRequestBody("application/json".toMediaType())
        val refreshRequest = Request.Builder()
            .url(refreshUrl)
            .post(body)
            .addHeader("Authorization", "Bearer $currentToken")
            .addHeader("Content-Type", "application/json")
            .build()

        return try {
            val refreshResponse = refreshClient.newCall(refreshRequest).execute()
            if (refreshResponse.isSuccessful) {
                val responseBody = refreshResponse.body?.string() ?: return null
                val jsonElement = json.parseToJsonElement(responseBody)
                val newToken = jsonElement.jsonObject["token"]?.jsonPrimitive?.content ?: return null
                tokenManager.saveToken(newToken)
                response.request.newBuilder()
                    .header("Authorization", "Bearer $newToken")
                    .header("X-Refresh-Attempted", "true")
                    .build()
            } else {
                tokenManager.clearToken()
                null
            }
        } catch (_: Exception) {
            null
        }
    }
}

// ── RFC 7807 Problem Detail interceptor ──

private class ProblemDetailInterceptor(private val json: Json) : okhttp3.Interceptor {
    override fun intercept(chain: okhttp3.Interceptor.Chain): Response {
        val response = chain.proceed(chain.request())
        if (response.code == 429) {
            val bodyStr = response.peekBody(8192).string()
            val isRateLimited = runCatching {
                json.parseToJsonElement(bodyStr).jsonObject["error"]?.jsonPrimitive?.content == "rate_limit_exceeded"
            }.getOrDefault(false)
            if (isRateLimited) {
                throw ProblemDetailException(
                    ProblemDetail(
                        type = "about:blank",
                        title = "Too Many Requests",
                        status = 429,
                        detail = "Too many requests. Please try again later.",
                        instance = null,
                        code = "rate_limit_exceeded",
                        // UI layer maps this key to mobile_retailer_ui_too_many_requests for display.
                        messageKey = "mobile_retailer_ui_too_many_requests",
                        retryable = true,
                        action = null,
                    ),
                )
            }
        }
        val contentType = response.header("Content-Type") ?: return response
        if (!contentType.contains("application/problem+json")) return response
        val body = response.peekBody(8192).string()
        val problem = try {
            json.decodeFromString<ProblemDetail>(body).copy(rawBody = body)
        } catch (_: Exception) {
            return response
        }
        throw ProblemDetailException(problem)
    }
}

@EntryPoint
@InstallIn(SingletonComponent::class)
interface PegasusApiEntryPoint {
    fun pegasusApi(): PegasusApi
}
