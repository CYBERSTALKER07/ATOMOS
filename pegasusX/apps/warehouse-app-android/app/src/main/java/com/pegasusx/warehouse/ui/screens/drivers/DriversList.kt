package com.pegasusx.warehouse.ui.screens.drivers

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.Driver
import com.pegasusx.warehouse.data.model.Vehicle
import com.pegasusx.warehouse.ui.components.WarehouseStatusChip
import com.pegasusx.warehouse.ui.theme.PegasusSpacing

private val DRIVER_UNAVAILABLE_REASON_LABELS = mapOf(
    "MAINTENANCE" to "Maintenance",
    "TRUCK_DAMAGED" to "Truck Damaged",
    "REGULATORY_HOLD" to "Regulatory Hold",
    "MANUAL_HOLD" to "Manual Hold",
)

@Composable
fun DriversList(
    drivers: List<Driver>,
    vehicles: List<Vehicle>,
    assigningDriverId: String?,
    onAssignClick: (Driver) -> Unit,
    modifier: Modifier = Modifier
) {
    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        modifier = modifier.fillMaxSize(),
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        items(drivers, key = { it.driverId }) { driver ->
            ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                Row(modifier = Modifier.padding(PegasusSpacing.lg), verticalAlignment = Alignment.CenterVertically) {
                    Column(modifier = Modifier.weight(1f)) {
                        Text(driver.name, style = MaterialTheme.typography.titleSmall)
                        Text(driver.phone, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        Text(
                            assignedVehicleLabel(driver, vehicles),
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                        assignedVehicleReason(driver, vehicles)?.let { reason ->
                            Text(
                                reason,
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.tertiary,
                            )
                        }
                    }
                    Column(horizontalAlignment = Alignment.End, verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                        WarehouseStatusChip(status = driver.truckStatus.ifBlank { "IDLE" })
                        OutlinedButton(
                            onClick = { onAssignClick(driver) },
                            enabled = assigningDriverId != driver.driverId,
                        ) {
                            if (assigningDriverId == driver.driverId) {
                                CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp)
                            } else {
                                Text(if (driver.vehicleId.isNullOrBlank()) "Assign" else "Reassign")
                            }
                        }
                    }
                }
            }
        }
    }
}

private fun assignedVehicleLabel(driver: Driver, vehicles: List<Vehicle>): String {
    val vehicleId = driver.vehicleId ?: return "Unassigned"
    val vehicle = vehicles.firstOrNull { it.vehicleId == vehicleId } ?: return "Assigned vehicle unavailable"
    return vehicleLabel(vehicle)
}

private fun assignedVehicleReason(driver: Driver, vehicles: List<Vehicle>): String? {
    val vehicleId = driver.vehicleId ?: return null
    if (!driver.vehicleIsActive) {
        return driver.vehicleUnavailableReason?.takeIf { it.isNotBlank() }
            ?.let { "Vehicle unavailable: ${vehicleUnavailableReasonLabel(it)}" }
            ?: "Vehicle unavailable"
    }
    val vehicle = vehicles.firstOrNull { it.vehicleId == vehicleId } ?: return null
    if (!vehicle.isActive) {
        return vehicle.unavailableReason?.takeIf { it.isNotBlank() }
            ?.let { "Vehicle unavailable: ${vehicleUnavailableReasonLabel(it)}" }
            ?: "Vehicle unavailable"
    }
    return null
}

fun vehicleLabel(vehicle: Vehicle): String {
    val title = if (vehicle.label.isBlank()) vehicle.licensePlate else vehicle.label
    return listOf(title, vehicle.vehicleClass).filter { it.isNotBlank() }.joinToString(" · ")
}

private fun vehicleUnavailableReasonLabel(reason: String): String {
    return DRIVER_UNAVAILABLE_REASON_LABELS[reason]
        ?: reason.lowercase().split('_').joinToString(" ") { token ->
            token.replaceFirstChar { ch -> ch.titlecase() }
        }
}
