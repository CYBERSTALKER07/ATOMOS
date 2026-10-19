package com.pegasusx.warehouse.ui.screens.vehicles

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ChevronRight
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.Vehicle
import com.pegasusx.warehouse.ui.components.WarehouseStatusChip
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import com.pegasusx.warehouse.R

@Composable
fun VehiclesList(
    vehicles: List<Vehicle>,
    onVehicleClick: (String) -> Unit,
    modifier: Modifier = Modifier
) {
    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        modifier = modifier.fillMaxSize(),
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        items(vehicles, key = { it.vehicleId }) { v ->
            ElevatedCard(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable { onVehicleClick(v.vehicleId) },
            ) {
                Row(
                    modifier = Modifier.padding(PegasusSpacing.lg),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Column(modifier = Modifier.weight(1f)) {
                        Text(v.label.ifBlank { v.licensePlate }, style = MaterialTheme.typography.titleSmall)
                        Text(
                            stringResource(R.string.mobile_warehouse_ui_vehicleclass_capacityvu_vu, v.vehicleClass, v.capacityVu),
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                        Text(
                            v.assignedDriverName.ifBlank { "Unassigned" },
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                        if (!v.isActive) {
                            Text(
                                formatUnavailableReason(v.unavailableReason, v.unavailableNote),
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.tertiary,
                            )
                        }
                    }
                    WarehouseStatusChip(
                        status = if (v.isActive) v.status.ifBlank { "AVAILABLE" } else "UNAVAILABLE",
                    )
                    Icon(
                        Icons.Default.ChevronRight,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }
    }
}
