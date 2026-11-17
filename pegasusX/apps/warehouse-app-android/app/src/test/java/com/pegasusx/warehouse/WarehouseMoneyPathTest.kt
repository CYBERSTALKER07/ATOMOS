package com.pegasusx.warehouse

import com.pegasusx.warehouse.data.model.AnalyticsData
import com.pegasusx.warehouse.ui.design.MarketPack
import com.pegasusx.warehouse.ui.design.formatMinorDecimal
import com.pegasusx.warehouse.ui.design.formatPackMoney
import com.pegasusx.warehouse.ui.design.formatTiyinFixed
import com.pegasusx.warehouse.ui.design.parseMajorToMinor
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class WarehouseMoneyPathTest {

    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun analyticsData_avgOrderValue_decodesIntegerDecimalAnd64BitLong() {
        val fromInt = json.decodeFromString<AnalyticsData>(
            """{"period":"30d","total_revenue":900000,"total_orders":2,"avg_order_value":450000}""",
        )
        assertEquals(450_000L, fromInt.avgOrderValue)

        val fromDecimal = json.decodeFromString<AnalyticsData>(
            """{"period":"30d","total_revenue":900001,"total_orders":2,"avg_order_value":450000.6}""",
        )
        assertEquals(450_001L, fromDecimal.avgOrderValue)

        val fromLargeLong = json.decodeFromString<AnalyticsData>(
            """{"period":"30d","total_revenue":10000000000,"total_orders":2,"avg_order_value":5000000000}""",
        )
        assertEquals(5_000_000_000L, fromLargeLong.avgOrderValue)
    }

    @Test
    fun marketPack_moneyHelpers_formatAndParseWithoutFloatMath() {
        val pack = MarketPack(
            code = "UZ",
            name = "Uzbekistan",
            timezone = "Asia/Tashkent",
            currencyCode = "UZS",
            fiscalAdapter = "MY_SOLIQ",
            mapsAdapter = "MAPLIBRE",
            checkoutReadsThis = true,
        )
        assertEquals("50 000 000 UZS", formatPackMoney(5_000_000_000L, pack))
        assertEquals("-0.5", formatMinorDecimal(-50L, 2, null))
        assertEquals("-0.50", formatTiyinFixed(-50L))
        assertEquals(250_075L, parseMajorToMinor("2500.75"))
        assertNull(parseMajorToMinor("-5.00"))
    }
}
