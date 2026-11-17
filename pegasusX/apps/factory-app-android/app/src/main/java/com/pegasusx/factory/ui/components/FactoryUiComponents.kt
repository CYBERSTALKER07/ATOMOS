package com.pegasusx.factory.ui.components

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.AssistChip
import androidx.compose.material3.AssistChipDefaults
import androidx.compose.material3.Badge
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedCard
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.factory.ui.theme.PegasusSpacing
import java.util.Locale

enum class FactoryKpiBadge {
    Alert,
    Done,
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FactoryKpiTile(
    label: String,
    value: String,
    icon: ImageVector,
    modifier: Modifier = Modifier,
    supporting: String? = null,
    badge: FactoryKpiBadge? = null,
    onClick: (() -> Unit)? = null,
) {
    val cardModifier = modifier.fillMaxWidth()
    val shape = RoundedCornerShape(16.dp)
    val border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.5f))

    if (onClick != null) {
        OutlinedCard(onClick = onClick, modifier = cardModifier, shape = shape, border = border) {
            FactoryKpiContent(icon = icon, badge = badge, value = value, label = label, supporting = supporting)
        }
    } else {
        OutlinedCard(modifier = cardModifier, shape = shape, border = border) {
            FactoryKpiContent(icon = icon, badge = badge, value = value, label = label, supporting = supporting)
        }
    }
}

@Composable
private fun FactoryKpiContent(
    icon: ImageVector,
    badge: FactoryKpiBadge?,
    value: String,
    label: String,
    supporting: String?,
) {
    Column(
        modifier = Modifier.padding(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            FactoryLeadingIcon(icon = icon)
            badge?.let { FactoryKpiBadgeChip(badge = it) }
        }
        Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
            Text(
                text = value,
                style = MaterialTheme.typography.headlineMedium.copy(
                    fontFamily = FontFamily.Monospace,
                    fontWeight = FontWeight.Black,
                ),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            Text(
                text = label.uppercase(),
                style = MaterialTheme.typography.labelSmall.copy(
                    fontFamily = FontFamily.Monospace,
                    fontWeight = FontWeight.Bold,
                    letterSpacing = 1.1.sp,
                ),
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            if (!supporting.isNullOrBlank()) {
                Text(
                    text = supporting,
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

@Composable
private fun FactoryKpiBadgeChip(badge: FactoryKpiBadge) {
    val (label, colors) = when (badge) {
        FactoryKpiBadge.Alert -> "ALERT" to AssistChipDefaults.assistChipColors(
            disabledContainerColor = MaterialTheme.colorScheme.errorContainer,
            disabledLabelColor = MaterialTheme.colorScheme.onErrorContainer,
        )
        FactoryKpiBadge.Done -> "DONE" to AssistChipDefaults.assistChipColors(
            disabledContainerColor = MaterialTheme.colorScheme.primaryContainer,
            disabledLabelColor = MaterialTheme.colorScheme.onPrimaryContainer,
        )
    }
    AssistChip(
        onClick = {},
        enabled = false,
        label = { Text(label, style = MaterialTheme.typography.labelSmall) },
        colors = colors,
    )
}

@Composable
fun FactoryMetricTile(
    label: String,
    value: String,
    modifier: Modifier = Modifier,
) {
    OutlinedCard(
        modifier = modifier,
        shape = RoundedCornerShape(14.dp),
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.4f)),
    ) {
        Column(Modifier.padding(PegasusSpacing.md)) {
            Text(
                text = label.uppercase(),
                style = MaterialTheme.typography.labelSmall.copy(
                    fontFamily = FontFamily.Monospace,
                    fontWeight = FontWeight.Bold,
                    letterSpacing = 1.sp,
                ),
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Spacer(Modifier.height(PegasusSpacing.xs))
            Text(
                text = value,
                style = MaterialTheme.typography.titleLarge.copy(
                    fontFamily = FontFamily.Monospace,
                    fontWeight = FontWeight.Black,
                ),
            )
        }
    }
}

@Composable
fun FactoryStatusChip(
    status: String,
    modifier: Modifier = Modifier,
) {
    val normalized = status.trim().ifBlank { "—" }.uppercase(Locale.US)
    val colors = when {
        normalized in setOf("CANCELLED", "FAILED", "ERROR", "REJECTED", "CRITICAL", "UNAVAILABLE") ->
            AssistChipDefaults.assistChipColors(
                disabledContainerColor = MaterialTheme.colorScheme.errorContainer,
                disabledLabelColor = MaterialTheme.colorScheme.onErrorContainer,
            )
        normalized in setOf(
            "COMPLETED", "SEALED", "DELIVERED", "FULFILLED", "ACTIVE", "AVAILABLE",
            "OPEN", "RECEIVED", "ARRIVED", "ON_SHIFT", "ON SHIFT",
        ) ->
            AssistChipDefaults.assistChipColors(
                disabledContainerColor = MaterialTheme.colorScheme.tertiaryContainer,
                disabledLabelColor = MaterialTheme.colorScheme.onTertiaryContainer,
            )
        normalized in setOf(
            "LOADING", "IN_TRANSIT", "IN TRANSIT", "PENDING", "DRAFT", "APPROVED",
            "DISPATCHED", "STANDARD", "HIGH", "URGENT",
        ) ->
            AssistChipDefaults.assistChipColors(
                disabledContainerColor = MaterialTheme.colorScheme.primaryContainer,
                disabledLabelColor = MaterialTheme.colorScheme.onPrimaryContainer,
            )
        else ->
            AssistChipDefaults.assistChipColors(
                disabledContainerColor = MaterialTheme.colorScheme.surfaceContainerHigh,
                disabledLabelColor = MaterialTheme.colorScheme.onSurfaceVariant,
            )
    }

    AssistChip(
        onClick = {},
        enabled = false,
        modifier = modifier,
        label = {
            Text(
                text = normalized.replace('_', ' '),
                style = MaterialTheme.typography.labelSmall,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        },
        colors = colors,
    )
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FactoryOpsListCard(
    headline: String,
    supporting: String,
    modifier: Modifier = Modifier,
    status: String? = null,
    secondaryStatus: String? = null,
    onClick: (() -> Unit)? = null,
) {
    val cardModifier = modifier.fillMaxWidth()
    val shape = RoundedCornerShape(16.dp)
    val border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.4f))

    if (onClick != null) {
        OutlinedCard(onClick = onClick, modifier = cardModifier, shape = shape, border = border) {
            FactoryOpsListContent(headline = headline, supporting = supporting, status = status, secondaryStatus = secondaryStatus)
        }
    } else {
        OutlinedCard(modifier = cardModifier, shape = shape, border = border) {
            FactoryOpsListContent(headline = headline, supporting = supporting, status = status, secondaryStatus = secondaryStatus)
        }
    }
}

@Composable
private fun FactoryOpsListContent(
    headline: String,
    supporting: String,
    status: String?,
    secondaryStatus: String?,
) {
    Row(
        modifier = Modifier.padding(PegasusSpacing.lg),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        Column(
            modifier = Modifier.weight(1f),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
        ) {
            Text(
                text = headline,
                style = MaterialTheme.typography.titleSmall,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            Text(
                text = supporting,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
        }
        Column(
            horizontalAlignment = Alignment.End,
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
        ) {
            if (!status.isNullOrBlank()) {
                FactoryStatusChip(status = status)
            }
            if (!secondaryStatus.isNullOrBlank()) {
                FactoryStatusChip(status = secondaryStatus)
            }
        }
    }
}

@Composable
fun FactorySectionTitle(
    title: String,
    modifier: Modifier = Modifier,
) {
    Text(
        text = title.uppercase(),
        style = MaterialTheme.typography.labelMedium.copy(
            fontFamily = FontFamily.Monospace,
            fontWeight = FontWeight.Black,
            letterSpacing = 1.2.sp,
        ),
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        modifier = modifier,
    )
}

@Composable
fun FactorySectionHeader(
    title: String,
    count: Int,
    modifier: Modifier = Modifier,
) {
    Row(
        modifier = modifier.padding(top = PegasusSpacing.sm),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        FactorySectionTitle(title = title)
        Spacer(Modifier.width(PegasusSpacing.sm))
        Badge { Text("$count") }
    }
}

@Composable
fun FactoryLeadingIcon(
    icon: ImageVector,
    modifier: Modifier = Modifier,
) {
    Surface(
        modifier = modifier.size(44.dp),
        shape = RoundedCornerShape(12.dp),
        color = MaterialTheme.colorScheme.secondaryContainer,
    ) {
        Box(contentAlignment = Alignment.Center) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onSecondaryContainer,
                modifier = Modifier.size(24.dp),
            )
        }
    }
}
