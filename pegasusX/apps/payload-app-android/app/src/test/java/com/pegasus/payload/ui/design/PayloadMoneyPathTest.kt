package com.pegasus.payload.ui.design

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class PayloadMoneyPathTest {

    @Test
    fun formatMinorDecimal_and_formatTiyinFixed_preserveSignAnd64BitPrecision() {
        assertEquals("0", formatMinorDecimal(0L, 2))
        assertEquals("-0.05", formatMinorDecimal(-5L, 2))
        assertEquals("50 000 000.25", formatMinorDecimal(5_000_000_025L, 2))
        assertEquals("-0.05", formatTiyinFixed(-5L))
        assertEquals("125.00", formatTiyinFixed(12_500L))
    }

    @Test
    fun parseMajorToMinor_parsesDeterministicMajorStrings() {
        assertEquals(1999L, parseMajorToMinor("19.99"))
        assertEquals(1950L, parseMajorToMinor("19,5"))
        assertEquals(5_000_000_000L, parseMajorToMinor("50000000"))
        assertNull(parseMajorToMinor(""))
        assertNull(parseMajorToMinor("-1"))
        assertNull(parseMajorToMinor("1.234"))
    }
}
