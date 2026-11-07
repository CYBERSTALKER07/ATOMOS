package com.pegasus.payload.ui.home

import com.pegasus.payload.data.model.Manifest
import com.pegasus.payload.data.model.Truck

/** GS-U7 payload board: columns are manifest states, not a single trucks count. */
object ManifestBoard {
    val BOARD_STATES = listOf("DRAFT", "LOADING", "SEALED", "DISPATCHED")

    fun canonicalState(state: String?): String {
        val s = state?.trim()?.uppercase().orEmpty()
        return if (s in BOARD_STATES) s else ""
    }

    fun isBoardState(state: String?): Boolean = canonicalState(state).isNotEmpty()

    data class Column(val state: String, val trucks: List<Truck>)

    fun attach(trucks: List<Truck>, manifests: List<Manifest>): List<Truck> {
        return trucks.map { truck ->
            if (canonicalState(truck.truckStatus).isNotEmpty()) return@map truck
            val match = manifests
                .filter { it.matchesTruck(truck.id) && isBoardState(it.state) }
                .maxByOrNull { it.createdAt }
            if (match == null) truck
            else truck.copy(
                truckStatus = canonicalState(match.state),
                usedVolumeVu = match.totalVolumeVu.toLong(),
                maxVolumeVu = match.maxVolumeVu.toLong(),
                stopCount = match.stopCount,
            )
        }
    }

    fun group(trucks: List<Truck>): List<Column> {
        return BOARD_STATES.map { state ->
            Column(state = state, trucks = trucks.filter { canonicalState(it.truckStatus) == state })
        }
    }

    fun unassigned(trucks: List<Truck>): List<Truck> =
        trucks.filter { canonicalState(it.truckStatus).isEmpty() }
}
