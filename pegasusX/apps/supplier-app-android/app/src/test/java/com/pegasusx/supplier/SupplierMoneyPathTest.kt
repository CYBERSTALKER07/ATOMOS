package com.pegasusx.supplier

import com.pegasusx.supplier.data.model.PromoSimulateInput
import com.pegasusx.supplier.ui.components.formatMinorAmount
import com.pegasusx.supplier.ui.design.MarketPack
import com.pegasusx.supplier.ui.design.formatMinorDecimal
import com.pegasusx.supplier.ui.design.formatPackMoney
import com.pegasusx.supplier.ui.design.formatTiyinFixed
import com.pegasusx.supplier.ui.design.parseMajorToMinor
import java.util.Locale
import kotlinx.serialization.ExperimentalSerializationApi
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonUnquotedLiteral
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class SupplierMoneyPathTest {

    @Test
    fun formatMinorAmount_formatsWholeAndFractionalTiyinWithoutFloatMath() {
        assertEquals("125 UZS", formatMinorAmount(12_500L, "UZS", Locale.US))
        assertEquals("125.5 UZS", formatMinorAmount(12_550L, "UZS", Locale.US))
        assertEquals("125.05 UZS", formatMinorAmount(12_505L, "UZS", Locale.US))
        assertEquals("-0.5 UZS", formatMinorAmount(-50L, "UZS", Locale.US))
        assertEquals("50,000,000 UZS", formatMinorAmount(5_000_000_000L, "UZS", Locale.US))
        assertEquals("125.75", formatMinorAmount(12_575L, "", Locale.US))
    }

    @Test
    fun formatTiyinFixed_and_parseMajorToMinor_roundTripDeterministically() {
        assertEquals("19.99", formatTiyinFixed(1999L))
        assertEquals("-0.05", formatTiyinFixed(-5L))
        assertEquals(1999L, parseMajorToMinor("19.99"))
        assertEquals(1950L, parseMajorToMinor("19,5"))
        assertEquals(10_000L, parseMajorToMinor("100"))
        assertEquals(5_000_000_000L, parseMajorToMinor("50000000.00"))
        assertNull(parseMajorToMinor("-1"))
        assertNull(parseMajorToMinor("12.345"))
        assertNull(parseMajorToMinor("abc"))
    }

    @OptIn(ExperimentalSerializationApi::class)
    @Test
    fun promoSimulateInput_encodesDiscountPctWithoutDouble() {
        val discountBps = 1250L
        val literal = JsonUnquotedLiteral(
            formatMinorDecimal(minor = discountBps, decimalPlaces = 2, groupSeparator = null),
        )
        val input = PromoSimulateInput(
            promotionId = "promo-1",
            discountPct = literal,
            expectedUnits = 500L,
            avgUnitMarginMinor = 1000L,
        )
        val encoded = Json.encodeToString(input)
        assertTrue(encoded.contains("\"discount_pct\":12.5"))
    }

    @Test
    fun formatPackMoney_respectsPackCurrencyAndAvoidsFloatDivision() {
        val pack = MarketPack(
            code = "UZ",
            name = "Uzbekistan",
            timezone = "Asia/Tashkent",
            currencyCode = "UZS",
            fiscalAdapter = "MY_SOLIQ",
            mapsAdapter = "MAPLIBRE",
            checkoutReadsThis = true,
        )
        assertEquals("1 234.56 UZS", formatPackMoney(123_456L, pack))
        assertEquals("1 234.56", formatPackMoney(123_456L, null))
    }
}
