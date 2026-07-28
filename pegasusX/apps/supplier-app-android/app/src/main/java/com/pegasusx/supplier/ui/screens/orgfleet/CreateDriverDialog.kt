package com.pegasusx.supplier.ui.screens.orgfleet

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.FleetDriverCreateRequest
import com.pegasusx.supplier.data.model.FleetVehicle
import com.pegasusx.supplier.data.model.SupplierTopologyResponse
import com.pegasusx.supplier.ui.theme.PegasusSpacing

@Composable
fun CreateDriverDialog(
    topology: SupplierTopologyResponse,
    vehicles: List<FleetVehicle>,
    onDismiss: () -> Unit,
    onCreate: (FleetDriverCreateRequest) -> Unit,
) {
    var name by remember { mutableStateOf("") }
    var phone by remember { mutableStateOf("") }
    var pin by remember { mutableStateOf("") }
    var nodeType by remember { mutableStateOf("WAREHOUSE") }
    var nodeId by remember { mutableStateOf("") }
    var vehicleId by remember { mutableStateOf("") }
    val nodeOptions = if (nodeType == "FACTORY") {
        topology.factories.map { it.factoryId to it.name }
    } else {
        topology.warehouses.map { it.warehouseId to it.name }
    }
    val vehicleOptions = vehicles.filter { it.homeNodeType == nodeType && it.homeNodeId == nodeId }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Create driver") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                OutlinedTextField(name, { name = it }, label = { Text("Name") }, modifier = Modifier.fillMaxWidth())
                OutlinedTextField(phone, { phone = it }, label = { Text("Phone") }, modifier = Modifier.fillMaxWidth())
                OutlinedTextField(pin, { pin = it }, label = { Text("PIN") }, modifier = Modifier.fillMaxWidth())
                NodeTypePicker(nodeType) { nodeType = it; nodeId = ""; vehicleId = "" }
                NodePicker(nodeOptions, nodeId) { nodeId = it; vehicleId = "" }
                if (vehicleOptions.isNotEmpty()) {
                    VehiclePicker(vehicleOptions, vehicleId) { vehicleId = it }
                }
            }
        },
        confirmButton = {
            TextButton(onClick = {
                if (name.isBlank() || phone.isBlank() || pin.isBlank() || nodeId.isBlank()) return@TextButton
                onCreate(
                    FleetDriverCreateRequest(
                        name = name.trim(),
                        phone = phone.trim(),
                        pin = pin,
                        homeNodeType = nodeType,
                        homeNodeId = nodeId,
                        vehicleId = vehicleId.ifBlank { null },
                    ),
                )
            }) { Text("Create") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}
