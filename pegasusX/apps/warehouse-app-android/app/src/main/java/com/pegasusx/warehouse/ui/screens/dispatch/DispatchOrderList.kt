package com.pegasusx.warehouse.ui.screens.dispatch

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.DispatchPreview
import com.pegasusx.warehouse.data.model.Vehicle
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.warehouse.ui.components.DispatchPreviewMapLibre
import com.pegasusx.warehouse.ui.components.OrderDetailOpenMode
import com.pegasusx.warehouse.ui.components.OrderOpsCard
import com.pegasusx.warehouse.ui.components.WarehouseSectionTitle
import com.pegasusx.warehouse.ui.screens.vehicles.FleetTruckDispatchCard
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import java.text.NumberFormat
import com.pegasusx.warehouse.R

private const val DISPATCH_TETRIS_BUFFER = 0.95

/**
 * Orders segment of the Dispatch screen.
 *
 * Renders the fleet truck cards, dispatch-mode selector (Smart / Manual),
 * driver picker, undispatched order list with selection checkboxes,
 * smart-suggest preview warnings, and proposed routes.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DispatchOrderList(
    preview: DispatchPreview,
    fleetVehicles: List<Vehicle>,
    vehicleReasons: Map<String, String>,
    vehicleNotes: Map<String, String>,
    mutatingFleetVehicleId: String?,
    dispatchMode: String,
    selectedDriverId: String,
    selectedOrderIds: Set<String>,
    executing: Boolean,
    opsActingId: String?,
    driverMenuExpanded: Boolean,
    fmt: NumberFormat,
    onDispatchModeChange: (String) -> Unit,
    onDriverSelect: (String) -> Unit,
    onDriverMenuExpandChange: (Boolean) -> Unit,
    onToggleOrder: (String, Boolean) -> Unit,
    onManualDispatch: () -> Unit,
    onSmartDispatch: () -> Unit,
    onProposeDate: (String) -> Unit,
    onReject: (String) -> Unit,
    onOrderClick: (String) -> Unit,
    onVehicleClick: (String) -> Unit,
    onVehicleReasonChange: (String, String) -> Unit,
    onVehicleNoteChange: (String, String) -> Unit,
    onMarkVehicleUnavailable: (Vehicle, String, String) -> Unit,
    onRestoreVehicle: (Vehicle) -> Unit,
) {
    Column(modifier = Modifier.fillMaxSize()) {
        // ── Fleet trucks grid ──
        if (fleetVehicles.isNotEmpty()) {
            WarehouseSectionTitle(
                title = stringResource(R.string.mobile_warehouse_ui_fleet_trucks_size, fleetVehicles.size),
                modifier = Modifier.padding(horizontal = PegasusSpacing.lg, vertical = PegasusSpacing.sm),
            )
            LazyVerticalGrid(
                columns = GridCells.Adaptive(minSize = 340.dp),
                modifier = Modifier.heightIn(max = 320.dp),
                contentPadding = PaddingValues(horizontal = PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                items(fleetVehicles, key = { it.vehicleId }) { vehicle ->
                    val selectedReason = vehicleReasons[vehicle.vehicleId]
                        ?: vehicle.unavailableReason?.takeIf { it.isNotBlank() }
                        ?: "MANUAL_HOLD"
                    val customNote = vehicleNotes[vehicle.vehicleId]
                        ?: vehicle.unavailableNote.orEmpty()
                    FleetTruckDispatchCard(
                        vehicle = vehicle,
                        selectedReason = selectedReason,
                        customNote = customNote,
                        mutating = mutatingFleetVehicleId == vehicle.vehicleId,
                        onReasonChange = { reason ->
                            onVehicleReasonChange(vehicle.vehicleId, reason)
                        },
                        onNoteChange = { note ->
                            onVehicleNoteChange(vehicle.vehicleId, note)
                        },
                        onMarkUnavailable = {
                            val note = if (selectedReason == "OTHER") customNote.trim().takeIf { it.isNotEmpty() } else null
                            onMarkVehicleUnavailable(vehicle, selectedReason, note ?: "")
                        },
                        onRestore = { onRestoreVehicle(vehicle) },
                        onOpenDetail = { onVehicleClick(vehicle.vehicleId) },
                    )
                }
            }
        }

        // ── Empty state ──
        if (preview.undispatchedOrders.isEmpty()) {
            PegasusStatePane(
                kind = PegasusStateKind.Empty,
                headline = "All orders dispatched",
                body = "No undispatched orders remain in the preview queue.",
            )
        } else {
            val selectedDriver = preview.availableDrivers.firstOrNull { it.driverId == selectedDriverId }
            val selectedVolume = preview.undispatchedOrders
                .filter { selectedOrderIds.contains(it.orderId) }
                .sumOf { it.volumeVu }
            val effectiveMax = when {
                selectedDriver?.freeVolumeVu != null && selectedDriver.freeVolumeVu > 0 ->
                    selectedDriver.freeVolumeVu * DISPATCH_TETRIS_BUFFER
                else -> (selectedDriver?.maxVolumeVu ?: 0.0) * DISPATCH_TETRIS_BUFFER
            }

            Column(modifier = Modifier.fillMaxSize()) {
                // ── Dispatch-mode selector & driver picker ──
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = PegasusSpacing.lg, vertical = PegasusSpacing.md),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                ) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                    ) {
                        FilterChip(
                            selected = dispatchMode == "smart",
                            onClick = { onDispatchModeChange("smart") },
                            label = { Text("Smart fleet") },
                            modifier = Modifier.weight(1f),
                        )
                        FilterChip(
                            selected = dispatchMode == "manual",
                            onClick = { onDispatchModeChange("manual") },
                            label = { Text("Manual truck") },
                            modifier = Modifier.weight(1f),
                        )
                    }
                    if (dispatchMode == "manual") {
                    Box {
                        OutlinedButton(
                            onClick = { onDriverMenuExpandChange(true) },
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            Text(
                                selectedDriver?.let { "${it.name} · ${it.maxVolumeVu} VU max" }
                                    ?: "Select truck / driver",
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                            )
                        }
                        DropdownMenu(
                            expanded = driverMenuExpanded,
                            onDismissRequest = { onDriverMenuExpandChange(false) },
                        ) {
                            preview.availableDrivers.forEach { driver ->
                                DropdownMenuItem(
                                    text = {
                                        Text(stringResource(R.string.mobile_warehouse_ui_name_truckstatus, driver.name, driver.vehicleLabel.ifBlank { driver.truckStatus }),
                                            maxLines = 1,
                                            overflow = TextOverflow.Ellipsis,
                                        )
                                    },
                                    onClick = {
                                        onDriverSelect(driver.driverId)
                                        onDriverMenuExpandChange(false)
                                    },
                                )
                            }
                        }
                    }
                    if (selectedDriver != null) {
                        Text(stringResource(R.string.mobile_warehouse_ui_loaded_format_format_2_vu_effective, "%.1f".format(selectedVolume), "%.1f".format(effectiveMax)),
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                    }
                    if (dispatchMode == "manual") {
                    Button(
                        onClick = onManualDispatch,
                        enabled = !executing && selectedDriverId.isNotBlank() && selectedOrderIds.isNotEmpty(),
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Text(if (executing) "Dispatching…" else "Manual (${selectedOrderIds.size})")
                    }
                    } else {
                    OutlinedButton(
                        onClick = onSmartDispatch,
                        enabled = !executing && preview.undispatchedOrders.isNotEmpty() && preview.availableDrivers.isNotEmpty(),
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Text("Smart Dispatch")
                    }
                    }
                    if (preview.fleetEffectiveCapacityVu > 0) {
                        Text(stringResource(R.string.mobile_warehouse_ui_fleet_format_vu_effective, "%.1f".format(preview.fleetEffectiveCapacityVu)),
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                }

                // ── Orders grid ──
                LazyVerticalGrid(
                    columns = GridCells.Adaptive(minSize = 340.dp),
                    contentPadding = PaddingValues(horizontal = PegasusSpacing.lg, vertical = PegasusSpacing.md),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                    horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                ) {
                items(preview.undispatchedOrders, key = { it.orderId }) { o ->
                    OrderOpsCard(
                        retailerName = o.retailerName,
                        orderId = o.orderId,
                        state = "PENDING",
                        amountLabel = fmt.format(o.totalUzs) + " ${com.pegasus.design.sessionPackCurrency()} · ${"%.1f".format(o.volumeVu)} VU",
                        showOpsMenu = true,
                        detailOpenMode = OrderDetailOpenMode.Double,
                        canDelay = true,
                        canReject = true,
                        enabled = opsActingId != o.orderId,
                        onOpenDetail = { onOrderClick(o.orderId) },
                        onDelay = { onProposeDate(o.orderId) },
                        onReject = { onReject(o.orderId) },
                        leadingContent = {
                            Checkbox(
                                checked = selectedOrderIds.contains(o.orderId),
                                onCheckedChange = { checked ->
                                    onToggleOrder(o.orderId, checked)
                                },
                            )
                        },
                    )
                }

                // ── Smart suggest preview warnings ──
                if (preview.windowConstrainedCount > 0 || preview.optimizerWarnings.isNotEmpty()) {
                    item(span = { GridItemSpan(maxLineSpan) }) {
                        Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                            WarehouseSectionTitle("Smart suggest preview")
                            if (preview.windowConstrainedCount > 0) {
                                Text(
                                    stringResource(R.string.mobile_warehouse_ui_windowconstrainedcount_order_s_constrained_by_receiving_window, preview.windowConstrainedCount),
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.tertiary,
                                )
                            }
                            preview.optimizerSource?.let { source ->
                                Text(stringResource(R.string.mobile_warehouse_ui_source_source_2, source), style = MaterialTheme.typography.bodySmall)
                            }
                            preview.optimizerWarnings.forEach { warning ->
                                Text(warning, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.tertiary)
                            }
                        }
                    }
                }

                // ── Proposed routes ──
                if (preview.proposedRoutes.isNotEmpty()) {
                    item(span = { GridItemSpan(maxLineSpan) }) {
                        Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                            WarehouseSectionTitle("Smart suggest routes (${preview.proposedRoutes.size})")
                            DispatchPreviewMapLibre(routes = preview.proposedRoutes)
                        }
                    }
                    items(preview.proposedRoutes.size) { index ->
                        val route = preview.proposedRoutes[index]
                        ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                            Column(modifier = Modifier.padding(PegasusSpacing.lg)) {
                                Text(
                                    route.driverName ?: route.driverId ?: "Driver",
                                    style = MaterialTheme.typography.titleSmall,
                                )
                                Text(stringResource(R.string.mobile_warehouse_ui_size_stops_format_vu, route.stopCount ?: route.orderIds.size, "%.1f".format(route.volumeVu ?: 0.0)),
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                                Text(
                                    route.orderIds.joinToString(" → "),
                                    style = MaterialTheme.typography.labelSmall,
                                    maxLines = 2,
                                    overflow = TextOverflow.Ellipsis,
                                )
                            }
                        }
                    }
                }
                }
            }
        }
    }
}
