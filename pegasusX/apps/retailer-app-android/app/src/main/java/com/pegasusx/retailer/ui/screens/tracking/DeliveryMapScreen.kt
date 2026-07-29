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
import com.pegasusx.retailer.ui.screens.tracking.components.OrderInfoCard
import com.pegasusx.retailer.ui.screens.tracking.components.RecentReceiptsStrip
import com.pegasusx.retailer.ui.screens.tracking.components.FiscalReceiptQROverlay
import com.pegasusx.retailer.ui.screens.tracking.components.TrackingMap
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

        TrackingMap(
            isLoading = uiState.isLoading,
            visibleOrders = visibleOrders,
            hasLocationPermission = hasLocationPermission,
            cameraPositionState = cameraPositionState,
            selectedOrder = selectedOrder,
            onOrderSelected = { selectedOrder = it },
            activeDeliveryCount = uiState.activeDeliveryCount,
            emptyStateMessage = uiState.emptyStateMessage,
            onRefresh = viewModel::refresh
        )
    }
}


