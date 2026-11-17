package com.pegasusx.retailer.ui.components

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.retailer.ui.theme.PillShape
import com.pegasusx.retailer.ui.theme.SoftSquircleShape
import com.pegasusx.retailer.util.PackMultipleSuggestion

@Composable
fun PackSuggestionBanner(
    suggestions: List<PackMultipleSuggestion>,
    onApply: (skuId: String, suggestedQuantity: Int) -> Unit,
    onDismiss: () -> Unit,
    modifier: Modifier = Modifier,
) {
    if (suggestions.isEmpty()) return

    Surface(
        modifier = modifier.fillMaxWidth(),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.secondaryContainer.copy(alpha = 0.72f),
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.secondary.copy(alpha = 0.35f)),
    ) {
        Column(modifier = Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    "Pack size adjustment required",
                    style = MaterialTheme.typography.labelLarge.copy(fontWeight = FontWeight.Bold),
                    color = MaterialTheme.colorScheme.onSecondaryContainer,
                )
                TextButton(onClick = onDismiss) {
                    Text("Dismiss", style = MaterialTheme.typography.labelSmall)
                }
            }

            suggestions.forEach { suggestion ->
                Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    val title = suggestion.productName ?: suggestion.skuId.ifBlank { "Item" }
                    Text(
                        "$title: pack of ${suggestion.packSize}" +
                            (suggestion.requestedQty?.let { " (requested $it)" } ?: ""),
                        style = MaterialTheme.typography.bodySmall.copy(fontWeight = FontWeight.SemiBold),
                        color = MaterialTheme.colorScheme.onSecondaryContainer,
                    )
                    if (suggestion.message.isNotBlank()) {
                        Text(
                            suggestion.message,
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onSecondaryContainer.copy(alpha = 0.8f),
                        )
                    }
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        PackFixActionChip(
                            label = "Snap ${suggestion.suggestedNearest}",
                            highlighted = true,
                            onClick = { onApply(suggestion.skuId, suggestion.suggestedNearest) },
                        )
                        if (suggestion.suggestedFloor > 0 && suggestion.suggestedFloor != suggestion.suggestedNearest) {
                            PackFixActionChip(
                                label = "Floor ${suggestion.suggestedFloor}",
                                highlighted = false,
                                onClick = { onApply(suggestion.skuId, suggestion.suggestedFloor) },
                            )
                        }
                        if (suggestion.suggestedCeil != suggestion.suggestedNearest) {
                            PackFixActionChip(
                                label = "Ceil ${suggestion.suggestedCeil}",
                                highlighted = false,
                                onClick = { onApply(suggestion.skuId, suggestion.suggestedCeil) },
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
fun LineOptionTogglePill(
    label: String,
    selected: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Surface(
        modifier = modifier,
        onClick = onClick,
        shape = PillShape,
        color = if (selected) {
            MaterialTheme.colorScheme.primaryContainer
        } else {
            MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.38f)
        },
        border = BorderStroke(
            1.dp,
            if (selected) {
                MaterialTheme.colorScheme.primary
            } else {
                MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.5f)
            },
        ),
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.labelSmall.copy(
                fontSize = 10.sp,
                fontWeight = if (selected) FontWeight.Bold else FontWeight.Medium,
            ),
            color = if (selected) {
                MaterialTheme.colorScheme.onPrimaryContainer
            } else {
                MaterialTheme.colorScheme.onSurface.copy(alpha = 0.72f)
            },
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
        )
    }
}

@Composable
private fun PackFixActionChip(
    label: String,
    highlighted: Boolean,
    onClick: () -> Unit,
) {
    Surface(
        onClick = onClick,
        shape = PillShape,
        color = if (highlighted) {
            MaterialTheme.colorScheme.primary
        } else {
            MaterialTheme.colorScheme.surface
        },
        border = BorderStroke(
            1.dp,
            if (highlighted) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.outlineVariant,
        ),
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.Bold),
            color = if (highlighted) {
                MaterialTheme.colorScheme.onPrimary
            } else {
                MaterialTheme.colorScheme.onSurface
            },
            modifier = Modifier.padding(horizontal = 10.dp, vertical = 5.dp),
        )
    }
}
