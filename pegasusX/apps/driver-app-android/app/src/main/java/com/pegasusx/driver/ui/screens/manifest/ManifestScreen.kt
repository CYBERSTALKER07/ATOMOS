package com.pegasusx.driver.ui.screens.manifest

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
import androidx.compose.material3.AssistChip
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
import com.pegasusx.driver.data.model.Order
import com.pegasusx.driver.data.model.OrderState
import androidx.compose.material.icons.filled.Refresh
import com.pegasusx.driver.ui.components.ExplainStatusBanner
import com.pegasusx.driver.ui.components.DriverLoadingState
import com.pegasusx.driver.ui.components.DriverStateKind
import com.pegasusx.driver.ui.components.DriverStatePane
import com.pegasusx.driver.ui.components.PegasusCard
import com.pegasusx.driver.data.remote.ConnectionState
import com.pegasusx.driver.ui.components.StateBadge
import com.pegasusx.driver.ui.components.StaggeredAppear
import com.pegasusx.driver.ui.components.WsConnectionPill
import com.pegasusx.driver.ui.theme.PegasusSpacing
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.formattedAmount
import com.pegasusx.driver.ui.theme.pressable
import androidx.compose.material3.MaterialTheme
import com.pegasusx.driver.ui.screens.manifest.components.EarlyCompleteDialog
import com.pegasusx.driver.ui.screens.manifest.components.ManifestEmptyView
import com.pegasusx.driver.ui.screens.manifest.components.ManifestHeader
import com.pegasusx.driver.ui.screens.manifest.components.ManifestLoadingView
import com.pegasusx.driver.ui.screens.manifest.components.RideCard

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
    val lab = LocalPegasusColors.current
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
            state.isLoading -> ManifestLoadingView()
            state.orders.isEmpty() -> ManifestEmptyView()
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
                            wsConnectionState = state.wsConnectionState,
                            onToggleLoadingMode = { loadingMode = !loadingMode },
                            onRefresh = { viewModel.loadManifest() },
                        )
                    }

                    // LEO: Ghost Stop Prevention banner
                    if (state.awaitingSeal) {
                        item {
                            ExplainStatusBanner(
                                explain = state.gateExplain,
                                fallbackTitle = "AWAITING PAYLOAD SEAL",
                                fallbackDetail = "Manifest is ${state.manifestState ?: "not sealed"}. Payloader must complete loading and seal before you can depart.",
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(horizontal = PegasusSpacing.s16, vertical = 8.dp),
                            )
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
                                            modifier = Modifier.size(48.dp)
                                        ) {
                                            Icon(
                                                Icons.Default.KeyboardArrowUp,
                                                contentDescription = "Move up",
                                                tint = if (index > 0) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.primary.copy(alpha = 0.2f),
                                                modifier = Modifier.size(24.dp)
                                            )
                                        }
                                        IconButton(
                                            onClick = { if (index < displayOrders.lastIndex) viewModel.moveOrder(index, index + 1) },
                                            enabled = index < displayOrders.lastIndex,
                                            modifier = Modifier.size(48.dp)
                                        ) {
                                            Icon(
                                                Icons.Default.KeyboardArrowDown,
                                                contentDescription = "Move down",
                                                tint = if (index < displayOrders.lastIndex) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.primary.copy(alpha = 0.2f),
                                                modifier = Modifier.size(24.dp)
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
        if (hasPendingOrders && !state.isLoading && !state.isRequestingEarlyComplete) {
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
