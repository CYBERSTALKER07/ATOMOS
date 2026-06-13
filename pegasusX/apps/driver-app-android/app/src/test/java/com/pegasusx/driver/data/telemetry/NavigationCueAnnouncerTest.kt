package com.pegasusx.driver.data.telemetry

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class NavigationCueAnnouncerTest {

    @Test
    fun shouldAnnounceManeuverAdvance_trueWhenIndexIncreases() {
        assertTrue(shouldAnnounceManeuverAdvance(previousIndex = 0, nextIndex = 1))
        assertTrue(shouldAnnounceManeuverAdvance(previousIndex = 1, nextIndex = 3))
    }

    @Test
    fun shouldAnnounceManeuverAdvance_falseWhenUnchangedOrRegressed() {
        assertFalse(shouldAnnounceManeuverAdvance(previousIndex = 1, nextIndex = 1))
        assertFalse(shouldAnnounceManeuverAdvance(previousIndex = 2, nextIndex = 1))
        assertFalse(shouldAnnounceManeuverAdvance(previousIndex = 0, nextIndex = 0))
    }
}
