package com.pegasusx.warehouse.ui.screens.operations

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.pegasusx.warehouse.data.model.BroadcastTemplate
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import com.pegasusx.warehouse.ui.components.WarehouseSectionTitle

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OperationsBroadcastForm(
    templates: List<BroadcastTemplate>,
    templateDate: String,
    onTemplateDateChange: (String) -> Unit,
    customReason: String,
    onCustomReasonChange: (String) -> Unit,
    title: String,
    onTitleChange: (String) -> Unit,
    broadcastRole: String,
    onBroadcastRoleChange: (String) -> Unit,
    body: String,
    onBodyChange: (String) -> Unit,
    saveAsTemplate: Boolean,
    onSaveAsTemplateChange: (Boolean) -> Unit,
    broadcasting: Boolean,
    savingTemplate: Boolean,
    onSelectTemplate: (BroadcastTemplate) -> Unit,
    onDeleteTemplate: (BroadcastTemplate) -> Unit,
    onBroadcast: () -> Unit
) {
    val broadcastRoles = listOf("DRIVER", "RETAILER", "ALL")
    var roleExpanded by remember { mutableStateOf(false) }

    Column {
        WarehouseSectionTitle("Broadcast templates")
        Text(
            "Built-in depot starters plus your saved custom messages.",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .horizontalScroll(rememberScrollState()),
            horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
        ) {
            templates.forEach { template ->
                Row(verticalAlignment = Alignment.CenterVertically) {
                    FilterChip(
                        selected = false,
                        onClick = { onSelectTemplate(template) },
                        label = {
                            val suffix = if (template.source == "custom") " · saved" else ""
                            Text(template.title + suffix, maxLines = 1)
                        },
                    )
                    if (template.source == "custom") {
                        IconButton(onClick = { onDeleteTemplate(template) }) {
                            Icon(Icons.Default.Close, contentDescription = stringResource(R.string.mobile_warehouse_ui_delete_title, template.title))
                        }
                    }
                }
            }
        }

        HorizontalDivider()
        WarehouseSectionTitle("Send depot broadcast")
        OutlinedTextField(
            value = templateDate,
            onValueChange = onTemplateDateChange,
            label = { Text("Effective date (optional)") },
            placeholder = { Text("2026-07-01") },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true,
        )
        OutlinedTextField(
            value = customReason,
            onValueChange = onCustomReasonChange,
            label = { Text("Custom reason (optional)") },
            placeholder = { Text("Bay 2 closed") },
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
        ExposedDropdownMenuBox(expanded = roleExpanded, onExpandedChange = { roleExpanded = it }) {
            OutlinedTextField(
                value = broadcastRole,
                onValueChange = {},
                readOnly = true,
                label = { Text("Target role") },
                trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = roleExpanded) },
                modifier = Modifier
                    .menuAnchor()
                    .fillMaxWidth(),
            )
            ExposedDropdownMenu(expanded = roleExpanded, onDismissRequest = { roleExpanded = false }) {
                broadcastRoles.forEach { role ->
                    androidx.compose.material3.DropdownMenuItem(
                        text = { Text(role) },
                        onClick = {
                            onBroadcastRoleChange(role)
                            roleExpanded = false
                        },
                    )
                }
            }
        }
        OutlinedTextField(
            value = body,
            onValueChange = onBodyChange,
            label = { Text("Message") },
            modifier = Modifier.fillMaxWidth(),
            minLines = 4,
        )
        Row(verticalAlignment = Alignment.CenterVertically) {
            Checkbox(checked = saveAsTemplate, onCheckedChange = onSaveAsTemplateChange)
            Text("Save as custom template for this depot")
        }
        Button(
            onClick = onBroadcast,
            enabled = !broadcasting && !savingTemplate,
            modifier = Modifier.fillMaxWidth(),
        ) {
            Text(if (broadcasting || savingTemplate) "Sending…" else "Send broadcast")
        }
    }
}
