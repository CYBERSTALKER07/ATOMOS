package com.pegasusx.factory.ui.screens.supply.components

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.factory.data.model.SupplyRequest
import com.pegasusx.factory.ui.theme.PegasusSpacing

@Composable
fun SupplyBoard(
    requests: List<SupplyRequest>,
    transitioningId: String?,
    qcById: Map<String, String> = emptyMap(),
    onAction: (SupplyRequest, String) -> Unit,
    onQC: (SupplyRequest, String) -> Unit = { _, _ -> },
) {
    val lanes = boardLanes
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .horizontalScroll(rememberScrollState()),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
    ) {
        lanes.forEach { lane ->
            Column(
                modifier = Modifier.width(220.dp),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                Text(lane.replace('_', ' '), style = MaterialTheme.typography.labelLarge)
                requests.filter { it.state == lane }.forEach { request ->
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(PegasusSpacing.md), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                            Text(request.warehouseId.take(8), style = MaterialTheme.typography.titleSmall)
                            Text(request.priority, style = MaterialTheme.typography.bodySmall)
                            qcById[request.id]?.takeIf { it.isNotBlank() }?.let {
                                Text("QC $it", style = MaterialTheme.typography.labelSmall)
                            }
                            actionsForState(request.state).forEach { spec ->
                                FilledTonalButton(
                                    onClick = { onAction(request, spec.action) },
                                    enabled = transitioningId != request.id,
                                    modifier = Modifier.fillMaxWidth(),
                                ) { Text(spec.label) }
                            }
                            FilledTonalButton(
                                onClick = { onQC(request, "PASS") },
                                enabled = transitioningId != request.id,
                                modifier = Modifier.fillMaxWidth(),
                            ) { Text("PASS") }
                            FilledTonalButton(
                                onClick = { onQC(request, "FAIL") },
                                enabled = transitioningId != request.id,
                                modifier = Modifier.fillMaxWidth(),
                            ) { Text("FAIL") }
                        }
                    }
                }
            }
        }
    }
}
