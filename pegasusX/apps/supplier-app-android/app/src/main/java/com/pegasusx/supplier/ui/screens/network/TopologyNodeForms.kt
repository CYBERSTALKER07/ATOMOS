package com.pegasusx.supplier.ui.screens.network

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.ui.theme.PegasusSpacing

data class WarehouseDraft(
    val key: String,
    val warehouseId: String?,
    val name: String,
    val lat: String,
    val lng: String,
    val coverageRadiusKm: String,
    val isActive: Boolean,
    val isOnShift: Boolean,
    val transferMode: String,
)

data class FactoryDraft(
    val key: String,
    val factoryId: String?,
    val name: String,
    val lat: String,
    val lng: String,
    val isActive: Boolean,
)

@Composable
fun DraftField(label: String, value: String, onValueChange: (String) -> Unit) {
    OutlinedTextField(
        value = value,
        onValueChange = onValueChange,
        label = { Text(label) },
        modifier = Modifier.fillMaxWidth(),
    )
}

@Composable
fun SectionLabel(title: String) {
    Text(title, style = MaterialTheme.typography.titleSmall, color = MaterialTheme.colorScheme.primary)
}

@Composable
fun NodeCard(name: String, lat: Double, lng: Double, meta: String) {
    ElevatedCard(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(PegasusSpacing.lg)) {
            Text(name.ifEmpty { "Unnamed node" }, style = MaterialTheme.typography.titleMedium)
            Text(
                "%.4f, %.4f".format(lat, lng),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.outline,
            )
            Text(meta, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.outline)
        }
    }
}
