package com.pegasusx.supplier.ui.screens.orgfleet

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SupplierOrgMember
import com.pegasusx.supplier.data.model.SupplierOrgMemberUpdateRequest
import com.pegasusx.supplier.data.model.SupplierTopologyResponse
import com.pegasusx.supplier.ui.theme.PegasusSpacing

@Composable
fun EditOrgMemberDialog(
    member: SupplierOrgMember,
    topology: SupplierTopologyResponse,
    onDismiss: () -> Unit,
    onSave: (SupplierOrgMemberUpdateRequest) -> Unit,
) {
    var name by remember { mutableStateOf(member.name) }
    var role by remember { mutableStateOf(member.supplierRole) }
    var nodeId by remember {
        mutableStateOf(member.assignedWarehouseId ?: member.assignedFactoryId ?: "")
    }
    val nodeOptions = when (role) {
        "FACTORY_ADMIN" -> topology.factories.map { it.factoryId to it.name }
        "ADMIN" -> emptyList()
        else -> topology.warehouses.map { it.warehouseId to it.name }
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Edit org member") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                OutlinedTextField(name, { name = it }, label = { Text("Name") }, modifier = Modifier.fillMaxWidth())
                RolePicker(role) { role = it; nodeId = "" }
                if (role != "ADMIN" && nodeOptions.isNotEmpty()) {
                    NodePicker(nodeOptions, nodeId) { nodeId = it }
                }
            }
        },
        confirmButton = {
            TextButton(onClick = {
                if (name.isBlank()) return@TextButton
                val warehouseId = if (role == "WAREHOUSE_ADMIN" || role == "PAYLOAD") nodeId else null
                val factoryId = if (role == "FACTORY_ADMIN") nodeId else null
                onSave(
                    SupplierOrgMemberUpdateRequest(
                        name = name.trim(),
                        supplierRole = role,
                        assignedWarehouseId = warehouseId,
                        assignedFactoryId = factoryId,
                    ),
                )
            }) { Text("Save") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}
