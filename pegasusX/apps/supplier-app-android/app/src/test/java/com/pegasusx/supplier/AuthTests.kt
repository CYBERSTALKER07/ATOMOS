package com.pegasusx.supplier

import com.pegasusx.supplier.data.model.LoginRequest
import com.pegasusx.supplier.data.model.LoginResponse
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test

class AuthTests {
    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun testLoginResponseSerialization() {
        val rawJson = """
            {
                "supplier_id": "sup_123",
                "is_configured": true,
                "token": "jwt_token_123",
                "refresh_token": "refresh_123"
            }
        """.trimIndent()

        val response = json.decodeFromString<LoginResponse>(rawJson)
        assertEquals("sup_123", response.supplierId)
        assertTrue(response.isConfigured)
        assertEquals("jwt_token_123", response.token)
        assertEquals("refresh_123", response.refreshToken)
    }

    @Test
    fun testLoginRequestSerialization() {
        val request = LoginRequest(phone = "+998901234567", password = "password123")
        val jsonString = json.encodeToString(LoginRequest.serializer(), request)
        
        assertTrue(jsonString.contains("+998901234567"))
        assertTrue(jsonString.contains("password123"))
    }
}
