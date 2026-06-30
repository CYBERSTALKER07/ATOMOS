package com.pegasusx.supplier

import com.pegasusx.supplier.data.remote.SupplierRealtimeSignals
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runCurrent
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertTrue
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class RealtimeHubTests {

    @Test
    fun testRefreshTickEmission() = runTest {
        val signals = SupplierRealtimeSignals()
        var tickReceived = false

        val job = launch {
            signals.refreshTick.first() // Wait for first emission
            tickReceived = true
        }

        runCurrent() // Allow the subscriber to register
        signals.bump()
        advanceUntilIdle()

        assertTrue("Expected refreshTick to be emitted upon bump()", tickReceived)
        job.cancel()
    }

    @Test
    fun testReconnectTickEmission() = runTest {
        val signals = SupplierRealtimeSignals()
        var reconnectReceived = false
        var refreshReceived = false

        val job1 = launch {
            signals.reconnectTick.first()
            reconnectReceived = true
        }
        
        val job2 = launch {
            signals.refreshTick.first()
            refreshReceived = true
        }

        runCurrent() // Allow the subscribers to register
        signals.bumpReconnect()
        advanceUntilIdle()

        assertTrue("Expected reconnectTick to be emitted upon bumpReconnect()", reconnectReceived)
        assertTrue("Expected refreshTick to be emitted upon bumpReconnect()", refreshReceived)
        
        job1.cancel()
        job2.cancel()
    }
}
