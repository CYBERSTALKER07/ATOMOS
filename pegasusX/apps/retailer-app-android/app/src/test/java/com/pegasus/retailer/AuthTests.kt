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
        assertEquals("usr_123", response.user!!.id)
    }

    @Test
    fun testPendingOrgSelectSerialization() {
        val rawJson = """
            {
                "token": "pending_jwt",
                "token_type": "pending_org_select",
                "memberships": [
                  {"user_id":"u1","retailer_id":"org-a","retailer_role":"OWNER","name":"Shop A","is_active":true},
                  {"user_id":"u2","retailer_id":"org-b","retailer_role":"MANAGER","name":"Shop B","is_active":true}
                ],
                "expires_in_sec": 420
            }
        """.trimIndent()
        val response = json.decodeFromString<AuthResponse>(rawJson)
        assertTrue(response.isPendingOrgSelect)
        assertEquals(2, response.memberships.size)
        assertEquals("org-a", response.memberships[0].retailerId)
    }

    @Test
    fun testLoginRequestSerialization() {
        val request = LoginRequest(phoneNumber = "+998901234567", password = "password123")
        val jsonString = json.encodeToString(LoginRequest.serializer(), request)
        
        assertTrue(jsonString.contains("+998901234567"))
        assertTrue(jsonString.contains("password123"))
    }
}
