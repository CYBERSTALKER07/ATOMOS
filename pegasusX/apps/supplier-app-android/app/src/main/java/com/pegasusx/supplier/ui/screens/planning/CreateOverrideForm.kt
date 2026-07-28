package com.pegasusx.supplier.ui.screens.planning

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SeasonalTemplatesResponse

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CreateOverrideForm(
    data: SeasonalTemplatesResponse?,
    templateId: String,
    name: String,
    startDate: String,
    endDate: String,
    formError: String?,
    saving: Boolean,
    onTemplateIdChange: (String) -> Unit,
    onNameChange: (String) -> Unit,
    onStartDateChange: (String) -> Unit,
    onEndDateChange: (String) -> Unit,
    onSubmit: () -> Unit,
) {
    Column {
        if ((data?.builtinTemplates?.size ?: 0) > 0) {
            var expanded by remember { mutableStateOf(false) }
            ExposedDropdownMenuBox(expanded = expanded, onExpandedChange = { expanded = it }) {
                OutlinedTextField(
                    value = templateId.ifBlank { "Custom" },
                    onValueChange = {},
                    readOnly = true,
                    label = { Text("Template") },
                    trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = expanded) },
                    modifier = Modifier.menuAnchor().fillMaxWidth(),
                )
                ExposedDropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
                    DropdownMenuItem(
                        text = { Text("Custom") },
                        onClick = { onTemplateIdChange(""); expanded = false },
                    )
                    data?.builtinTemplates?.forEach { template ->
                        DropdownMenuItem(
                            text = { Text(template.name) },
                            onClick = { onTemplateIdChange(template.id); expanded = false },
                        )
                    }
                }
            }
        }
        OutlinedTextField(value = name, onValueChange = onNameChange, label = { Text("Name (optional)") }, modifier = Modifier.fillMaxWidth())
        OutlinedTextField(value = startDate, onValueChange = onStartDateChange, label = { Text("Start (YYYY-MM-DD)") }, modifier = Modifier.fillMaxWidth())
        OutlinedTextField(value = endDate, onValueChange = onEndDateChange, label = { Text("End (YYYY-MM-DD)") }, modifier = Modifier.fillMaxWidth())
        formError?.let { Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall) }
        Button(
            enabled = !saving,
            onClick = onSubmit,
            modifier = Modifier.fillMaxWidth(),
        ) {
            Text(if (saving) "Saving…" else "Create override")
        }
    }
}
