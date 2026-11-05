package com.pegasusx.supplier.ui.components

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
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.AssistChip
import androidx.compose.material3.AssistChipDefaults
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedCard
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import java.text.NumberFormat
import java.util.Locale

fun formatMinorAmount(minor: Long, currency: String, locale: Locale = Locale.getDefault()): String {
    val fmt = NumberFormat.getInstance(locale)
    val major = minor / 100.0
    return if (currency.isBlank()) fmt.format(major) else "${fmt.format(major)} $currency"
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SupplierKpiTile(
    label: String,
    value: String,
    icon: ImageVector,
    modifier: Modifier = Modifier,
    supporting: String? = null,
    onClick: (() -> Unit)? = null,
) {
    val cardModifier = modifier.fillMaxWidth()
    val shape = RoundedCornerShape(16.dp)
    val border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.5f))

    if (onClick != null) {
        OutlinedCard(onClick = onClick, modifier = cardModifier, shape = shape, border = border) {
            SupplierKpiContent(icon = icon, value = value, label = label, supporting = supporting)
        }
    } else {
        OutlinedCard(modifier = cardModifier, shape = shape, border = border) {
            SupplierKpiContent(icon = icon, value = value, label = label, supporting = supporting)
        }
    }
}

@Composable
private fun SupplierKpiContent(
    icon: ImageVector,
    value: String,
    label: String,
    supporting: String?,
) {
    Column(
        modifier = Modifier.padding(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        Surface(
            shape = RoundedCornerShape(12.dp),
            color = MaterialTheme.colorScheme.secondaryContainer,
            modifier = Modifier.size(44.dp),
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
fun SupplierMetricTile(
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
fun SupplierStatusChip(
    status: String,
    modifier: Modifier = Modifier,
) {
    val normalized = status.trim().ifBlank { "—" }.uppercase(Locale.US)
    val colors = when {
        normalized in setOf("CANCELLED", "FAILED", "ERROR", "REJECTED") ->
            AssistChipDefaults.assistChipColors(
                disabledContainerColor = MaterialTheme.colorScheme.errorContainer,
                disabledLabelColor = MaterialTheme.colorScheme.onErrorContainer,
            )
        normalized in setOf("COMPLETED", "SEALED", "DELIVERED", "FULFILLED", "ACTIVE") ->
            AssistChipDefaults.assistChipColors(
                disabledContainerColor = MaterialTheme.colorScheme.tertiaryContainer,
                disabledLabelColor = MaterialTheme.colorScheme.onTertiaryContainer,
            )
        normalized in setOf("LOADING", "IN_TRANSIT", "PENDING", "DRAFT", "AWAITING_PAYMENT") ->
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
                text = normalized,
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
fun SupplierOpsListCard(
    headline: String,
    supporting: String,
    modifier: Modifier = Modifier,
    status: String? = null,
    onClick: (() -> Unit)? = null,
    onReassign: (() -> Unit)? = null,
) {
    val cardModifier = modifier.fillMaxWidth()
    val shape = RoundedCornerShape(16.dp)
    val border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.4f))

    if (onClick != null) {
        OutlinedCard(onClick = onClick, modifier = cardModifier, shape = shape, border = border) {
            SupplierOpsListContent(
                headline = headline,
                supporting = supporting,
                status = status,
                onReassign = onReassign,
            )
        }
    } else {
        OutlinedCard(modifier = cardModifier, shape = shape, border = border) {
            SupplierOpsListContent(
                headline = headline,
                supporting = supporting,
                status = status,
                onReassign = onReassign,
            )
        }
    }
}

@Composable
private fun SupplierOpsListContent(
    headline: String,
    supporting: String,
    status: String?,
    onReassign: (() -> Unit)?,
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
        if (!status.isNullOrBlank()) {
            SupplierStatusChip(status = status)
        }
    }
    if (onReassign != null) {
        HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.5f))
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = PegasusSpacing.md, vertical = PegasusSpacing.sm),
            horizontalArrangement = Arrangement.End,
            verticalAlignment = Alignment.CenterVertically
        ) {
            TextButton(onClick = onReassign) {
                Text("Reassign")
            }
        }
    }
}

@Composable
fun SupplierSectionTitle(
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
fun SupplierLeadingIcon(
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
