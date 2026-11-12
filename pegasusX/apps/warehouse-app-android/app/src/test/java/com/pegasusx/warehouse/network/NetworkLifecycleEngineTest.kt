package com.pegasusx.warehouse.network

import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class NetworkLifecycleEngineTest {

    @Test
    fun testOfflineStateTransitions() = runTest {
        val engine = NetworkLifecycleEngine.shared
        engine.resetForTesting()

        engine.setOnlineForTesting(false)
        assertFalse(engine.isOnline.value)
        assertEquals(NetworkState.OFFLINE, engine.state.value)
    }

    @Test
    fun testFlappingHysteresisEntersCooldown() = runTest {
        val engine = NetworkLifecycleEngine.shared
        engine.resetForTesting()

        engine.setOnlineForTesting(false)
        engine.setOnlineForTesting(true)
        engine.setOnlineForTesting(false)

        assertEquals(NetworkState.FLAPPING, engine.state.value)
        assertFalse(engine.isOnline.value)
    }

    @Test
    fun testReconnectCoordinatesEpochRefresh() = runTest {
        val engine = NetworkLifecycleEngine.shared
        engine.resetForTesting()

        engine.setOnlineForTesting(false)
        val initialEpoch = engine.syncEpoch.value

        engine.setOnlineForTesting(true)
        engine.triggerSyncEpoch()

        assertEquals(initialEpoch + 1, engine.syncEpoch.value)
        assertTrue(engine.isOnline.value)
    }
}
