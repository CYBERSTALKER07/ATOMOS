package com.pegasus.payload.ui.home

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.ui.draw.clip
import androidx.compose.foundation.background
import com.pegasus.design.PegasusStatePane
import com.pegasus.design.PegasusStateKind
import com.pegasus.payload.data.model.Truck
import com.pegasus.payload.ui.components.ExplainStatusBanner
import com.pegasus.payload.ui.components.ManifestKpiGrid
import com.pegasus.payload.R

@Composable
fun ManifestDetailPane(
    truck: Truck?,
    state: HomeUiState,
    onRefresh: () -> Unit,
    onStartLoading: () -> Unit,
    onSelectOrder: (String) -> Unit,
    onToggleItem: (String) -> Unit,
    onSealOrder: () -> Unit,
    onDismissCountdown: () -> Unit,
    onReportMissingItems: (String) -> Unit,
    reportingMissingItems: Boolean,
    canSealOrder: (String) -> Boolean,
    allOrdersSealed: Boolean,
    onSealManifest: () -> Unit,
    onStartNewManifest: () -> Unit,
    onShowInject: () -> Unit,
    onShowException: (String) -> Unit,
    onShowReDispatch: (String) -> Unit,
    onClearEscalated: () -> Unit,
    onScanProduct: () -> Unit,
) {
    Surface(
        color = MaterialTheme.colorScheme.surfaceContainerLow,
        modifier = Modifier.fillMaxSize(),
    ) {
        Box(
            modifier = Modifier.fillMaxSize(),
            contentAlignment = Alignment.TopCenter
        ) {
            Column(
                modifier = Modifier
                    .widthIn(max = 840.dp)
                    .fillMaxSize()
                    .padding(24.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp),
            ) {
            if (truck == null) {
                PegasusStatePane(
                    kind = PegasusStateKind.Empty,
                    headline = "Select a vehicle",
                    body = "Pick a truck from the sidebar to load its manifest.",
                )
                return@Column
            }
            val inboundLat = state.manifest?.driverLat
            val inboundLng = state.manifest?.driverLng
            if (inboundLat != null && inboundLng != null) {
                Text(
                    text = "Inbound truck ${if (state.manifest?.liveLocationAvailable == true) "LIVE" else "last known"}: " +
                        "%.5f, %.5f".format(inboundLat, inboundLng),
                    style = MaterialTheme.typography.labelMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            DetailHeader(
                truck = truck,
                onRefresh = onRefresh,
                showInject = state.manifest?.state == "LOADING",
                onShowInject = onShowInject,
            )

            state.escalatedMessage?.let { msg ->
                EscalatedBanner(message = msg, onDismiss = onClearEscalated)
            }

            state.barcodeScanMessage?.let { msg ->
                com.pegasus.design.PegasusRuntimeBanner(
                    tone = if (msg.contains("error", ignoreCase = true) || msg.contains("failed", ignoreCase = true) || msg.contains("not", ignoreCase = true)) com.pegasus.design.PegasusRuntimeTone.Warning else com.pegasus.design.PegasusRuntimeTone.Live,
                    message = msg,
                    onRetry = null,
                )
            }

            // All Sealed success — terminal state, supersedes everything else.
            if (state.manifestSealed) {
                AllSealedSuccessCard(
                    dispatchCodes = state.dispatchCodes,
                    onStartNewManifest = onStartNewManifest,
                )
                return@Column
            }

            when {
                state.loadingManifest -> com.pegasus.design.PegasusLoadingState(
                    title = stringResource(R.string.mobile_payload_ui_loading_manifest),
                    body = "Syncing orders and volume for this vehicle.",
                )
                state.manifest == null -> PegasusStatePane(
                    kind = PegasusStateKind.Empty,
                    headline = "No open manifest",
                    body = "This vehicle doesn't have an active loading manifest.",
                )
                else -> {
                    ManifestKpiGrid(manifest = state.manifest)

                    if (state.error != null || state.errorExplain != null) {
                        ExplainStatusBanner(
                            explain = state.errorExplain,
                            fallbackTitle = state.error,
                            modifier = Modifier.fillMaxWidth(),
                        )
                    }

                    val phase = state.manifest.state
                    if (phase == "DRAFT") {
                        StartLoadingButton(
                            loading = state.startingLoading,
                            onClick = onStartLoading,
                        )
                        Text(
                            "Tap Start Loading to open the manifest for tap-check and per-order seal.",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    } else if (phase == "LOADING" || phase == "SEALED") {
                        // 60-second post-seal countdown takes the spotlight.
                        if (state.postSealOrderId != null) {
                            PostSealCountdownCard(
                                orderId = state.postSealOrderId,
                                dispatchCode = state.dispatchCodes[state.postSealOrderId].orEmpty(),
                                secondsLeft = state.postSealCountdown,
                                reportingMissingItems = reportingMissingItems,
                                onDismiss = onDismissCountdown,
                                onReportMissingItems = { onReportMissingItems(state.postSealOrderId) },
                            )
                        }

                        OrderChecklist(
                            orders = state.orders,
                            loading = state.loadingOrders,
                            selectedOrderId = state.selectedOrderId,
                            checkedItems = state.checkedItems,
                            sealedOrderIds = state.sealedOrderIds,
                            dispatchCodes = state.dispatchCodes,
                            sealingOrderId = state.sealingOrderId,
                            exceptionLoadingOrderId = state.exceptionLoadingOrderId,
                            onSelectOrder = onSelectOrder,
                            onToggleItem = onToggleItem,
                            onSealOrder = onSealOrder,
                            canSealSelected = state.selectedOrderId?.let { canSealOrder(it) } ?: false,
                            onShowException = onShowException,
                            onShowReDispatch = onShowReDispatch,
                            onScanProduct = onScanProduct,
                        )

                        if (allOrdersSealed && phase != "SEALED") {
                            SealManifestButton(
                                loading = state.sealingManifest,
                                onClick = onSealManifest,
                            )
                        }
                    }
                }
            }
        }
    }
}
}

@Composable
fun DetailHeader(
    truck: Truck,
    onRefresh: () -> Unit,
    showInject: Boolean,
    onShowInject: () -> Unit,
) {
    val displayLabel = truck.label.ifBlank { truck.licensePlate.ifBlank { "TRK-${truck.id.take(8)}" } }.uppercase()
    Row(
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text(
                displayLabel,
                style = MaterialTheme.typography.headlineMedium,
                fontFamily = androidx.compose.ui.text.font.FontFamily.Monospace,
                fontWeight = androidx.compose.ui.text.font.FontWeight.Black,
                letterSpacing = 1.2.sp,
            )
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                if (truck.licensePlate.isNotBlank()) {
                    Text(
                        truck.licensePlate.uppercase(),
                        style = MaterialTheme.typography.bodySmall,
                        fontFamily = androidx.compose.ui.text.font.FontFamily.Monospace,
                        fontWeight = androidx.compose.ui.text.font.FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                if (truck.vehicleClass.isNotBlank()) {
                    Text("—", color = MaterialTheme.colorScheme.outlineVariant)
                    Text(
                        truck.vehicleClass.uppercase(),
                        style = MaterialTheme.typography.bodySmall,
                        fontFamily = androidx.compose.ui.text.font.FontFamily.Monospace,
                        fontWeight = androidx.compose.ui.text.font.FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                Surface(
                    color = MaterialTheme.colorScheme.tertiary.copy(alpha = 0.12f),
                    shape = RoundedCornerShape(50),
                ) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(4.dp),
                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 3.dp),
                    ) {
                        Box(
                            modifier = Modifier
                                .size(6.dp)
                                .clip(RoundedCornerShape(50))
                                .background(MaterialTheme.colorScheme.tertiary)
                        )
                        Text(
                            "ACTIVE",
                            style = MaterialTheme.typography.labelSmall,
                            fontFamily = androidx.compose.ui.text.font.FontFamily.Monospace,
                            fontWeight = androidx.compose.ui.text.font.FontWeight.Black,
                            color = MaterialTheme.colorScheme.tertiary,
                        )
                    }
                }
            }
        }
        Spacer(Modifier.width(12.dp))
        if (showInject) {
            IconButton(onClick = onShowInject) {
                Icon(Icons.Filled.Add, contentDescription = stringResource(R.string.mobile_payload_ui_inject_order))
            }
        }
        IconButton(onClick = onRefresh) {
            Icon(Icons.Filled.Refresh, contentDescription = stringResource(R.string.mobile_payload_ui_refresh_manifest))
        }
    }
}

@Composable
fun StartLoadingButton(loading: Boolean, onClick: () -> Unit) {
    Button(
        onClick = onClick,
        enabled = !loading,
        shape = RoundedCornerShape(16.dp),
        modifier = Modifier.fillMaxWidth().height(52.dp),
    ) {
        if (loading) {
            CircularProgressIndicator(
                modifier = Modifier.size(20.dp),
                strokeWidth = 2.dp,
                color = MaterialTheme.colorScheme.onPrimary,
            )
        } else {
            Text(
                "START LOADING",
                fontFamily = androidx.compose.ui.text.font.FontFamily.Monospace,
                fontWeight = androidx.compose.ui.text.font.FontWeight.Black,
                letterSpacing = 0.5.sp,
            )
        }
    }
}

@Composable
fun SealManifestButton(loading: Boolean, onClick: () -> Unit) {
    Button(
        onClick = onClick,
        enabled = !loading,
        shape = RoundedCornerShape(16.dp),
        modifier = Modifier.fillMaxWidth().height(52.dp),
        colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.tertiary),
    ) {
        Icon(Icons.Filled.Lock, contentDescription = null)
        Spacer(Modifier.size(8.dp))
        if (loading) {
            CircularProgressIndicator(
                modifier = Modifier.size(20.dp),
                strokeWidth = 2.dp,
                color = MaterialTheme.colorScheme.onTertiary,
            )
        } else {
            Text(
                "SEAL MANIFEST",
                fontFamily = androidx.compose.ui.text.font.FontFamily.Monospace,
                fontWeight = androidx.compose.ui.text.font.FontWeight.Black,
                letterSpacing = 0.5.sp,
            )
        }
    }
}
