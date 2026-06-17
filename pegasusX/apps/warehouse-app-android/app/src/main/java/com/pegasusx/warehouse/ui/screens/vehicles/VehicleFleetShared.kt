package com.pegasusx.warehouse.ui.screens.vehicles

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.Vehicle
import com.pegasusx.warehouse.ui.components.WarehouseStatusChip
import com.pegasusx.warehouse.ui.theme.PegasusSpacing

val VEHICLE_UNAVAILABLE_REASONS = listOf(
    "MAINTENANCE" to "Maintenance",
    "TRUCK_DAMAGED" to "Truck Damaged",
    "REGULATORY_HOLD" to "Regulatory Hold",
    "MANUAL_HOLD" to "Manual Hold",
    "OTHER" to "Other",
)

fun formatUnavailableReason(reason: String?, note: String? = null): String {
    if (reason.isNullOrBlank()) return note?.trim().orEmpty()
    if (reason.uppercase() == "OTHER" && !note.isNullOrBlank()) return note.trim()
    return vehicleUnavailableReasonLabel(reason)
}

fun vehicleUnavailableReasonLabel(reason: String): String =
    VEHICLE_UNAVAILABLE_REASONS.firstOrNull { it.first == reason }?.second
        ?: reason.lowercase().split('_').joinToString(" ") { token ->
            token.replaceFirstChar { ch -> ch.titlecase() }
        }

fun vehicleStatusLabel(isActive: Boolean) = if (isActive) "Active" else "Unavailable"

@Composable
fun VehicleAvailabilityPanel(
    vehicle: Vehicle,
    mutating: Boolean,
    onMarkUnavailable: (reason: String, note: String?) -> Unit,
    onRestore: () -> Unit,
    compact: Boolean = false,
    modifier: Modifier = Modifier,
) {
    var reason by remember(vehicle.vehicleId, vehicle.isActive, vehicle.unavailableReason) {
        mutableStateOf(vehicle.unavailableReason?.takeIf { it.isNotBlank() } ?: "MANUAL_HOLD")
    }
    var note by remember(vehicle.vehicleId, vehicle.unavailableNote) {
        mutableStateOf(vehicle.unavailableNote.orEmpty())
    }

    ElevatedCard(modifier = modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier.padding(if (compact) PegasusSpacing.md else PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    Text("Availability", style = MaterialTheme.typography.titleSmall)
                    Text(
                        "Dispatch excludes unavailable trucks immediately.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                WarehouseStatusChip(status = if (vehicle.isActive) "ACTIVE" else "UNAVAILABLE")
            }

            if (!vehicle.isActive && (!vehicle.unavailableReason.isNullOrBlank() || !vehicle.unavailableNote.isNullOrBlank())) {
                Text(
                    formatUnavailableReason(vehicle.unavailableReason, vehicle.unavailableNote),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.tertiary,
                )
            }

            if (vehicle.isActive) {
                Text("Unavailable reason", style = MaterialTheme.typography.labelMedium)
                Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                    VEHICLE_UNAVAILABLE_REASONS.forEach { (code, label) ->
                        OutlinedButton(
                            onClick = { reason = code },
                            enabled = !mutating,
                            modifier = Modifier.weight(1f),
                        ) {
                            Text(
                                label,
                                style = MaterialTheme.typography.labelSmall,
                                maxLines = 1,
                            )
                        }
                    }
                }
                if (reason == "OTHER") {
                    OutlinedTextField(
                        value = note,
                        onValueChange = { note = it },
                        label = { Text("Custom reason") },
                        singleLine = true,
                        enabled = !mutating,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
                Button(
                    onClick = {
                        onMarkUnavailable(
                            reason,
                            if (reason == "OTHER") note.trim().takeIf { it.isNotEmpty() } else null,
                        )
                    },
                    enabled = !mutating && (reason != "OTHER" || note.trim().isNotEmpty()),
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    if (mutating) {
                        CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp)
                    } else {
                        Text("Mark unavailable")
                    }
                }
            } else {
                OutlinedButton(
                    onClick = onRestore,
                    enabled = !mutating,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    if (mutating) {
                        CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp)
                    } else {
                        Text("Restore truck")
                    }
                }
            }
        }
    }
}

@Composable
fun FleetTruckDispatchCard(
    vehicle: Vehicle,
    selectedReason: String,
    customNote: String,
    mutating: Boolean,
    onReasonChange: (String) -> Unit,
    onNoteChange: (String) -> Unit,
    onMarkUnavailable: () -> Unit,
    onRestore: () -> Unit,
    onOpenDetail: (() -> Unit)? = null,
    modifier: Modifier = Modifier,
) {
    ElevatedCard(modifier = modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier
                .padding(PegasusSpacing.md)
                .then(if (onOpenDetail != null) Modifier.clickable(onClick = onOpenDetail) else Modifier),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.Top,
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        vehicle.label.ifBlank { vehicle.licensePlate },
                        style = MaterialTheme.typography.titleSmall,
                    )
                    Text(
                        "${vehicle.licensePlate} · ${vehicle.vehicleClass}",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    if (!vehicle.isActive) {
                        Text(
                            formatUnavailableReason(vehicle.unavailableReason, vehicle.unavailableNote),
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.tertiary,
                        )
                    }
                }
                WarehouseStatusChip(status = if (vehicle.isActive) "ACTIVE" else "UNAVAILABLE")
            }

            if (vehicle.isActive) {
                Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                    VEHICLE_UNAVAILABLE_REASONS.take(3).forEach { (code, label) ->
                        OutlinedButton(
                            onClick = { onReasonChange(code) },
                            enabled = !mutating,
                        ) {
                            Text(label, style = MaterialTheme.typography.labelSmall)
                        }
                    }
                }
                Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                    VEHICLE_UNAVAILABLE_REASONS.drop(3).forEach { (code, label) ->
                        OutlinedButton(
                            onClick = { onReasonChange(code) },
                            enabled = !mutating,
                        ) {
                            Text(label, style = MaterialTheme.typography.labelSmall)
                        }
                    }
                }
                if (selectedReason == "OTHER") {
                    OutlinedTextField(
                        value = customNote,
                        onValueChange = onNoteChange,
                        label = { Text("Custom reason") },
                        singleLine = true,
                        enabled = !mutating,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
                OutlinedButton(
                    onClick = onMarkUnavailable,
                    enabled = !mutating && (selectedReason != "OTHER" || customNote.trim().isNotEmpty()),
                ) {
                    Text(if (mutating) "Updating…" else "Mark unavailable")
                }
            } else {
                OutlinedButton(onClick = onRestore, enabled = !mutating) {
                    Text(if (mutating) "Updating…" else "Restore")
                }
            }
        }
    }
}
