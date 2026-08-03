package com.pegasusx.supplier.ui.screens.orgfleet

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SupplierOrgMemberCreateRequest
import com.pegasusx.supplier.data.model.SupplierTopologyResponse
import com.pegasusx.supplier.ui.theme.PegasusSpacing

@Composable
fun CreateOrgMemberDialog(
    topology: SupplierTopologyResponse,
    onDismiss: () -> Unit,
    onCreate: (SupplierOrgMemberCreateRequest) -> Unit,
) {
    var name by remember { mutableStateOf("") }
    var email by remember { mutableStateOf("") }
    var phone by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var role by remember { mutableStateOf("WAREHOUSE_ADMIN") }
    var nodeType by remember { mutableStateOf("WAREHOUSE") }
    var nodeId by remember { mutableStateOf("") }
    val nodeOptions = when (role) {
        "FACTORY_ADMIN" -> topology.factories.map { it.factoryId to it.name }
        "ADMIN" -> emptyList()
        else -> if (nodeType == "FACTORY") topology.factories.map { it.factoryId to it.name }
        else topology.warehouses.map { it.warehouseId to it.name }
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Create org member") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                OutlinedTextField(name, { name = it }, label = { Text("Name") }, modifier = Modifier.fillMaxWidth())
                OutlinedTextField(email, { email = it }, label = { Text("Email") }, modifier = Modifier.fillMaxWidth())
                OutlinedTextField(phone, { phone = it }, label = { Text("Phone") }, modifier = Modifier.fillMaxWidth())
                OutlinedTextField(password, { password = it }, label = { Text("Password") }, modifier = Modifier.fillMaxWidth())
                RolePicker(role) { role = it; nodeId = "" }
                if (role == "PAYLOAD") NodeTypePicker(nodeType) { nodeType = it; nodeId = "" }
                if (role != "ADMIN" && nodeOptions.isNotEmpty()) NodePicker(nodeOptions, nodeId) { nodeId = it }
            }
        },
        confirmButton = {
            TextButton(onClick = {
                if (name.isBlank() || phone.isBlank() || password.isBlank()) return@TextButton
                if (role != "ADMIN" && nodeId.isBlank()) return@TextButton
                val warehouseId = if (role == "WAREHOUSE_ADMIN" || (role == "PAYLOAD" && nodeType == "WAREHOUSE")) nodeId else null
                val factoryId = if (role == "FACTORY_ADMIN" || (role == "PAYLOAD" && nodeType == "FACTORY")) nodeId else null
                onCreate(
                    SupplierOrgMemberCreateRequest(
                        name = name.trim(),
                        email = email.ifBlank { null },
                        phone = phone.trim(),
                        password = password,
                        supplierRole = role,
                        assignedWarehouseId = warehouseId,
                        assignedFactoryId = factoryId,
                    ),
                )
            }) { Text("Create") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}
