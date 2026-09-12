package com.pegasus.retailer.data.local

import com.pegasusx.retailer.data.local.TokenManager
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class TokenAuthTest {
    @Test
    fun httpAuthorizationToken_usesSessionJwtNotFirebaseId() {
        assertEquals(
            "session-jwt",
            TokenManager.httpAuthorizationToken("session-jwt", "firebase-id-token"),
        )
        assertEquals("session-jwt", TokenManager.httpAuthorizationToken("session-jwt", null))
        assertNull(TokenManager.httpAuthorizationToken(null, "firebase-id-token"))
        assertNull(TokenManager.httpAuthorizationToken("  ", "firebase-id-token"))
        assertNull(TokenManager.httpAuthorizationToken(null, null))
    }
}
