package com.pegasus.payload.data.remote

import com.pegasus.payload.data.local.SecureStore
import okhttp3.Interceptor
import okhttp3.Response
import java.util.UUID
import javax.inject.Inject

/**
 * Adds Authorization (when a token is present) and a per-request X-Trace-Id
 * header so every backend log line and emitted Kafka event can be stitched
 * back to the originating action.
 */
class AuthInterceptor @Inject constructor(
    private val secureStore: SecureStore,
) : Interceptor {
    override fun intercept(chain: Interceptor.Chain): Response {
        val builder = chain.request().newBuilder()
            .header("X-Trace-Id", UUID.randomUUID().toString())
            .header("Accept", "application/json")
        secureStore.token?.let { builder.header("Authorization", "Bearer $it") }
        return chain.proceed(builder.build())
    }
}
