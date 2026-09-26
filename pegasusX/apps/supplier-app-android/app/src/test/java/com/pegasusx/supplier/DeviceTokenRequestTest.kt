package com.pegasusx.supplier

import com.pegasusx.supplier.data.model.DeviceTokenRequest
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class DeviceTokenRequestTest {
    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun encodesTokenAndPlatformOnly() {
        val encoded = json.encodeToString(
            DeviceTokenRequest.serializer(),
            DeviceTokenRequest(token = "fcm-reg-token", platform = "android"),
        )
        val map = json.decodeFromString<Map<String, String>>(encoded)
        assertEquals("fcm-reg-token", map["token"])
        assertEquals("android", map["platform"])
        assertTrue(!encoded.contains("supplier_id"))
    }
}
