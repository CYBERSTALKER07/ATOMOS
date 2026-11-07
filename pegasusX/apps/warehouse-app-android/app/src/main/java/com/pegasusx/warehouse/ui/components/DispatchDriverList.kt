package com.pegasusx.warehouse.ui.components

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.AvailableDriver
import com.pegasus.design.ui.PegasusStateKind
import com.pegasus.design.ui.PegasusStatePane
import com.pegasusx.warehouse.ui.screens.vehicles.vehicleUnavailableReasonLabel
import com.pegasusx.warehouse.ui.theme.PegasusSpacing

/**
 * Drivers segment of the Dispatch screen.
 *
 * Renders available and unavailable driver cards with status badges,
 * VU capacity, and unavailability reasons.
 */
@Composable
fun DispatchDriverList(
    availableDrivers: List<AvailableDriver>,
    unavailableDrivers: List<AvailableDriver>,
) {
    if (availableDrivers.isEmpty() && unavailableDrivers.isEmpty()) {
        PegasusStatePane(
            kind = PegasusStateKind.Empty,
            headline = "No drivers",
            body = "Available and unavailable drivers will appear here.",
        )
    } else {
        LazyVerticalGrid(
            columns = GridCells.Adaptive(minSize = 340.dp),
            contentPadding = PaddingValues(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            if (availableDrivers.isNotEmpty()) {
                item(span = { GridItemSpan(maxLineSpan) }) { WarehouseSectionTitle("Available") }
            }
            items(availableDrivers, key = { it.driverId }) { d ->
                val supporting = buildString {
                    append(d.vehicleLabel.ifBlank { d.phone.ifBlank { d.truckStatus.ifBlank { "No vehicle" } } })
                    if (d.freeVolumeVu != null && d.freeVolumeVu > 0) {
                        append(" · ${"%.1f".format(d.freeVolumeVu)} VU free")
                    }
                }
                WarehouseOpsListCard(
                    headline = d.name,
                    supporting = supporting,
                    status = d.truckStatus.ifBlank { "IDLE" },
                )
            }
            if (unavailableDrivers.isNotEmpty()) {
                item(span = { GridItemSpan(maxLineSpan) }) { WarehouseSectionTitle("Vehicle unavailable") }
            }
            items(unavailableDrivers, key = { "unavailable-${it.driverId}" }) { d ->
                WarehouseOpsListCard(
                    headline = d.name,
                    supporting = buildString {
                        append(d.vehicleLabel.ifBlank { d.phone.ifBlank { "Assigned vehicle unavailable" } })
                        if (!d.unavailableReason.isNullOrBlank()) {
                            append(" · ")
                            append(vehicleUnavailableReasonLabel(d.unavailableReason))
                        }
                    },
                    status = d.truckStatus.ifBlank { "UNAVAILABLE" },
                )
            }
        }
    }
}
