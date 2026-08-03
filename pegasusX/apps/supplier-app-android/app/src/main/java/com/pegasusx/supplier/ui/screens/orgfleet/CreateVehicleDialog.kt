package com.pegasusx.supplier.ui.screens.orgfleet

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.FleetVehicleCreateRequest
import com.pegasusx.supplier.data.model.SupplierTopologyResponse
import com.pegasusx.supplier.ui.theme.PegasusSpacing

@Composable
fun CreateVehicleDialog(
    topology: SupplierTopologyResponse,
    onDismiss: () -> Unit,
    onCreate: (FleetVehicleCreateRequest) -> Unit,
) {
    var label by remember { mutableStateOf("") }
    var plate by remember { mutableStateOf("") }
    var nodeType by remember { mutableStateOf("WAREHOUSE") }
    var nodeId by remember { mutableStateOf("") }
    val nodeOptions = if (nodeType == "FACTORY") {
        topology.factories.map { it.factoryId to it.name }
    } else {
        topology.warehouses.map { it.warehouseId to it.name }
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Create vehicle") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                OutlinedTextField(label, { label = it }, label = { Text("Label") }, modifier = Modifier.fillMaxWidth())
                OutlinedTextField(plate, { plate = it.uppercase() }, label = { Text("License plate") }, modifier = Modifier.fillMaxWidth())
                NodeTypePicker(nodeType) { nodeType = it; nodeId = "" }
                NodePicker(nodeOptions, nodeId) { nodeId = it }
            }
        },
        confirmButton = {
            TextButton(onClick = {
                if (plate.isBlank() || nodeId.isBlank()) return@TextButton
                onCreate(
                    FleetVehicleCreateRequest(
                        label = label.ifBlank { null },
                        licensePlate = plate.trim(),
                        homeNodeType = nodeType,
                        homeNodeId = nodeId,
                    ),
                )
            }) { Text("Create") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}
