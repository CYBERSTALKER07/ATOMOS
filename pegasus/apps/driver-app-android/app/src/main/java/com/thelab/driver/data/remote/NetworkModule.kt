package com.thelab.driver.data.remote

import android.content.Context
import androidx.room.Room
import com.jakewharton.retrofit2.converter.kotlinx.serialization.asConverterFactory
import com.pegasus.driver.BuildConfig
import com.thelab.driver.data.local.LabDriverDatabase
import com.thelab.driver.data.local.OrderDao
import com.thelab.driver.data.local.PendingMutationDao
import com.thelab.driver.data.local.RouteManifestDao
import com.thelab.driver.data.model.ProblemDetail
import com.thelab.driver.data.model.ProblemDetailException
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import okhttp3.Authenticator
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
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
    fun provideOkHttp(json: Json): OkHttpClient = OkHttpClient.Builder()
        .connectTimeout(15, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .writeTimeout(15, TimeUnit.SECONDS)
        .pingInterval(30, TimeUnit.SECONDS) // WebSocket keepalive
        .addInterceptor { chain ->
            val token = TokenHolder.firebaseIdToken ?: TokenHolder.token
            val request = chain.request().newBuilder()
                .addHeader("X-Trace-Id", java.util.UUID.randomUUID().toString())
                .apply { if (token != null) addHeader("Authorization", "Bearer $token") }
                .build()
            chain.proceed(request)
        }
        .authenticator(TokenRefreshAuthenticator(json))
        .addInterceptor(ProblemDetailInterceptor(json))
        .addInterceptor(
            HttpLoggingInterceptor().apply {
                level = if (BuildConfig.DEBUG) {
                    HttpLoggingInterceptor.Level.BODY
                } else {
                    HttpLoggingInterceptor.Level.NONE
                }
            }
        )
        .build()

    @Provides
    @Singleton
    fun provideRetrofit(client: OkHttpClient, json: Json): Retrofit = Retrofit.Builder()
        .baseUrl(BuildConfig.API_BASE_URL + "/")
        .client(client)
        .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
        .build()

    @Provides
    @Singleton
    fun provideDriverApi(retrofit: Retrofit): DriverApi =
        retrofit.create(DriverApi::class.java)

    @Provides
    @Singleton
    fun provideDatabase(@ApplicationContext context: Context): LabDriverDatabase =
        Room.databaseBuilder(context, LabDriverDatabase::class.java, "lab_driver.db")
            .fallbackToDestructiveMigration()
            .build()

    @Provides
    fun provideOrderDao(db: LabDriverDatabase): OrderDao = db.orderDao()

    @Provides
    fun provideRouteManifestDao(db: LabDriverDatabase): RouteManifestDao = db.routeManifestDao()

    @Provides
    fun providePendingMutationDao(db: LabDriverDatabase): PendingMutationDao = db.pendingMutationDao()

    @Provides
    @Singleton
    fun provideTelemetrySocket(client: OkHttpClient, json: Json): TelemetrySocket =
        TelemetrySocket(client, json)

    @Provides
    @Singleton
    fun provideDriverWebSocket(client: OkHttpClient, json: Json): DriverWebSocket =
        DriverWebSocket(client, json)
}

/** Secure token holder backed by EncryptedSharedPreferences. Call init(context) in Application.onCreate(). */
object TokenHolder {
    private lateinit var prefs: android.content.SharedPreferences

    fun init(context: android.content.Context) {
        val masterKey = androidx.security.crypto.MasterKey.Builder(context)
            .setKeyScheme(androidx.security.crypto.MasterKey.KeyScheme.AES256_GCM)
            .build()
        prefs = androidx.security.crypto.EncryptedSharedPreferences.create(
            context,
            "lab_driver_auth",
            masterKey,
            androidx.security.crypto.EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            androidx.security.crypto.EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
        )
    }

    var token: String?
        get() = prefs.getString("token", null)
        set(value) = prefs.edit().putString("token", value).apply()

    /** Firebase ID token — preferred over legacy JWT when non-null */
    var firebaseIdToken: String?
        get() = prefs.getString("firebaseIdToken", null)
        set(value) = prefs.edit().putString("firebaseIdToken", value).apply()

    var userId: String?
        get() = prefs.getString("userId", null)
        set(value) = prefs.edit().putString("userId", value).apply()

    var driverName: String?
        get() = prefs.getString("driverName", null)
        set(value) = prefs.edit().putString("driverName", value).apply()

    var vehicleType: String?
        get() = prefs.getString("vehicleType", null)
        set(value) = prefs.edit().putString("vehicleType", value).apply()

    var licensePlate: String?
        get() = prefs.getString("licensePlate", null)
        set(value) = prefs.edit().putString("licensePlate", value).apply()

    var vehicleId: String?
        get() = prefs.getString("vehicleId", null)
        set(value) = prefs.edit().putString("vehicleId", value).apply()

    var vehicleClass: String?
        get() = prefs.getString("vehicleClass", null)
        set(value) = prefs.edit().putString("vehicleClass", value).apply()

    var maxVolumeVU: Double
        get() = prefs.getString("maxVolumeVU", "0.0")?.toDoubleOrNull() ?: 0.0
        set(value) = prefs.edit().putString("maxVolumeVU", value.toString()).apply()

    var warehouseId: String?
        get() = prefs.getString("warehouseId", null)
        set(value) = prefs.edit().putString("warehouseId", value).apply()

    var warehouseName: String?
        get() = prefs.getString("warehouseName", null)
        set(value) = prefs.edit().putString("warehouseName", value).apply()

    var warehouseLat: Double
        get() = prefs.getString("warehouseLat", "0.0")?.toDoubleOrNull() ?: 0.0
        set(value) = prefs.edit().putString("warehouseLat", value.toString()).apply()

    var warehouseLng: Double
        get() = prefs.getString("warehouseLng", "0.0")?.toDoubleOrNull() ?: 0.0
        set(value) = prefs.edit().putString("warehouseLng", value.toString()).apply()

    fun clear() {
        prefs.edit().clear().apply()
    }
}

/** OkHttp Authenticator that attempts token refresh on 401 before clearing credentials. */
private class TokenRefreshAuthenticator(private val json: Json) : Authenticator {
    override fun authenticate(route: Route?, response: okhttp3.Response): Request? {
        // Prevent infinite retry loops — give up after a single refresh attempt
        if (response.request.header("X-Refresh-Attempted") != null) {
            TokenHolder.clear()
            return null
        }

        val currentToken = TokenHolder.token ?: run {
            TokenHolder.clear()
            return null
        }

        // Synchronously call the refresh endpoint
        val refreshRequest = Request.Builder()
            .url(BuildConfig.API_BASE_URL + "/v1/auth/refresh")
            .post("".toRequestBody("application/json".toMediaType()))
            .addHeader("Authorization", "Bearer $currentToken")
            .build()

        val client = OkHttpClient.Builder()
            .connectTimeout(10, TimeUnit.SECONDS)
            .readTimeout(10, TimeUnit.SECONDS)
            .build()

        val refreshResponse = try {
            client.newCall(refreshRequest).execute()
        } catch (_: Exception) {
            TokenHolder.clear()
            return null
        }

        if (refreshResponse.code != 200) {
            refreshResponse.close()
            TokenHolder.clear()
            return null
        }

        val body = refreshResponse.body?.string() ?: run {
            TokenHolder.clear()
            return null
        }

        val newToken = try {
            json.parseToJsonElement(body).jsonObject["token"]?.jsonPrimitive?.content
        } catch (_: Exception) {
            null
        }

        if (newToken == null) {
            TokenHolder.clear()
            return null
        }

        // Persist the refreshed token
        TokenHolder.token = newToken

        // Retry the original request with the new token
        return response.request.newBuilder()
            .header("Authorization", "Bearer $newToken")
            .header("X-Refresh-Attempted", "true")
            .build()
    }
}

// ── RFC 7807 Problem Detail interceptor ──

private class ProblemDetailInterceptor(private val json: Json) : okhttp3.Interceptor {
    override fun intercept(chain: okhttp3.Interceptor.Chain): okhttp3.Response {
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
