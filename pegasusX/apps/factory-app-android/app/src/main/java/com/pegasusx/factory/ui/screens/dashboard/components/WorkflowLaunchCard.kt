package com.pegasusx.factory.ui.screens.dashboard.components

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
                        text = "Operator workflows",
                        style = MaterialTheme.typography.titleMedium,
                    )
                    Text(
                        text = "Warehouse demand acknowledgements and live manifest overrides are available on mobile in streamlined native flows.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
            WorkflowLaunchRow(
                title = "Supply requests",
                supporting = "Review warehouse demand and advance requests through production states.",
                actionLabel = "Open requests",
                onClick = { onNavigate(FactoryRoutes.SUPPLY_REQUESTS) },
            )
            WorkflowLaunchRow(
                title = "Payload override",
                supporting = "Move transfers between loading manifests or release them back to approved stock.",
                actionLabel = "Open override",
                onClick = { onNavigate(FactoryRoutes.PAYLOAD_OVERRIDE) },
            )
            WorkflowLaunchRow(
                title = "Manifest lifecycle",
                supporting = "Advance manifests through draft, loading, sealed, dispatched, and completed.",
                actionLabel = "Open manifests",
                onClick = { onNavigate(FactoryRoutes.MANIFESTS) },
            )
            WorkflowLaunchRow(
                title = "Gate exceptions",
                supporting = "Review transfers removed from manifests and DLQ escalations.",
                actionLabel = "Open exceptions",
                onClick = { onNavigate(FactoryRoutes.MANIFEST_EXCEPTIONS) },
            )
            WorkflowLaunchRow(
                title = "Create transfer",
                supporting = "Stage a new factory-to-warehouse movement with volume and optional fleet assignment.",
                actionLabel = "Create transfer",
                onClick = { onNavigate(FactoryRoutes.TRANSFER_CREATE) },
            )
            WorkflowLaunchRow(
                title = "Replenishment insights",
                supporting = "Warehouse stock velocity and reorder pressure linked to this factory.",
                actionLabel = "Open insights",
                onClick = { onNavigate(FactoryRoutes.INSIGHTS) },
            )
            WorkflowLaunchRow(
                title = "Analytics overview",
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
