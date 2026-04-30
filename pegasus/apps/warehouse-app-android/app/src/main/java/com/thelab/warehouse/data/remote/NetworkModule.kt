package com.thelab.warehouse.data.remote

import android.content.Context
import android.content.SharedPreferences
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import com.pegasus.warehouse.BuildConfig
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import kotlinx.serialization.json.Json
import okhttp3.Interceptor
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import retrofit2.Retrofit
import retrofit2.converter.kotlinx.serialization.asConverterFactory
import java.util.concurrent.TimeUnit
import javax.inject.Singleton

object TokenHolder {
    private const val PREF_NAME = "warehouse_secure_prefs"
    private const val KEY_TOKEN = "warehouse_jwt"
    private const val KEY_REFRESH = "warehouse_refresh_token"
    private const val KEY_WAREHOUSE_ID = "warehouse_id"

    private lateinit var prefs: SharedPreferences

    fun init(context: Context) {
        val masterKey = MasterKey.Builder(context)
            .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
            .build()
        prefs = EncryptedSharedPreferences.create(
            context,
            PREF_NAME,
            masterKey,
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
        )
    }

    var token: String?
        get() = prefs.getString(KEY_TOKEN, null)
        set(value) = prefs.edit().putString(KEY_TOKEN, value).apply()

    var refreshToken: String?
        get() = prefs.getString(KEY_REFRESH, null)
        set(value) = prefs.edit().putString(KEY_REFRESH, value).apply()

    var warehouseId: String?
        get() = prefs.getString(KEY_WAREHOUSE_ID, null)
        set(value) = prefs.edit().putString(KEY_WAREHOUSE_ID, value).apply()

    fun clear() {
        prefs.edit().clear().apply()
    }

    val isLoggedIn: Boolean get() = !token.isNullOrBlank()
}

private class AuthInterceptor : Interceptor {
    override fun intercept(chain: Interceptor.Chain): Response {
        val original = chain.request()
        val token = TokenHolder.token ?: return chain.proceed(
            original.newBuilder().header("X-Trace-Id", java.util.UUID.randomUUID().toString()).build()
        )
        val request = original.newBuilder()
            .header("Authorization", "Bearer $token")
            .header("X-Trace-Id", java.util.UUID.randomUUID().toString())
            .build()
        return chain.proceed(request)
    }
}

private class TokenRefreshAuthenticator(
    private val json: Json,
    private val baseUrl: String,
) : okhttp3.Authenticator {
    override fun authenticate(route: okhttp3.Route?, response: Response): Request? {
        if (response.code != 401) return null
        val refresh = TokenHolder.refreshToken ?: return null

        val refreshClient = OkHttpClient.Builder()
            .connectTimeout(10, TimeUnit.SECONDS)
            .build()

        val body = json.encodeToString(
            kotlinx.serialization.serializer(),
            mapOf("refresh_token" to refresh)
        )
        val mediaType = "application/json; charset=utf-8".toMediaType()
        val refreshRequest = Request.Builder()
            .url("${baseUrl}v1/auth/warehouse/refresh")
            .post(okhttp3.RequestBody.Companion.create(mediaType, body))
            .build()

        return try {
            val refreshResponse = refreshClient.newCall(refreshRequest).execute()
            if (!refreshResponse.isSuccessful) {
                TokenHolder.clear()
                return null
            }
            val responseBody = refreshResponse.body?.string() ?: return null
            val auth = json.decodeFromString<com.thelab.warehouse.data.model.AuthResponse>(responseBody)
            TokenHolder.token = auth.token
            TokenHolder.refreshToken = auth.refreshToken

            response.request.newBuilder()
                .header("Authorization", "Bearer ${auth.token}")
                .build()
        } catch (_: Exception) {
            TokenHolder.clear()
            null
        }
    }
}

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
    fun provideOkHttpClient(json: Json): OkHttpClient {
        val baseUrl = BuildConfig.API_BASE_URL.trimEnd('/') + "/"
        return OkHttpClient.Builder()
            .addInterceptor(AuthInterceptor())
            .authenticator(TokenRefreshAuthenticator(json, baseUrl))
            .connectTimeout(30, TimeUnit.SECONDS)
            .readTimeout(30, TimeUnit.SECONDS)
            .writeTimeout(30, TimeUnit.SECONDS)
            .build()
    }

    @Provides
    @Singleton
    fun provideRetrofit(client: OkHttpClient, json: Json): Retrofit {
        val baseUrl = BuildConfig.API_BASE_URL.trimEnd('/') + "/"
        return Retrofit.Builder()
            .baseUrl(baseUrl)
            .client(client)
            .addConverterFactory(json.asConverterFactory("application/json; charset=utf-8".toMediaType()))
            .build()
    }

    @Provides
    @Singleton
    fun provideWarehouseApi(retrofit: Retrofit): WarehouseApi =
        retrofit.create(WarehouseApi::class.java)
}
