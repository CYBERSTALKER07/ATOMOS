package com.pegasusx.retailer.ui.screens.tracking

import android.Manifest
import android.content.pm.PackageManager
import android.graphics.Color as AndroidColor
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import android.graphics.Bitmap
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material.icons.filled.MyLocation
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FilterChipDefaults
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.FloatingActionButtonDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.SmallFloatingActionButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.core.content.ContextCompat
import com.google.android.gms.maps.CameraUpdateFactory
import com.google.android.gms.maps.model.BitmapDescriptor
import com.google.android.gms.maps.model.BitmapDescriptorFactory
import com.google.android.gms.maps.model.CameraPosition
import com.google.android.gms.maps.model.LatLng
import com.google.android.gms.maps.model.LatLngBounds
import com.google.maps.android.compose.GoogleMap
import com.google.maps.android.compose.MapProperties
import com.google.maps.android.compose.MapUiSettings
import com.google.maps.android.compose.Marker
import com.google.maps.android.compose.MarkerState
import com.google.maps.android.compose.rememberCameraPositionState
import com.google.zxing.BarcodeFormat
import com.google.zxing.qrcode.QRCodeWriter
import com.pegasusx.retailer.data.model.TrackingOrder
import com.pegasusx.retailer.ui.components.RetailerListCard
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusRuntimeBanner
import com.pegasus.design.PegasusRuntimeTone
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DeliveryMapScreen(
    viewModel: DeliveryTrackingViewModel,
    onBack: () -> Unit,
    embedded: Boolean = false,
    modifier: Modifier = Modifier,
) {
    val uiState by viewModel.state.collectAsState()
    val context = LocalContext.current
    val scope = rememberCoroutineScope()

    var hasLocationPermission by remember {
        mutableStateOf(
            ContextCompat.checkSelfPermission(context, Manifest.permission.ACCESS_FINE_LOCATION) ==
                    PackageManager.PERMISSION_GRANTED
        )
    }

    val permissionLauncher = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestPermission()
    ) { granted -> hasLocationPermission = granted }

    LaunchedEffect(Unit) {
        if (!hasLocationPermission) {
            permissionLauncher.launch(Manifest.permission.ACCESS_FINE_LOCATION)
        }
    }

    val visibleOrders = uiState.visibleOrders
    var selectedOrder by remember { mutableStateOf<TrackingOrder?>(null) }

    // Default center: Tashkent
    val defaultPosition = LatLng(41.2995, 69.2401)
    val cameraPositionState = rememberCameraPositionState {
        position = CameraPosition.fromLatLngZoom(defaultPosition, 12f)
    }

    // Fit camera to driver markers when data loads
    LaunchedEffect(visibleOrders) {
        val driverPoints = visibleOrders.mapNotNull { order ->
            val lat = order.driverLatitude ?: return@mapNotNull null
            val lng = order.driverLongitude ?: return@mapNotNull null
            LatLng(lat, lng)
        }
        if (driverPoints.isNotEmpty()) {
            val boundsBuilder = LatLngBounds.builder()
            driverPoints.forEach { boundsBuilder.include(it) }
            try {
                cameraPositionState.animate(
                    CameraUpdateFactory.newLatLngBounds(boundsBuilder.build(), 100),
                    durationMs = 600
                )
            } catch (_: Exception) {
                cameraPositionState.animate(
                    CameraUpdateFactory.newLatLngZoom(driverPoints.first(), 14f)
                )
            }
        }
    }

    Column(modifier = modifier.fillMaxSize()) {
        if (!embedded) {
            TopAppBar(
                title = { Text("Delivery Tracking") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
                actions = {
                    IconButton(onClick = viewModel::refresh) {
                        Icon(Icons.Default.Refresh, contentDescription = "Refresh")
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.surface,
                ),
            )
        }

        val syncMessage = when {
            uiState.loadIssue != null -> uiState.error ?: uiState.emptyStateMessage
            uiState.isLoading && uiState.orders.isNotEmpty() -> "Syncing live delivery positions..."
            else -> null
        }

        if (syncMessage != null) {
            val loadIssue = uiState.loadIssue
            val tone = when (loadIssue) {
                TrackingLoadIssue.OFFLINE -> PegasusRuntimeTone.Offline
                TrackingLoadIssue.RESTRICTED, TrackingLoadIssue.ERROR -> PegasusRuntimeTone.Warning
                null -> PegasusRuntimeTone.Refreshing
            }
            PegasusRuntimeBanner(
                tone = tone,
                message = syncMessage,
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
                onRetry = loadIssue?.let { { viewModel.refresh() } },
            )
        }

        // Supplier filter chips
        if (uiState.suppliers.size > 1) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .horizontalScroll(rememberScrollState())
                    .padding(horizontal = 16.dp, vertical = 8.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                for (supplier in uiState.suppliers) {
                    val isSelected = supplier.supplierId in uiState.selectedSupplierIds
                    FilterChip(
                        selected = isSelected,
                        onClick = { viewModel.toggleSupplier(supplier.supplierId) },
                        label = { Text(supplier.supplierName, maxLines = 1, overflow = TextOverflow.Ellipsis) },
                        colors = FilterChipDefaults.filterChipColors(
                            selectedContainerColor = MaterialTheme.colorScheme.primaryContainer,
                            selectedLabelColor = MaterialTheme.colorScheme.onPrimaryContainer,
                        ),
                    )
                }
            }
        }

        if (uiState.recentReceipts.isNotEmpty()) {
            RecentReceiptsStrip(
                receipts = uiState.recentReceipts.take(6),
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
            )
        }

        // Map
        Box(modifier = Modifier.fillMaxSize()) {
            if (uiState.isLoading && uiState.orders.isEmpty()) {
                PegasusLoadingState(
                    title = "Loading deliveries",
                    body = "Fetching live driver positions and inbound orders…",
                )
            } else {
                GoogleMap(
                    modifier = Modifier.fillMaxSize(),
                    cameraPositionState = cameraPositionState,
                    properties = MapProperties(isMyLocationEnabled = hasLocationPermission),
                    uiSettings = MapUiSettings(
                        zoomControlsEnabled = false,
                        myLocationButtonEnabled = false,
                        mapToolbarEnabled = false,
                    ),
                    onMapClick = { selectedOrder = null },
                ) {
                    for (order in visibleOrders) {
                        val driverLat = order.driverLatitude ?: continue
                        val driverLng = order.driverLongitude ?: continue
                        val position = LatLng(driverLat, driverLng)

                        val isGreen = order.isApproaching || order.state == "ARRIVED"
                        val markerColor = if (isGreen) BitmapDescriptorFactory.HUE_GREEN
                        else BitmapDescriptorFactory.HUE_VIOLET // Dark marker for active orders

                        Marker(
                            state = MarkerState(position = position),
                            title = order.supplierName,
                            snippet = "${order.state} — ${order.items.size} item${if (order.items.size != 1) "s" else ""}",
                            icon = BitmapDescriptorFactory.defaultMarker(markerColor),
                            onClick = {
                                selectedOrder = order
                                true
                            },
                        )
                    }
                }

                // My location FAB
                if (hasLocationPermission) {
                    SmallFloatingActionButton(
                        onClick = {
                            scope.launch {
                                // Camera animates to current location via built-in padding
                                cameraPositionState.animate(
                                    CameraUpdateFactory.zoomTo(14f),
                                    durationMs = 300
                                )
                            }
                        },
                        modifier = Modifier
                            .align(Alignment.BottomEnd)
                            .padding(end = 16.dp, bottom = if (selectedOrder != null) 200.dp else 16.dp),
                        containerColor = MaterialTheme.colorScheme.surface,
                        contentColor = MaterialTheme.colorScheme.onSurface,
                        elevation = FloatingActionButtonDefaults.elevation(defaultElevation = 2.dp),
                    ) {
                        Icon(Icons.Default.MyLocation, contentDescription = "My location", modifier = Modifier.size(20.dp))
                    }
                }

                // Active count badge
                if (uiState.activeDeliveryCount > 0) {
                    Box(
                        modifier = Modifier
                            .align(Alignment.TopStart)
                            .padding(16.dp)
                            .background(MaterialTheme.colorScheme.primaryContainer, MaterialTheme.shapes.small)
                            .padding(horizontal = 12.dp, vertical = 6.dp),
                    ) {
                        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                            Icon(Icons.Default.LocalShipping, contentDescription = null, modifier = Modifier.size(16.dp), tint = MaterialTheme.colorScheme.onPrimaryContainer)
                            Text(
                                "${uiState.activeDeliveryCount} active",
                                style = MaterialTheme.typography.labelMedium,
                                color = MaterialTheme.colorScheme.onPrimaryContainer,
                            )
                        }
                    }
                }

                // Selected order info card
                androidx.compose.animation.AnimatedVisibility(
                    visible = selectedOrder != null,
                    modifier = Modifier.align(Alignment.BottomCenter),
                    enter = slideInVertically(initialOffsetY = { it }) + fadeIn(),
                    exit = slideOutVertically(targetOffsetY = { it }) + fadeOut(),
                ) {
                    selectedOrder?.let { order ->
                        OrderInfoCard(order = order)
                    }
                }

                // Empty state
                if (visibleOrders.isEmpty() && !uiState.isLoading) {
                    PegasusStatePane(
                        kind = PegasusStateKind.Empty,
                        headline = "No active deliveries",
                        body = uiState.emptyStateMessage,
                        actionLabel = "Refresh",
                        onAction = viewModel::refresh,
                    )
                }
            }
        }
    }
}

@Composable
private fun OrderInfoCard(order: TrackingOrder) {
    val statusLabel = buildString {
        append(order.state.replace('_', ' '))
        if (order.liveLocationAvailable) append(" • Live GPS")
    }
    RetailerListCard(
        headline = order.supplierName.ifEmpty { "Unknown Supplier" },
        supporting = buildString {
            append(order.items.joinToString { "${it.productName} ×${it.quantity}" }.ifEmpty { "No items" })
            append(" · ")
            append(formatAmount(order.totalAmount))
        },
        status = statusLabel,
        modifier = Modifier.padding(16.dp),
    )
}

private fun formatAmount(amount: Long): String {
    return "%,d".format(amount)
}

@Composable
private fun RecentReceiptsStrip(
    receipts: List<TrackingOrder>,
    modifier: Modifier = Modifier,
) {
    var fiscalQrOrder by remember { mutableStateOf<TrackingOrder?>(null) }

    Column(modifier = modifier.fillMaxWidth()) {
        Text(
            "Recent receipts",
            style = MaterialTheme.typography.titleSmall,
            fontWeight = FontWeight.SemiBold,
        )
        Text(
            "Completed deliveries from the tracking feed. Tap for fiscal QR when available.",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(top = 2.dp, bottom = 8.dp),
        )
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .heightIn(max = 160.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            receipts.forEach { receipt ->
                val hasFiscalQr = receipt.fiscalQr.isNotBlank()
                RetailerListCard(
                    headline = receipt.supplierName.ifBlank { "Supplier" },
                    supporting = buildString {
                        append("#${receipt.orderId.takeLast(8)}")
                        append(" · ")
                        append(receipt.fiscalReceiptLabel)
                        append(" · ")
                        append(formatAmount(receipt.totalAmount))
                        if (hasFiscalQr) append(" · Fiscal QR")
                    },
                    modifier = Modifier
                        .fillMaxWidth()
                        .then(
                            if (hasFiscalQr) {
                                Modifier.clickable { fiscalQrOrder = receipt }
                            } else {
                                Modifier
                            },
                        ),
                )
            }
        }
    }

    FiscalReceiptQROverlay(
        order = fiscalQrOrder,
        onDismiss = { fiscalQrOrder = null },
    )
}

@Composable
private fun FiscalReceiptQROverlay(
    order: TrackingOrder?,
    onDismiss: () -> Unit,
) {
    if (order == null || order.fiscalQr.isBlank()) return
    Dialog(onDismissRequest = onDismiss) {
        Column(
            modifier = Modifier
                .background(MaterialTheme.colorScheme.surface, RoundedCornerShape(24.dp))
                .padding(24.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(
                "Fiscal receipt",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.Bold,
            )
            Text(
                order.fiscalReceiptLabel,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                textAlign = TextAlign.Center,
            )
            val bitmap = remember(order.fiscalQr) {
                runCatching {
                    val size = 512
                    val matrix = QRCodeWriter().encode(order.fiscalQr, BarcodeFormat.QR_CODE, size, size)
                    Bitmap.createBitmap(size, size, Bitmap.Config.ARGB_8888).also { bmp ->
                        for (x in 0 until size) {
                            for (y in 0 until size) {
                                bmp.setPixel(
                                    x,
                                    y,
                                    if (matrix[x, y]) AndroidColor.BLACK else AndroidColor.WHITE,
                                )
                            }
                        }
                    }
                }.getOrNull()
            }
            if (bitmap != null) {
                Image(
                    bitmap = bitmap.asImageBitmap(),
                    contentDescription = "Fiscal QR",
                    modifier = Modifier.size(220.dp),
                )
            } else {
                Text(
                    order.fiscalQr,
                    style = MaterialTheme.typography.bodySmall,
                    textAlign = TextAlign.Center,
                )
            }
            if (order.latestFiscalReceiptId.isNotBlank()) {
                Text(
                    "ID · ${order.latestFiscalReceiptId}",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            TextButton(onClick = onDismiss) {
                Text("Close")
            }
        }
    }
}
