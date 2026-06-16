package com.pegasusx.supplier.ui.screens.catalog

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import com.pegasus.barcode.EanBarcode
import com.pegasus.barcode.EanBarcodeScannerPreview
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

@Composable
fun CatalogBarcodeField(
    value: String,
    onValueChange: (String) -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
) {
    var showScanner by rememberSaveable { mutableStateOf(false) }
    var scannerEnabled by rememberSaveable { mutableStateOf(true) }
    var validationError by rememberSaveable { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
        OutlinedTextField(
            value = value,
            onValueChange = {
                validationError = null
                onValueChange(it)
            },
            label = { Text("EAN / GTIN barcode") },
            placeholder = { Text("8–14 digits or scan label") },
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            singleLine = true,
            isError = validationError != null,
            supportingText = validationError?.let { { Text(it) } },
            modifier = Modifier.fillMaxWidth(),
            enabled = enabled,
        )
        TextButton(
            onClick = { showScanner = !showScanner },
            enabled = enabled,
        ) {
            Text(if (showScanner) "Hide camera" else "Scan barcode")
        }
        if (showScanner && enabled) {
            EanBarcodeScannerPreview(
                enabled = scannerEnabled,
                onBarcode = { scanned ->
                    val normalized = EanBarcode.normalize(scanned)
                    if (normalized == null) {
                        validationError = "Invalid EAN/GTIN — check the label"
                        return@EanBarcodeScannerPreview
                    }
                    validationError = null
                    onValueChange(normalized)
                    scannerEnabled = false
                    scope.launch {
                        delay(1500)
                        scannerEnabled = true
                    }
                },
            )
        }
        if (value.isNotBlank()) {
            val normalized = EanBarcode.normalize(value)
            Text(
                if (normalized != null) "Valid GTIN: $normalized" else "Enter a valid EAN/GTIN checksum",
                style = MaterialTheme.typography.labelSmall,
                color = if (normalized != null) {
                    MaterialTheme.colorScheme.tertiary
                } else {
                    MaterialTheme.colorScheme.error
                },
            )
        }
    }
}
