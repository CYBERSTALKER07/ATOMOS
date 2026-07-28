package com.pegasusx.supplier.ui.screens.orgfleet

import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.ListItem
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.data.model.FleetVehicle
import com.pegasusx.supplier.data.model.SupplierTopologyResponse
import com.pegasusx.supplier.ui.theme.PegasusSpacing

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
