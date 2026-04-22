package com.thelab.retailer.data.api

import com.thelab.retailer.BuildConfig
import com.thelab.retailer.data.local.TokenManager
import com.thelab.retailer.data.model.ProblemDetail
import com.thelab.retailer.data.model.ProblemDetailException
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import com.jakewharton.retrofit2.converter.kotlinx.serialization.asConverterFactory
import okhttp3.Authenticator
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
            .addInterceptor { chain ->
                val token = tokenManager.getPreferredToken()
                val request = chain.request().newBuilder()
                    .addHeader("X-Trace-Id", java.util.UUID.randomUUID().toString())
                    .apply { if (token != null) addHeader("Authorization", "Bearer $token") }
                    .build()
                chain.proceed(request)
            }
            .authenticator(TokenRefreshAuthenticator(tokenManager, json))
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
    fun provideLabApi(retrofit: Retrofit): LabApi {
        return retrofit.create(LabApi::class.java)
    }

    @Provides
    @Singleton
    fun provideRetailerWebSocket(tokenManager: TokenManager, json: Json): RetailerWebSocket {
        return RetailerWebSocket(tokenManager, json)
    }
}

// ── Silent 401 → refresh → retry ──

private class TokenRefreshAuthenticator(
    private val tokenManager: TokenManager,
    private val json: Json,
) : Authenticator {
    override fun authenticate(route: Route?, response: Response): Request? {
        // Prevent infinite loops
        if (response.request.header("X-Refresh-Attempted") != null) {
            tokenManager.clearToken()
            return null
        }
        val currentToken = tokenManager.getToken() ?: return null

        val refreshUrl = "${BuildConfig.BASE_URL}v1/auth/refresh"
        val body = "{}".toRequestBody("application/json".toMediaType())
        val refreshRequest = Request.Builder()
            .url(refreshUrl)
            .post(body)
            .addHeader("Authorization", "Bearer $currentToken")
            .addHeader("Content-Type", "application/json")
            .build()

        return try {
            val client = OkHttpClient.Builder()
                .connectTimeout(10, TimeUnit.SECONDS)
                .readTimeout(10, TimeUnit.SECONDS)
                .build()
            val refreshResponse = client.newCall(refreshRequest).execute()
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
        val contentType = response.header("Content-Type") ?: return response
        if (!contentType.contains("application/problem+json")) return response
        val body = response.peekBody(8192).string()
        val problem = try {
            json.decodeFromString<ProblemDetail>(body)
        } catch (_: Exception) {
            return response
        }
        throw ProblemDetailException(problem)
    }
}
