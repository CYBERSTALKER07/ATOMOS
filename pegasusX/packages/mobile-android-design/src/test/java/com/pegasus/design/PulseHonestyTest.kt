package com.pegasus.design

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class PulseHonestyTest {
    @Test
    fun failedKeepsPreviousEvents() {
        val previous = listOf("keep")
        val result = PulseHonesty.applyHttp(ok = false, incoming = null, previous = previous)
        assertEquals(previous, result.events)
        assertEquals(PulseHonesty.FAILED, result.error)
    }

    @Test
    fun successMayBeHonestEmpty() {
        val result = PulseHonesty.applyHttp(ok = true, incoming = emptyList(), previous = listOf("old"))
        assertEquals(emptyList<String>(), result.events)
        assertNull(result.error)
    }

    @Test
    fun unsuccessfulBodyDoesNotReplaceTimeline() {
        val previous = listOf("old")
        val result = PulseHonesty.applyHttp(ok = false, incoming = listOf("new"), previous = previous)
        assertEquals(previous, result.events)
        assertEquals(PulseHonesty.FAILED, result.error)
    }

    @Test
    fun commandFailureKeepsPreviousPulse() {
        val previous = "last"
        val result = PulseHonesty.applyObject(ok = false, incoming = null, previous = previous)
        assertEquals(previous, result.value)
        assertEquals(PulseHonesty.COMMAND_FAILED, result.error)
    }

    @Test
    fun commandSuccessMayBeHonestEmpty() {
        val result = PulseHonesty.applyObject(ok = true, incoming = "empty", previous = "old")
        assertEquals("empty", result.value)
        assertNull(result.error)
    }
}
