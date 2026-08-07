package com.pegasusx.supplier.ui.screens.network

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.*
import androidx.compose.ui.text.input.KeyboardType
import com.pegasusx.supplier.data.remote.GeocodeApi
import com.pegasusx.supplier.ui.components.AddressLocationField
import com.pegasusx.supplier.ui.components.AddressLocationValue
import com.pegasusx.supplier.ui.theme.PegasusSpacing

@Composable
fun AddWarehouseDialog(
    geocodeApi: GeocodeApi,
    onDismiss: () -> Unit,
    onSave: (String, AddressLocationValue, Double) -> Unit,
) {
    val (defaultLat, defaultLng) = defaultWarehouseCoordinates()
    var name by remember { mutableStateOf("") }
    var location by remember {
        mutableStateOf(AddressLocationValue(lat = defaultLat, lng = defaultLng))
    }
    var radius by remember { mutableStateOf("50") }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Add warehouse") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                OutlinedTextField(value = name, onValueChange = { name = it }, label = { Text("Name") }, singleLine = true)
                AddressLocationField(
                    geocodeApi = geocodeApi,
                    value = location,
                    onValueChange = { location = it },
                    label = stringResource(R.string.supplier_portal_residual_text_warehouse_address),
                )
                OutlinedTextField(value = radius, onValueChange = { radius = it }, label = { Text("Coverage km") }, keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal), singleLine = true)
            }
        },
        confirmButton = {
            TextButton(
                onClick = {
                    if (name.isNotBlank() && location.address.isNotBlank() && location.lat != 0.0 && location.lng != 0.0) {
                        onSave(name, location, radius.toDoubleOrNull() ?: 50.0)
                    }
                },
                enabled = name.isNotBlank() && location.address.isNotBlank(),
            ) { Text("Save") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}
