package com.pegasusx.factory.ui.screens.supply.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.pegasusx.factory.data.model.SupplyRequest
import com.pegasusx.factory.ui.theme.PegasusSpacing

@Composable
fun SupplyRequestCard(
    request: SupplyRequest,
    transitioning: Boolean,
    onAction: (String) -> Unit,
) {
    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                verticalAlignment = Alignment.Top,
            ) {
                Column(
                    modifier = Modifier.weight(1f),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
                ) {
                    Text(
                        text = requestLabel(request),
                        style = MaterialTheme.typography.titleMedium,
                    )
                    Text(
                        text = stringResource(R.string.mobile_factory_ui_request_take, request.id.take(8)),
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                Column(
                    horizontalAlignment = Alignment.End,
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
                ) {
                    RequestTag(
                        text = request.state,
                        containerColor = MaterialTheme.colorScheme.secondaryContainer,
                        contentColor = MaterialTheme.colorScheme.onSecondaryContainer,
                    )
                    RequestTag(
                        text = request.priority.ifBlank { "NORMAL" },
                        containerColor = MaterialTheme.colorScheme.surfaceContainerHighest,
                        contentColor = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                SupplyMetric("Volume", "${trimDecimal(request.totalVolumeVU)} VU", Modifier.weight(1f))
                SupplyMetric("Created", formatDate(request.createdAt), Modifier.weight(1f))
                SupplyMetric("Delivery", formatDate(request.requestedDeliveryDate), Modifier.weight(1f))
            }

            if (request.notes.isNotBlank()) {
                Surface(
                    modifier = Modifier.fillMaxWidth(),
                    shape = MaterialTheme.shapes.medium,
                    color = MaterialTheme.colorScheme.surfaceContainerLowest,
                ) {
                    Text(
                        text = request.notes,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(PegasusSpacing.md),
                    )
                }
            }

            val actions = actionsForState(request.state)
            if (actions.isEmpty()) {
                Text(
                    text = stringResource(R.string.mobile_factory_ui_no_manual_action_is_available_for_the_current_state),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            } else {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                ) {
                    actions.forEach { action ->
                        val buttonModifier = Modifier.weight(1f)
                        if (action.emphasized) {
                            FilledTonalButton(
                                onClick = { onAction(action.action) },
                                enabled = !transitioning,
                                modifier = buttonModifier,
                            ) {
                                Text(if (transitioning) "Working…" else action.label)
                            }
                        } else {
                            Button(
                                onClick = { onAction(action.action) },
                                enabled = !transitioning,
                                modifier = buttonModifier,
                            ) {
                                Text(if (transitioning) "Working…" else action.label)
                            }
                        }
                    }
                }
            }
        }
    }
}
