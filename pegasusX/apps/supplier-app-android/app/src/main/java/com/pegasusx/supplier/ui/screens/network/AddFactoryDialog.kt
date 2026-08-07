package com.pegasusx.supplier.ui.screens.network

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.*
import com.pegasusx.supplier.data.remote.GeocodeApi
import com.pegasusx.supplier.ui.components.AddressLocationField
import com.pegasusx.supplier.ui.components.AddressLocationValue
import com.pegasusx.supplier.ui.theme.PegasusSpacing

@Composable
fun AddFactoryDialog(
    geocodeApi: GeocodeApi,
    onDismiss: () -> Unit,
    onSave: (String, AddressLocationValue) -> Unit,
) {
    var name by remember { mutableStateOf("") }
    var location by remember { mutableStateOf(AddressLocationValue(lat = 41.3111, lng = 69.2797)) }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Add factory") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                OutlinedTextField(value = name, onValueChange = { name = it }, label = { Text("Name") }, singleLine = true)
                AddressLocationField(
                    geocodeApi = geocodeApi,
                    value = location,
                    onValueChange = { location = it },
                    label = stringResource(R.string.factory_portal_residual_text_factory_address),
                )
            }
        },
        confirmButton = {
            TextButton(
                onClick = {
                    if (name.isNotBlank() && location.address.isNotBlank() && location.lat != 0.0 && location.lng != 0.0) {
                        onSave(name, location)
                    }
                },
                enabled = name.isNotBlank() && location.address.isNotBlank(),
            ) { Text("Save") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}
