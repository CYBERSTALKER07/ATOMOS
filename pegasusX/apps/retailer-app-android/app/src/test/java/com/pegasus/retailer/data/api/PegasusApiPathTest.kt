package com.pegasusx.retailer.data.api

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test
import retrofit2.http.GET

class PegasusApiPathTest {
    @Test
    fun retailerAiPredictionsUsesCanonicalPath() {
        val method = PegasusApi::class.java.methods.first { it.name == "getRetailerAIPredictions" }
        val get = method.getAnnotation(GET::class.java)
        assertEquals("/v1/retailer/ai/predictions", get.value)
    }

    @Test
    fun aliasPredictionsPathIsNotOnApi() {
        val alias = PegasusApi::class.java.methods.firstOrNull { method ->
            method.getAnnotation(GET::class.java)?.value == "/v1/ai/predictions"
        }
        assertNull(alias)
    }
}
