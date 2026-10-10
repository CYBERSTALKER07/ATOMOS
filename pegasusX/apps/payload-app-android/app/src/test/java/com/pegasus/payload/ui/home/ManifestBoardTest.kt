package com.pegasus.payload.ui.home

import com.pegasus.payload.data.model.Manifest
import com.pegasus.payload.data.model.Truck
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class ManifestBoardTest {

    @Test
    fun boardHasFourManifestStateColumns() {
        val trucks = listOf(
            Truck(id = "d", truckStatus = "DRAFT"),
            Truck(id = "l", truckStatus = "LOADING"),
            Truck(id = "s", truckStatus = "SEALED"),
            Truck(id = "x", truckStatus = "DISPATCHED"),
            Truck(id = "done", truckStatus = "COMPLETED"),
            Truck(id = "none", truckStatus = ""),
        )
        val cols = ManifestBoard.group(trucks)
        assertEquals(listOf("DRAFT", "LOADING", "SEALED", "DISPATCHED"), cols.map { it.state })
        assertEquals(listOf("d"), cols[0].trucks.map { it.id })
        assertEquals(listOf("l"), cols[1].trucks.map { it.id })
        assertEquals(listOf("s"), cols[2].trucks.map { it.id })
        assertEquals(listOf("x"), cols[3].trucks.map { it.id })
        assertEquals(listOf("done", "none"), ManifestBoard.unassigned(trucks).map { it.id })
    }

    @Test
    fun attachUsesManifestStateNotInventedDraft() {
        val trucks = listOf(Truck(id = "veh-1"), Truck(id = "veh-2"))
        val manifests = listOf(
            Manifest(manifestId = "m1", vehicleId = "veh-1", state = "SEALED", totalVolumeVu = 8.0, maxVolumeVu = 40.0, stopCount = 2),
            Manifest(manifestId = "m2", vehicleId = "veh-2", state = "COMPLETED"),
        )
        val attached = ManifestBoard.attach(trucks, manifests)
        assertEquals("SEALED", attached[0].truckStatus)
        assertEquals(8L, attached[0].usedVolumeVu)
        assertEquals("", attached[1].truckStatus)
        val cols = ManifestBoard.group(attached)
        assertEquals(1, cols[2].trucks.size)
        assertTrue(cols[0].trucks.isEmpty())
    }

    @Test
    fun emptyBoardIsEmptyColumnsNotZeroTheatre() {
        val cols = ManifestBoard.group(emptyList())
        assertEquals(4, cols.size)
        assertTrue(cols.all { it.trucks.isEmpty() })
    }

    @Test
    fun pulseFailureDoesNotTreatAsEmptyTimeline() {
        val src = java.io.File("src/main/java/com/pegasus/payload/ui/home/HomeViewModel.kt").readText()
        assertTrue(src.contains("PulseHonesty.FAILED"))
        assertTrue(!src.contains("pulseEvents = emptyList()"))
        val screen = java.io.File("src/main/java/com/pegasus/payload/ui/home/HomeScreen.kt").readText()
        assertTrue(screen.contains("error = state.pulseError"))
    }
}
