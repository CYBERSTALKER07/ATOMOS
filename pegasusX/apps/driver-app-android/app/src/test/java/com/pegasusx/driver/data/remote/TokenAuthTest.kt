package com.pegasusx.driver.data.remote

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class TokenAuthTest {
    @Test
    fun httpAuthorizationToken_usesSessionJwtNotFirebaseId() {
        assertEquals(
            "session-jwt",
            TokenHolder.httpAuthorizationToken("session-jwt", "firebase-id-token"),
        )
        assertEquals("session-jwt", TokenHolder.httpAuthorizationToken("session-jwt", null))
        assertNull(TokenHolder.httpAuthorizationToken(null, "firebase-id-token"))
        assertNull(TokenHolder.httpAuthorizationToken("  ", "firebase-id-token"))
        assertNull(TokenHolder.httpAuthorizationToken(null, null))
    }
}
