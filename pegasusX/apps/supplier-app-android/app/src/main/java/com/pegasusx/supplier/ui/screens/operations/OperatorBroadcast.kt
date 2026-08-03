package com.pegasusx.supplier.ui.screens.operations

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SUPPLIER_BROADCAST_TEMPLATES
import com.pegasusx.supplier.ui.components.SupplierSectionTitle
import com.pegasusx.supplier.ui.theme.PegasusSpacing

private val broadcastRoles = listOf("ALL", "DRIVER", "RETAILER", "PAYLOAD")

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OperatorBroadcast(
    title: String,
    body: String,
    broadcastRole: String,
    templateDate: String,
    broadcasting: Boolean,
    onTitleChange: (String) -> Unit,
    onBodyChange: (String) -> Unit,
    onBroadcastRoleChange: (String) -> Unit,
    onTemplateDateChange: (String) -> Unit,
    onBroadcast: () -> Unit,
) {
    var roleExpanded by remember { mutableStateOf(false) }

    Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md)) {
        SupplierSectionTitle("Operator broadcast")
        Row(Modifier.horizontalScroll(rememberScrollState()), horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
            SUPPLIER_BROADCAST_TEMPLATES.forEach { template ->
                FilterChip(
                    selected = false,
                    onClick = {
                        val dateLabel = templateDate.trim().ifBlank { "the selected date" }
                        onTitleChange(template.title)
                        onBodyChange(template.body.replace("{date}", dateLabel))
                        if (broadcastRoles.contains(template.defaultRole)) {
                            onBroadcastRoleChange(template.defaultRole)
                        }
                    },
                    label = { Text(template.title, maxLines = 1) },
                )
            }
        }
        OutlinedTextField(
            value = templateDate,
            onValueChange = onTemplateDateChange,
            label = { Text("Template date (optional)") },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true,
        )
        OutlinedTextField(
            value = title,
            onValueChange = onTitleChange,
            label = { Text("Title") },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true,
        )
        OutlinedTextField(
            value = body,
            onValueChange = onBodyChange,
            label = { Text("Message") },
            modifier = Modifier.fillMaxWidth(),
            minLines = 3,
        )
        ExposedDropdownMenuBox(expanded = roleExpanded, onExpandedChange = { roleExpanded = it }) {
            OutlinedTextField(
                value = broadcastRole,
                onValueChange = {},
                readOnly = true,
                label = { Text("Target role") },
                trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = roleExpanded) },
                modifier = Modifier.menuAnchor().fillMaxWidth(),
            )
            ExposedDropdownMenu(expanded = roleExpanded, onDismissRequest = { roleExpanded = false }) {
                broadcastRoles.forEach { role ->
                    DropdownMenuItem(
                        text = { Text(role) },
                        onClick = {
                            onBroadcastRoleChange(role)
                            roleExpanded = false
                        },
                    )
                }
            }
        }
        Button(onClick = onBroadcast, enabled = !broadcasting && title.isNotBlank() && body.isNotBlank(), modifier = Modifier.fillMaxWidth()) {
            Text(if (broadcasting) "Sending…" else "Send broadcast")
        }
    }
}
