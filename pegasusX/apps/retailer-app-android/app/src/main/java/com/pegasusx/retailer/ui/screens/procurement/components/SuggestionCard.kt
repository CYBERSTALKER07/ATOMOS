package com.pegasusx.retailer.ui.screens.procurement.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.retailer.data.model.DemandForecast
import com.pegasusx.retailer.ui.screens.procurement.ProcurementUiState
import com.pegasusx.retailer.ui.theme.SoftSquircleShape
import com.pegasusx.retailer.R

@Composable
fun SuggestionCard(
    forecast: DemandForecast,
    isSelected: Boolean,
    quantity: Int,
    onToggle: () -> Unit,
    onDecrement: () -> Unit,
    onIncrement: () -> Unit,
) {
    val borderModifier = if (isSelected) {
        Modifier.border(2.dp, MaterialTheme.colorScheme.primary.copy(alpha = 0.4f), SoftSquircleShape)
    } else {
        Modifier
    }

    Surface(
        modifier = Modifier
            .fillMaxWidth()
            .then(borderModifier),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Checkbox(checked = isSelected, onCheckedChange = { onToggle() })
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        forecast.productName,
                        style = MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.SemiBold),
                        maxLines = 1,
                    )
                    Text(
                        stringResource(R.string.mobile_retailer_ui_confidence_confidencepercent, forecast.confidencePercent),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
                    )
                }
                if (isSelected) {
                    QuantityStepper(quantity = quantity, onDecrement = onDecrement, onIncrement = onIncrement)
                } else {
                    Text(
                        stringResource(R.string.mobile_retailer_ui_predictedquantity_units, forecast.predictedQuantity),
                        style = MaterialTheme.typography.labelMedium.copy(fontWeight = FontWeight.Bold),
                        color = MaterialTheme.colorScheme.primary,
                    )
                }
            }
            Spacer(modifier = Modifier.height(8.dp))
            Text(
                forecast.reasoning,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.55f),
                maxLines = 2,
            )
        }
    }
}
