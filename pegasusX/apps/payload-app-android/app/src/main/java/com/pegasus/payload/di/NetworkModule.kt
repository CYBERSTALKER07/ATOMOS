package com.pegasus.payload.di

import com.pegasus.payload.ui.design.CellPinInterceptor
import com.pegasus.payload.BuildConfig
import com.pegasus.payload.data.local.SecureStore
import com.pegasus.payload.data.remote.AuthInterceptor
import com.pegasus.payload.data.remote.TokenRefreshAuthenticator
import com.pegasus.payload.data.remote.PayloadApi
import com.jakewharton.retrofit2.converter.kotlinx.serialization.asConverterFactory
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent
import kotlinx.serialization.json.Json
import okhttp3.CertificatePinner
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import java.util.concurrent.TimeUnit
import javax.inject.Singleton

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

@Module
@InstallIn(SingletonComponent::class)
object NetworkModule {

    @Provides @Singleton
    fun provideJson(): Json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
        coerceInputValues = true
    }

    @Provides @Singleton
    fun provideOkHttp(
        authInterceptor: AuthInterceptor,
        tokenRefreshAuthenticator: TokenRefreshAuthenticator,
        secureStore: SecureStore,
    ): OkHttpClient {
        val logging = HttpLoggingInterceptor().apply {
            level = if (BuildConfig.DEBUG) HttpLoggingInterceptor.Level.BASIC
            else HttpLoggingInterceptor.Level.NONE
        }
        return OkHttpClient.Builder()
            .apply {
                configuredCertificatePinner(BuildConfig.API_BASE_URL, BuildConfig.API_TLS_SPKI_PINS)
                    ?.let { certificatePinner(it) }
            }
            .addInterceptor(CellPinInterceptor(BuildConfig.API_BASE_URL) { secureStore.token })
            .addInterceptor(authInterceptor)
            .authenticator(tokenRefreshAuthenticator)
            .addInterceptor(com.pegasus.payload.ui.design.PegasusHttpResilienceInterceptor())
            .addInterceptor(logging)
            .connectTimeout(15, TimeUnit.SECONDS)
            .readTimeout(30, TimeUnit.SECONDS)
            .writeTimeout(30, TimeUnit.SECONDS)
            .pingInterval(20, TimeUnit.SECONDS) // keep WS connections alive
            .build()
    }

    @Provides @Singleton
    fun provideRetrofit(json: Json, client: OkHttpClient): Retrofit =
        Retrofit.Builder()
            .baseUrl(BuildConfig.API_BASE_URL.trimEnd('/') + "/")
            .client(client)
            .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
            .build()

    @Provides @Singleton
    fun providePayloadApi(retrofit: Retrofit): PayloadApi =
        retrofit.create(PayloadApi::class.java)
}
