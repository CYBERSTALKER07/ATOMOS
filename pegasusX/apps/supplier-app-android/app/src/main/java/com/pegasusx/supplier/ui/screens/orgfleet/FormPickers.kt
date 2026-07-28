package com.pegasusx.supplier.ui.screens.orgfleet

import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.FleetVehicle

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun NodeTypePicker(selected: String, onSelect: (String) -> Unit) {
    var expanded by remember { mutableStateOf(false) }
    ExposedDropdownMenuBox(expanded = expanded, onExpandedChange = { expanded = !expanded }) {
        OutlinedTextField(
            value = if (selected == "FACTORY") "Factory" else "Warehouse",
            onValueChange = {},
            readOnly = true,
            label = { Text("Node type") },
            modifier = Modifier.menuAnchor().fillMaxWidth(),
        )
        ExposedDropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
            DropdownMenuItem(text = { Text("Warehouse") }, onClick = { onSelect("WAREHOUSE"); expanded = false })
            DropdownMenuItem(text = { Text("Factory") }, onClick = { onSelect("FACTORY"); expanded = false })
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun NodePicker(options: List<Pair<String, String>>, selected: String, onSelect: (String) -> Unit) {
    var expanded by remember { mutableStateOf(false) }
    val label = options.find { it.first == selected }?.second ?: "Select node"
    ExposedDropdownMenuBox(expanded = expanded, onExpandedChange = { expanded = !expanded }) {
        OutlinedTextField(
            value = label,
            onValueChange = {},
            readOnly = true,
            label = { Text("Home node") },
            modifier = Modifier.menuAnchor().fillMaxWidth(),
        )
        ExposedDropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
            options.forEach { (id, name) ->
                DropdownMenuItem(text = { Text(name) }, onClick = { onSelect(id); expanded = false })
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun VehiclePicker(vehicles: List<FleetVehicle>, selected: String, onSelect: (String) -> Unit) {
    var expanded by remember { mutableStateOf(false) }
    val label = vehicles.find { it.vehicleId == selected }?.licensePlate ?: "Assign later"
    ExposedDropdownMenuBox(expanded = expanded, onExpandedChange = { expanded = !expanded }) {
        OutlinedTextField(
            value = label,
            onValueChange = {},
            readOnly = true,
            label = { Text("Vehicle") },
            modifier = Modifier.menuAnchor().fillMaxWidth(),
        )
        ExposedDropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
            DropdownMenuItem(text = { Text("Assign later") }, onClick = { onSelect(""); expanded = false })
            vehicles.forEach { vehicle ->
                DropdownMenuItem(
                    text = { Text(vehicle.licensePlate) },
                    onClick = { onSelect(vehicle.vehicleId); expanded = false },
                )
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RolePicker(selected: String, onSelect: (String) -> Unit) {
    val roles = listOf(
        "WAREHOUSE_ADMIN" to "Warehouse admin",
        "FACTORY_ADMIN" to "Factory admin",
        "PAYLOAD" to "Payload staff",
        "ADMIN" to "Supplier operator",
    )
    var expanded by remember { mutableStateOf(false) }
    val label = roles.find { it.first == selected }?.second ?: selected
    ExposedDropdownMenuBox(expanded = expanded, onExpandedChange = { expanded = !expanded }) {
        OutlinedTextField(
            value = label,
            onValueChange = {},
            readOnly = true,
            label = { Text("Role") },
            modifier = Modifier.menuAnchor().fillMaxWidth(),
        )
        ExposedDropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
            roles.forEach { (id, name) ->
                DropdownMenuItem(text = { Text(name) }, onClick = { onSelect(id); expanded = false })
            }
        }
    }
}
