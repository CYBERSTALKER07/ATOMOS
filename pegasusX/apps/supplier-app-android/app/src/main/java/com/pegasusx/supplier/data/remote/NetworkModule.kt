package com.pegasusx.supplier.data.remote

import android.content.Context
import android.content.SharedPreferences
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import com.pegasus.design.network.CellPinInterceptor
import com.pegasusx.supplier.BuildConfig
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import kotlinx.serialization.json.Json
import okhttp3.Interceptor
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import retrofit2.Retrofit
import retrofit2.converter.kotlinx.serialization.asConverterFactory
import java.util.concurrent.TimeUnit
import javax.inject.Singleton

object TokenHolder {
    private const val PREF_NAME = "supplier_secure_prefs"
    private const val KEY_TOKEN = "pegasus_supplier_jwt"
    private const val KEY_REFRESH_TOKEN = "refresh_token"
    private const val KEY_SUPPLIER_ID = "supplier_id"
    private const val KEY_CONFIGURED = "is_configured"

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
        get() = prefs.getString(KEY_REFRESH_TOKEN, null)
        set(value) = prefs.edit().putString(KEY_REFRESH_TOKEN, value).apply()

    var supplierId: String?
        get() = prefs.getString(KEY_SUPPLIER_ID, null)
        set(value) = prefs.edit().putString(KEY_SUPPLIER_ID, value).apply()

    var isConfigured: Boolean
        get() = prefs.getBoolean(KEY_CONFIGURED, false)
        set(value) = prefs.edit().putBoolean(KEY_CONFIGURED, value).apply()

    fun clear() {
        prefs.edit().clear().apply()
    }

    val isLoggedIn: Boolean get() = !token.isNullOrBlank()
}

private class AuthInterceptor : Interceptor {
    override fun intercept(chain: Interceptor.Chain): okhttp3.Response {
        val original = chain.request()
        val traced = original.newBuilder()
            .header("X-Trace-Id", java.util.UUID.randomUUID().toString())
            .build()
        val token = TokenHolder.token ?: return chain.proceed(traced)
        return chain.proceed(
            traced.newBuilder()
                .header("Authorization", "Bearer $token")
                .build(),
        )
    }
}

private class TokenRefreshAuthenticator(
    private val json: Json,
    private val baseUrl: String,
) : okhttp3.Authenticator {
    private val refreshClient = OkHttpClient.Builder()
        .connectTimeout(10, TimeUnit.SECONDS)
        .readTimeout(10, TimeUnit.SECONDS)
        .build()

    @Synchronized
    override fun authenticate(route: okhttp3.Route?, response: okhttp3.Response): okhttp3.Request? {
        if (response.code != 401) return null
        if (response.request.header("X-Refresh-Attempted") != null) {
            TokenHolder.clear()
            return null
        }
        val currentToken = TokenHolder.token
        val requestToken = response.request.header("Authorization")?.removePrefix("Bearer ")?.trim()
        if (!currentToken.isNullOrBlank() && !requestToken.isNullOrBlank() && currentToken != requestToken) {
            return response.request.newBuilder()
                .header("Authorization", "Bearer $currentToken")
                .header("X-Refresh-Attempted", "true")
                .build()
        }
        val refresh = TokenHolder.refreshToken ?: return null
        val body = json.encodeToString(
            kotlinx.serialization.serializer(),
            mapOf("refresh_token" to refresh),
        )
        val mediaType = "application/json; charset=utf-8".toMediaType()
        val refreshRequest = okhttp3.Request.Builder()
            .url("${baseUrl}v1/auth/supplier/refresh")
            .post(okhttp3.RequestBody.Companion.create(mediaType, body))
            .build()

        return try {
            val refreshResponse = refreshClient.newCall(refreshRequest).execute()
            if (!refreshResponse.isSuccessful) {
                refreshResponse.close()
                TokenHolder.clear()
                return null
            }
            val responseBody = refreshResponse.body?.string() ?: return null
            val obj = json.parseToJsonElement(responseBody) as? kotlinx.serialization.json.JsonObject ?: return null
            val newToken = (obj["token"] as? kotlinx.serialization.json.JsonPrimitive)?.content?.takeIf { it.isNotBlank() } ?: return null
            val newRefresh = (obj["refresh_token"] as? kotlinx.serialization.json.JsonPrimitive)?.content
            TokenHolder.token = newToken
            if (!newRefresh.isNullOrBlank()) {
                TokenHolder.refreshToken = newRefresh
            }
            response.request.newBuilder()
                .header("Authorization", "Bearer $newToken")
                .header("X-Refresh-Attempted", "true")
                .build()
        } catch (_: Exception) {
            null
        }
    }
}

@Module
@InstallIn(SingletonComponent::class)
object NetworkModule {
    private val BASE_URL: String get() = BuildConfig.API_BASE_URL.trimEnd('/') + "/"

    @Provides
    @Singleton
    fun provideJson(): Json = Json {
        ignoreUnknownKeys = true
        coerceInputValues = true
        encodeDefaults = true
    }

    @Provides
    @Singleton
    fun provideOkHttpClient(json: Json): OkHttpClient =
        OkHttpClient.Builder()
            .addInterceptor(com.pegasus.design.network.CellPinInterceptor(BuildConfig.API_BASE_URL) { TokenHolder.token })
            .addInterceptor(AuthInterceptor())
            .addInterceptor(com.pegasus.design.network.PegasusHttpResilienceInterceptor())
            .authenticator(TokenRefreshAuthenticator(json, BASE_URL))
            .connectTimeout(30, TimeUnit.SECONDS)
            .readTimeout(30, TimeUnit.SECONDS)
            .writeTimeout(30, TimeUnit.SECONDS)
            .pingInterval(25, TimeUnit.SECONDS)
            .build()

    @Provides
    @Singleton
    fun provideRetrofit(client: OkHttpClient, json: Json): Retrofit =
        Retrofit.Builder()
            .baseUrl(BASE_URL)
            .client(client)
            .addConverterFactory(json.asConverterFactory("application/json; charset=utf-8".toMediaType()))
            .build()

    @Provides
    @Singleton
    fun provideSupplierApi(retrofit: Retrofit): SupplierApi =
        retrofit.create(SupplierApi::class.java)

    @Provides
    @Singleton
    fun provideGeocodeApi(retrofit: Retrofit): GeocodeApi =
        retrofit.create(GeocodeApi::class.java)

    @Provides
    @Singleton
    fun provideSupplierOperationsRepository(api: SupplierApi): SupplierOperationsRepository =
        SupplierOperationsRepository(api)
}
