package com.pegasusx.supplier

import com.pegasusx.supplier.data.model.ForecastConfidence
import com.pegasusx.supplier.util.brainForecastLine
import com.pegasusx.supplier.util.factoryPlanningDisabledCode
import com.pegasusx.supplier.util.isForecastBlocked
import com.pegasusx.supplier.util.planBrainTabFromQuery
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class PlanningTests {
    @Test
    fun testPlanBrainTabAndBlockedForecast() {
        assertEquals("brain", planBrainTabFromQuery("brain"))
        assertEquals("planning", planBrainTabFromQuery(null))
        val blocked = ForecastConfidence(
            blockedReason = "sparsity_blocked",
            label = "insufficient_history",
        )
        assertTrue(isForecastBlocked(blocked))
        assertNull(brainForecastLine(blocked, listOf(1.0, 2.0, 3.0)))
        assertEquals(
            "factory_planning_disabled",
            factoryPlanningDisabledCode(409, """{"error":"factory_planning_disabled"}"""),
        )
    }
}
