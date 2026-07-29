package com.pegasusx.warehouse.ui.screens.operations

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.RetailerOverridePreview
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import com.pegasusx.warehouse.ui.screens.components.WarehouseSectionTitle

@Composable
fun OperationsPricingPreview(
    productId: String,
    onProductIdChange: (String) -> Unit,
    retailerId: String,
    onRetailerIdChange: (String) -> Unit,
    proposedPrice: String,
    onProposedPriceChange: (String) -> Unit,
    previewLoading: Boolean,
    preview: RetailerOverridePreview?
) {
    Column {
        WarehouseSectionTitle("Pricing impact preview (read-only)")
        Text(
            "Preview how a proposed retailer price would compare to catalog list price. Does not create overrides.",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        OutlinedTextField(
            value = productId,
            onValueChange = onProductIdChange,
            label = { Text("Product / SKU ID") },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true,
        )
        OutlinedTextField(
            value = retailerId,
            onValueChange = onRetailerIdChange,
            label = { Text("Retailer ID (optional)") },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true,
        )
        OutlinedTextField(
            value = proposedPrice,
            onValueChange = onProposedPriceChange,
            label = { Text("Proposed price (minor units)") },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
        )
        if (previewLoading) {
            Text("Loading preview…", style = MaterialTheme.typography.bodySmall)
        }
        preview?.let { p ->
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = PegasusSpacing.sm),
                verticalArrangement = Arrangement.spacedBy(4.dp),
            ) {
                Text("Retailers on SKU: ${p.retailersOnSkuCount}")
                Text("Active overrides: ${p.activeOverrideCount}")
                Text("Catalog list price: ${p.catalogListPrice}")
                Text("Margin delta / unit: ${p.marginDeltaPerUnit}")
                Text(
                    p.marginEstimateLabel,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                if (p.readOnly == true) {
                    Text(
                        "Read-only — contact supplier to apply overrides.",
                        style = MaterialTheme.typography.bodyMedium,
                    )
                }
            }
        }
    }
}
