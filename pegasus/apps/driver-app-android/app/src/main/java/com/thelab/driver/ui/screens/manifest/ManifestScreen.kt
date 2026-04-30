package com.pegasus.driver.ui.screens.manifest

import androidx.compose.foundation.background
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowForward
import androidx.compose.material.icons.filled.CreditCard
import androidx.compose.material.icons.filled.KeyboardArrowDown
import androidx.compose.material.icons.filled.KeyboardArrowUp
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.RadioButton
import androidx.compose.material3.TextButton
import androidx.compose.material3.IconButton
import androidx.compose.material3.Switch
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasus.driver.data.model.Order
import com.pegasus.driver.data.model.OrderState
import com.pegasus.driver.ui.components.LabCard
import com.pegasus.driver.ui.components.StateBadge
import com.pegasus.driver.ui.components.StaggeredAppear
import com.pegasus.driver.ui.theme.LabSpacing
import com.pegasus.driver.ui.theme.LocalLabColors
import com.pegasus.driver.ui.theme.formattedAmount
import com.pegasus.driver.ui.theme.pressable
import androidx.compose.material3.MaterialTheme

/**
 * ManifestScreen — redesigned to match iOS RidesListView.
 * Custom header with monospaced labels, premium ride cards, staggered animations.
 */
@Composable
fun ManifestScreen(
    viewModel: ManifestViewModel,
    onOrderClick: (Order) -> Unit = {},
    onRequestEarlyComplete: (reason: String, note: String) -> Unit = { _, _ -> }
) {
    val state by viewModel.state.collectAsState()
    val lab = LocalLabColors.current
    var loadingMode by remember { mutableStateOf(false) }
    var showEarlyCompleteDialog by remember { mutableStateOf(false) }

    // Early Complete Confirmation Dialog (Edge 27)
    if (showEarlyCompleteDialog) {
        EarlyCompleteDialog(
            onDismiss = { showEarlyCompleteDialog = false },
            onConfirm = { reason, note ->
                showEarlyCompleteDialog = false
                onRequestEarlyComplete(reason, note)
            }
        )
    }

    Box(modifier = Modifier.fillMaxSize()) {

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(lab.bg)
    ) {
        when {
            state.isLoading -> LoadingView()
            state.orders.isEmpty() -> EmptyView()
            else -> {
                val pendingOrders = state.orders.filter {
                    it.state != OrderState.COMPLETED && it.state != OrderState.CANCELLED
                }
                val displayOrders = if (loadingMode) pendingOrders.reversed() else pendingOrders

                LazyColumn(
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(bottom = 100.dp)
                ) {
                    // Header
                    item {
                        ManifestHeader(
                            pendingCount = pendingOrders.size,
                            loadingMode = loadingMode,
                            onToggleLoadingMode = { loadingMode = !loadingMode }
                        )
                    }

                    // LEO: Ghost Stop Prevention banner
                    if (state.awaitingSeal) {
                        item {
                            Box(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(horizontal = LabSpacing.s16, vertical = 8.dp)
                                    .clip(RoundedCornerShape(12.dp))
                                    .background(MaterialTheme.colorScheme.errorContainer)
                                    .padding(16.dp)
                            ) {
                                Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                                    Text(
                                        text = "AWAITING PAYLOAD SEAL",
                                        style = MaterialTheme.typography.labelSmall.copy(
                                            fontWeight = FontWeight.Black,
                                            fontFamily = FontFamily.Monospace,
                                            letterSpacing = 1.sp
                                        ),
                                        color = MaterialTheme.colorScheme.onErrorContainer
                                    )
                                    Text(
                                        text = "Manifest is ${state.manifestState ?: "not sealed"}. Payloader must complete loading and seal before you can depart.",
                                        style = MaterialTheme.typography.bodySmall,
                                        color = MaterialTheme.colorScheme.onErrorContainer
                                    )
                                }
                            }
                        }
                    }

                    // Ride cards
                    itemsIndexed(
                        items = displayOrders,
                        key = { _, order -> order.id }
                    ) { index, order ->
                        val loadSeqLabel = if (loadingMode) {
                            when (index) {
                                0 -> "Load #${index + 1} · Back of Truck"
                                displayOrders.lastIndex -> "Load #${index + 1} · By the Doors"
                                else -> "Load #${index + 1}"
                            }
                        } else null
                        StaggeredAppear(index = index) {
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                if (!loadingMode && displayOrders.size > 1) {
                                    Column {
                                        IconButton(
                                            onClick = { if (index > 0) viewModel.moveOrder(index, index - 1) },
                                            enabled = index > 0,
                                            modifier = Modifier.size(32.dp)
                                        ) {
                                            Icon(
                                                Icons.Default.KeyboardArrowUp,
                                                contentDescription = "Move up",
                                                tint = if (index > 0) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.primary.copy(alpha = 0.2f),
                                                modifier = Modifier.size(20.dp)
                                            )
                                        }
                                        IconButton(
                                            onClick = { if (index < displayOrders.lastIndex) viewModel.moveOrder(index, index + 1) },
                                            enabled = index < displayOrders.lastIndex,
                                            modifier = Modifier.size(32.dp)
                                        ) {
                                            Icon(
                                                Icons.Default.KeyboardArrowDown,
                                                contentDescription = "Move down",
                                                tint = if (index < displayOrders.lastIndex) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.primary.copy(alpha = 0.2f),
                                                modifier = Modifier.size(20.dp)
                                            )
                                        }
                                    }
                                }
                                Box(modifier = Modifier.weight(1f)) {
                                    RideCard(
                                        order = order,
                                        loadSeqLabel = loadSeqLabel,
                                        onClick = { onOrderClick(order) }
                                    )
                                }
                            }
                        }
                    }
                }
            }
        }
    } // end Column

        // Edge 27: Early Complete FAB — only visible when there are pending orders
        val hasPendingOrders = state.orders.any {
            it.state != OrderState.COMPLETED && it.state != OrderState.CANCELLED && it.state != OrderState.QUARANTINE
        }
        if (hasPendingOrders && !state.isLoading) {
            FloatingActionButton(
                onClick = { showEarlyCompleteDialog = true },
                modifier = Modifier
                    .align(Alignment.BottomEnd)
                    .padding(24.dp),
                containerColor = MaterialTheme.colorScheme.errorContainer,
                contentColor = MaterialTheme.colorScheme.onErrorContainer,
            ) {
                Icon(Icons.Default.Warning, contentDescription = "Request Early Complete")
            }
        }
    } // end Box
}

/**
 * Edge 27: EarlyCompleteDialog — driver selects reason and optional note.
 */
@Composable
private fun EarlyCompleteDialog(
    onDismiss: () -> Unit,
    onConfirm: (reason: String, note: String) -> Unit
) {
    val reasons = listOf("FATIGUE", "TRAFFIC", "VEHICLE_ISSUE", "OTHER")
    val labels = listOf("Fatigue / Feeling Unwell", "Heavy Traffic / Road Block", "Vehicle Issue", "Other")
    var selectedReason by remember { mutableStateOf(reasons[0]) }
    var note by remember { mutableStateOf("") }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = {
            Text(
                "Request Early Route Complete",
                style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold)
            )
        },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(
                    "Remaining orders will be returned to the supplier for next-day re-dispatch.",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
                Spacer(modifier = Modifier.height(4.dp))
                reasons.forEachIndexed { i, reason ->
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        RadioButton(
                            selected = selectedReason == reason,
                            onClick = { selectedReason = reason }
                        )
                        Text(labels[i], style = MaterialTheme.typography.bodyMedium)
                    }
                }
                Spacer(modifier = Modifier.height(4.dp))
                OutlinedTextField(
                    value = note,
                    onValueChange = { note = it },
                    label = { Text("Note (optional)") },
                    modifier = Modifier.fillMaxWidth(),
                    maxLines = 2,
                    singleLine = false
                )
            }
        },
        confirmButton = {
            TextButton(onClick = { onConfirm(selectedReason, note) }) {
                Text("Submit Request", color = MaterialTheme.colorScheme.error)
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text("Cancel")
            }
        }
    )
}

@Composable
private fun ManifestHeader(
    pendingCount: Int,
    loadingMode: Boolean,
    onToggleLoadingMode: () -> Unit
) {
    val colorScheme = MaterialTheme.colorScheme
    val typography = MaterialTheme.typography
    Column(
        modifier = Modifier
            .padding(horizontal = LabSpacing.s20)
            .padding(top = 60.dp, bottom = LabSpacing.s20)
    ) {
        Text(
            text = if (loadingMode) "LOADING SEQUENCE" else "UPCOMING",
            style = typography.labelSmall.copy(
                fontWeight = FontWeight.Black,
                fontFamily = FontFamily.Monospace,
                letterSpacing = 1.2.sp,
            ),
            color = if (loadingMode) colorScheme.primary else colorScheme.onSurfaceVariant,
        )
        Spacer(modifier = Modifier.height(6.dp))
        Row(
            modifier = Modifier.fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.SpaceBetween
        ) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(10.dp)
            ) {
                Text(
                    text = if (loadingMode) "Loading Manifest" else "Route Manifest",
                    style = typography.headlineSmall.copy(fontWeight = FontWeight.Bold),
                    color = colorScheme.onSurface,
                )
                if (pendingCount > 0) {
                    Box(
                        modifier = Modifier
                            .size(24.dp)
                            .clip(CircleShape)
                            .background(if (loadingMode) colorScheme.primary else colorScheme.primary),
                        contentAlignment = Alignment.Center
                    ) {
                        Text(
                            text = "$pendingCount",
                            style = typography.labelSmall.copy(fontWeight = FontWeight.Bold),
                            color = colorScheme.onPrimary,
                        )
                    }
                }
            }
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                Text(
                    text = "Loading Mode",
                    style = typography.labelSmall.copy(fontFamily = FontFamily.Monospace),
                    color = if (loadingMode) colorScheme.primary else colorScheme.onSurfaceVariant,
                )
                Switch(
                    checked = loadingMode,
                    onCheckedChange = { onToggleLoadingMode() }
                )
            }
        }
    }
}

@Composable
private fun RideCard(order: Order, loadSeqLabel: String? = null, onClick: () -> Unit) {
    val lab = LocalLabColors.current
    val isDark = isSystemInDarkTheme()
    val colorScheme = MaterialTheme.colorScheme

    LabCard(
        modifier = Modifier
            .padding(horizontal = LabSpacing.s16, vertical = 7.dp)
            .pressable(onClick = onClick)
    ) {
        Column(
            modifier = Modifier.padding(LabSpacing.s20),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // Loading sequence badge
            if (loadSeqLabel != null) {
                Box(
                    modifier = Modifier
                        .clip(RoundedCornerShape(4.dp))
                        .background(colorScheme.primaryContainer)
                        .padding(horizontal = 10.dp, vertical = 4.dp)
                ) {
                    Text(
                        text = loadSeqLabel,
                        fontSize = 10.sp,
                        fontWeight = FontWeight.Bold,
                        fontFamily = FontFamily.Monospace,
                        color = colorScheme.onPrimaryContainer,
                    )
                }
            }

            // Top: order ID + status
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = order.id,
                    fontSize = 15.sp,
                    fontWeight = FontWeight.Bold,
                    fontFamily = FontFamily.Monospace,
                    color = lab.fg
                )
                StateBadge(state = order.state)
            }

            // Info chips
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                InfoChip(icon = Icons.Default.CreditCard, text = order.retailerName)
                InfoChip(icon = Icons.Default.LocalShipping, text = order.totalAmount.formattedAmount())
            }

            // Delivery target
            Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                Text(
                    text = "DELIVERY TARGET",
                    fontSize = 9.sp,
                    fontWeight = FontWeight.Bold,
                    fontFamily = FontFamily.Monospace,
                    color = lab.fgTertiary
                )
                if (order.latitude != null && order.longitude != null) {
                    Text(
                        text = String.format("%.4f, %.4f", order.latitude, order.longitude),
                        fontSize = 14.sp,
                        fontWeight = FontWeight.SemiBold,
                        fontFamily = FontFamily.Monospace,
                        color = lab.fg
                    )
                } else {
                    Text(
                        text = order.deliveryAddress,
                        fontSize = 14.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = lab.fg
                    )
                }
            }

            // Bottom row: items count + arrow
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = "${order.items.size} items",
                    fontSize = 12.sp,
                    fontWeight = FontWeight.Bold,
                    fontFamily = FontFamily.Monospace,
                    color = lab.fgSecondary
                )
                Box(
                    modifier = Modifier
                        .size(32.dp)
                        .clip(CircleShape)
                        .background(lab.fg.copy(alpha = 0.08f)),
                    contentAlignment = Alignment.Center
                ) {
                    Icon(
                        imageVector = Icons.AutoMirrored.Filled.ArrowForward,
                        contentDescription = null,
                        tint = lab.fg,
                        modifier = Modifier.size(12.dp)
                    )
                }
            }

            // Bottom accent bar
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(2.dp)
                    .clip(RoundedCornerShape(2.dp))
                    .background(lab.fg.copy(alpha = 0.12f))
            )
        }
    }
}

@Composable
private fun InfoChip(
    icon: androidx.compose.ui.graphics.vector.ImageVector,
    text: String
) {
    val colorScheme = MaterialTheme.colorScheme
    Row(
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(6.dp),
        modifier = Modifier
            .clip(RoundedCornerShape(50))
            .background(colorScheme.surfaceContainerHighest)
            .padding(horizontal = 10.dp, vertical = 6.dp)
    ) {
        Icon(
            imageVector = icon,
            contentDescription = null,
            tint = colorScheme.onSurfaceVariant,
            modifier = Modifier.size(10.dp)
        )
        Text(
            text = text,
            style = MaterialTheme.typography.labelMedium,
            color = colorScheme.onSurfaceVariant,
        )
    }
}

@Composable
private fun LoadingView() {
    val colorScheme = MaterialTheme.colorScheme
    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(colorScheme.surface),
        contentAlignment = Alignment.Center
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            CircularProgressIndicator(color = colorScheme.primary)
            Text(
                text = "Loading routes...",
                style = MaterialTheme.typography.bodyMedium,
                color = colorScheme.onSurfaceVariant,
            )
        }
    }
}

@Composable
private fun EmptyView() {
    val colorScheme = MaterialTheme.colorScheme
    val typography = MaterialTheme.typography
    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(colorScheme.surface),
        contentAlignment = Alignment.Center
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            Icon(
                imageVector = Icons.Default.LocalShipping,
                contentDescription = null,
                tint = colorScheme.onSurfaceVariant,
                modifier = Modifier.size(40.dp)
            )
            Text(
                text = "No upcoming rides",
                style = typography.titleMedium.copy(fontWeight = FontWeight.Bold),
                color = colorScheme.onSurfaceVariant,
            )
            Text(
                text = "Pull to refresh or check back later",
                style = typography.bodyMedium,
                color = colorScheme.onSurfaceVariant,
            )
        }
    }
}
