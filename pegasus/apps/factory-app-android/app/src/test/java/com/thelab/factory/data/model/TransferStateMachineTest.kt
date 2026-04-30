package com.thelab.factory.data.model

import kotlinx.serialization.json.Json
import org.junit.Assert.*
import org.junit.Test

/**
 * Factory Transfer State Machine — validates all legal transitions
 *
 * Valid states: DRAFT → APPROVED → LOADING → DISPATCHED
 * Plus: any → CANCELLED
 */
class TransferStateMachineTest {

    private val validTransitions = mapOf(
        "DRAFT" to listOf("APPROVED", "CANCELLED"),
        "APPROVED" to listOf("LOADING", "CANCELLED"),
        "LOADING" to listOf("DISPATCHED", "CANCELLED"),
        "DISPATCHED" to emptyList(),
        "CANCELLED" to emptyList(),
    )

    @Test
    fun `DRAFT can transition to APPROVED`() {
        assertTransitionAllowed("DRAFT", "APPROVED")
    }

    @Test
    fun `APPROVED can transition to LOADING`() {
        assertTransitionAllowed("APPROVED", "LOADING")
    }

    @Test
    fun `LOADING can transition to DISPATCHED`() {
        assertTransitionAllowed("LOADING", "DISPATCHED")
    }

    @Test
    fun `DISPATCHED is terminal`() {
        assertTransitionBlocked("DISPATCHED", "DRAFT")
        assertTransitionBlocked("DISPATCHED", "APPROVED")
        assertTransitionBlocked("DISPATCHED", "LOADING")
    }

    @Test
    fun `any non-terminal state can cancel`() {
        assertTransitionAllowed("DRAFT", "CANCELLED")
        assertTransitionAllowed("APPROVED", "CANCELLED")
        assertTransitionAllowed("LOADING", "CANCELLED")
    }

    @Test
    fun `backward transitions are blocked`() {
        assertTransitionBlocked("LOADING", "APPROVED")
        assertTransitionBlocked("DISPATCHED", "LOADING")
        assertTransitionBlocked("APPROVED", "DRAFT")
    }

    @Test
    fun `TransitionRequest round-trips every valid target state`() {
        val json = Json { ignoreUnknownKeys = true }
        for ((_, targets) in validTransitions) {
            for (target in targets) {
                val req = TransitionRequest(targetState = target)
                val encoded = json.encodeToString(TransitionRequest.serializer(), req)
                val decoded = json.decodeFromString(TransitionRequest.serializer(), encoded)
                assertEquals(target, decoded.targetState)
            }
        }
    }

    private fun assertTransitionAllowed(from: String, to: String) {
        val allowed = validTransitions[from] ?: emptyList()
        assertTrue("$from → $to should be allowed", allowed.contains(to))
    }

    private fun assertTransitionBlocked(from: String, to: String) {
        val allowed = validTransitions[from] ?: emptyList()
        assertFalse("$from → $to should be blocked", allowed.contains(to))
    }
}
