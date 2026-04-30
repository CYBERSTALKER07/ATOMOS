package com.pegasus.driver.ui.screens.offload

import org.junit.Assert.*
import org.junit.Test

/**
 * CashCollectionUiState — State transitions, validation
 */
class CashCollectionUiStateTest {

    @Test
    fun `default state has empty orderId and zero amount`() {
        val state = CashCollectionUiState()
        assertEquals("", state.orderId)
        assertEquals(0L, state.amount)
        assertFalse(state.isCompleting)
        assertFalse(state.completed)
        assertNull(state.error)
        assertNull(state.distanceM)
        assertTrue(state.locationAvailable)
    }

    @Test
    fun `state with orderId and amount preserves values`() {
        val state = CashCollectionUiState(orderId = "o-123", amount = 150_000L)
        assertEquals("o-123", state.orderId)
        assertEquals(150_000L, state.amount)
    }

    @Test
    fun `isCompleting flag set during collection`() {
        val state = CashCollectionUiState(isCompleting = true)
        assertTrue(state.isCompleting)
        assertFalse(state.completed)
    }

    @Test
    fun `completed state after successful collection`() {
        val state = CashCollectionUiState(
            orderId = "o-123",
            amount = 150_000L,
            completed = true,
            distanceM = 45.0,
        )
        assertTrue(state.completed)
        assertFalse(state.isCompleting)
        assertEquals(45.0, state.distanceM!!, 0.001)
    }

    @Test
    fun `error state with location unavailable`() {
        val state = CashCollectionUiState(
            locationAvailable = false,
            error = "Unable to get GPS location",
        )
        assertFalse(state.locationAvailable)
        assertNotNull(state.error)
        assertTrue(state.error!!.contains("GPS"))
    }

    @Test
    fun `error clears on retry`() {
        val initial = CashCollectionUiState(error = "Network error")
        val retrying = initial.copy(isCompleting = true, error = null)
        assertNull(retrying.error)
        assertTrue(retrying.isCompleting)
    }
}
