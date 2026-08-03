package com.pegasusx.supplier.ui.screens.orgfleet.components

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.*
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing

@Composable
fun DriverRoster(drivers: List<FleetDriver>, topology: SupplierTopologyResponse?) {
    if (drivers.isEmpty()) {
        PegasusStatePane(PegasusStateKind.Empty, "No drivers", "Create a driver to start fleet onboarding.")
        return
    }
    LazyColumn(contentPadding = PaddingValues(PegasusSpacing.lg)) {
        items(drivers, key = { it.driverId }) { driver ->
            ListItem(
                headlineContent = { Text(driver.name) },
                supportingContent = {
                    Text("${nodeLabel(driver.homeNodeType, driver.homeNodeId, topology)} · ${driver.phone}")
                },
            )
        }
    }
}

@Composable
fun VehicleRoster(vehicles: List<FleetVehicle>, topology: SupplierTopologyResponse?) {
    if (vehicles.isEmpty()) {
        PegasusStatePane(PegasusStateKind.Empty, "No vehicles", "Create a vehicle for driver assignment.")
        return
    }
    LazyColumn(contentPadding = PaddingValues(PegasusSpacing.lg)) {
        items(vehicles, key = { it.vehicleId }) { vehicle ->
            ListItem(
                headlineContent = { Text(vehicle.label ?: vehicle.licensePlate) },
                supportingContent = {
                    Text("${vehicle.licensePlate} · ${nodeLabel(vehicle.homeNodeType, vehicle.homeNodeId, topology)}")
                },
            )
        }
    }
}

@Composable
fun OrgRoster(
    members: List<SupplierOrgMember>,
    onEdit: (SupplierOrgMember) -> Unit,
    onDeactivate: (String) -> Unit,
    actionId: String?,
) {
    if (members.isEmpty()) {
        PegasusStatePane(PegasusStateKind.Empty, "No org members", "Create warehouse, factory, or payload staff.")
        return
    }
    LazyColumn(contentPadding = PaddingValues(PegasusSpacing.lg)) {
        items(members, key = { it.userId }) { member ->
            ListItem(
                headlineContent = { Text(member.name) },
                supportingContent = {
                    Text("${member.supplierRole} · ${member.phone} · ${if (member.isActive) "Active" else "Inactive"}")
                },
                trailingContent = {
                    Row {
                        TextButton(
                            enabled = actionId != member.userId,
                            onClick = { onEdit(member) },
                        ) { Text("Edit") }
                        if (member.isActive) {
                            TextButton(
                                enabled = actionId != member.userId,
                                onClick = { onDeactivate(member.userId) },
                            ) { Text("Deactivate") }
                        }
                    }
                },
            )
        }
    }
}