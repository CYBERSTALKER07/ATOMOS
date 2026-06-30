package com.pegasus.retailer

import com.pegasusx.retailer.data.model.LoginRequest
import com.pegasusx.retailer.data.model.AuthResponse
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
                "token": "jwt_token_123",
                "firebase_token": "fb_token_123",
                "user": {
                    "id": "usr_123",
                    "name": "Test User",
                    "phone": "+998901234567"
                }
            }
        """.trimIndent()

        val response = json.decodeFromString<AuthResponse>(rawJson)
        assertEquals("jwt_token_123", response.token)
        assertEquals("fb_token_123", response.firebaseToken)
        assertNotNull(response.user)
        assertEquals("usr_123", response.user.id)
    }

    @Test
    fun testLoginRequestSerialization() {
        val request = LoginRequest(phoneNumber = "+998901234567", password = "password123")
        val jsonString = json.encodeToString(LoginRequest.serializer(), request)
        
        assertTrue(jsonString.contains("+998901234567"))
        assertTrue(jsonString.contains("password123"))
    }
}
