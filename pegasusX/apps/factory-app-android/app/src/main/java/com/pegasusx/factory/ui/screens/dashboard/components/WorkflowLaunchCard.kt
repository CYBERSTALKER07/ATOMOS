package com.pegasusx.factory.ui.screens.dashboard.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Computer
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.factory.ui.navigation.FactoryRoutes
import com.pegasusx.factory.ui.theme.PegasusSpacing

@Composable
fun WorkflowLaunchCard(
    onNavigate: (String) -> Unit,
) {
    ElevatedCard(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.elevatedCardColors(
            containerColor = MaterialTheme.colorScheme.surfaceContainer,
        ),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Row(
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Surface(
                    shape = MaterialTheme.shapes.small,
                    color = MaterialTheme.colorScheme.tertiaryContainer,
                ) {
                    Icon(
                        imageVector = Icons.Default.Computer,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.onTertiaryContainer,
                        modifier = Modifier
                            .padding(PegasusSpacing.sm)
                            .size(20.dp),
                    )
                }
                Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                    Text(
                        text = stringResource(R.string.mobile_factory_ui_operator_workflows),
                        style = MaterialTheme.typography.titleMedium,
                    )
                    Text(
                        text = stringResource(R.string.mobile_factory_ui_warehouse_demand_acknowledgements_and_live_manifest_overrides_ar),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
            WorkflowLaunchRow(
                title = stringResource(R.string.factory_portal_supply_requests_text_supply_requests),
                supporting = "Review warehouse demand and advance requests through production states.",
                actionLabel = "Open requests",
                onClick = { onNavigate(FactoryRoutes.SUPPLY_REQUESTS) },
            )
            WorkflowLaunchRow(
                title = stringResource(R.string.factory_portal_payload_override_text_payload_override),
                supporting = "Move transfers between loading manifests or release them back to approved stock.",
                actionLabel = "Open override",
                onClick = { onNavigate(FactoryRoutes.PAYLOAD_OVERRIDE) },
            )
            WorkflowLaunchRow(
                title = stringResource(R.string.mobile_factory_ui_manifest_lifecycle),
                supporting = "Advance manifests through draft, loading, sealed, dispatched, and completed.",
                actionLabel = "Open manifests",
                onClick = { onNavigate(FactoryRoutes.MANIFESTS) },
            )
            WorkflowLaunchRow(
                title = stringResource(R.string.factory_portal_manifest_exceptions_text_gate_exceptions),
                supporting = "Review transfers removed from manifests and DLQ escalations.",
                actionLabel = "Open exceptions",
                onClick = { onNavigate(FactoryRoutes.MANIFEST_EXCEPTIONS) },
            )
            WorkflowLaunchRow(
                title = stringResource(R.string.factory_portal_transfers_create_text_create_transfer),
                supporting = "Stage a new factory-to-warehouse movement with volume and optional fleet assignment.",
                actionLabel = "Create transfer",
                onClick = { onNavigate(FactoryRoutes.TRANSFER_CREATE) },
            )
            WorkflowLaunchRow(
                title = stringResource(R.string.factory_portal_insights_text_replenishment_insights),
                supporting = "Warehouse stock velocity and reorder pressure linked to this factory.",
                actionLabel = "Open insights",
                onClick = { onNavigate(FactoryRoutes.INSIGHTS) },
            )
            WorkflowLaunchRow(
                title = stringResource(R.string.factory_portal_analytics_text_analytics_overview),
                supporting = "Factory throughput, active manifests, exception queue, and lead time.",
                actionLabel = "Open analytics",
                onClick = { onNavigate(FactoryRoutes.ANALYTICS) },
            )
        }
    }
}

@Composable
fun WorkflowLaunchRow(
    title: String,
    supporting: String,
    actionLabel: String,
    onClick: () -> Unit,
) {
    Surface(
        modifier = Modifier.fillMaxWidth(),
        shape = MaterialTheme.shapes.medium,
        color = MaterialTheme.colorScheme.surfaceContainerHigh,
    ) {
        Row(
            modifier = Modifier.padding(PegasusSpacing.md),
            horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(
                modifier = Modifier.weight(1f),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
            ) {
                Text(
                    text = title,
                    style = MaterialTheme.typography.titleSmall,
                )
                Text(
                    text = supporting,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            FilledTonalButton(onClick = onClick) {
                Text(actionLabel)
            }
        }
    }
}
