package com.pegasus.payload

import com.pegasus.payload.di.configuredCertificatePinner
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Test

class SecurityPinningTest {
    @Test
    fun `skips pinning on loopback emulator and plaintext URLs`() {
        val samplePin = "sha256/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
        assertNull(configuredCertificatePinner("http://10.0.2.2:8180", samplePin))
        assertNull(configuredCertificatePinner("https://10.0.2.2:8443", samplePin))
        assertNull(configuredCertificatePinner("https://localhost:8443", samplePin))
        assertNull(configuredCertificatePinner("https://127.0.0.1:8443", samplePin))
        assertNull(configuredCertificatePinner("http://api.pegasusx.app", samplePin))
    }

    @Test
    fun `skips pinning when no valid pins are configured`() {
        assertNull(configuredCertificatePinner("https://api.pegasusx.app", ""))
        assertNull(configuredCertificatePinner("https://api.pegasusx.app", "invalid-pin"))
    }

    @Test
    fun `configures OkHttp SPKI pinning for production HTTPS host`() {
        val pin1 = "sha256/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
        val pin2 = "sha256/BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB="
        val pinner = configuredCertificatePinner("https://api.pegasusx.app/v1", "$pin1, $pin2")
        assertNotNull(pinner)
        assertEquals(2, pinner!!.pins.size)
    }
}
