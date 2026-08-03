package com.pegasusx.supplier.ui.screens.operations

import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.ui.components.SupplierSectionTitle
import com.pegasusx.supplier.ui.theme.PegasusSpacing

@Composable
fun ReplenishmentAction(
    replenishing: Boolean,
    onOpenReplenishmentPolicies: () -> Unit,
    onTriggerReplenishment: () -> Unit,
) {
    Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md)) {
        HorizontalDivider()
        SupplierSectionTitle("Replenishment")
        Text(
            "Opens a warehouse supply request against your primary active warehouse.",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        OutlinedButton(onClick = onOpenReplenishmentPolicies, modifier = Modifier.fillMaxWidth()) {
            Text("View replenishment policies")
        }
        Button(onClick = onTriggerReplenishment, enabled = !replenishing, modifier = Modifier.fillMaxWidth()) {
            Text(if (replenishing) "Triggering…" else "Trigger replenishment")
        }
    }
}
