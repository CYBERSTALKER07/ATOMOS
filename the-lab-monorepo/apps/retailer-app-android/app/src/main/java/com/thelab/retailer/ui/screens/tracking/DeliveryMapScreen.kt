package com.thelab.retailer.ui.screens.tracking

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
import androidx.compose.foundation.background
import androidx.compose.foundation.horizontalScroll
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
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material.icons.filled.MyLocation
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
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
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
import com.thelab.retailer.data.model.TrackingOrder
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DeliveryMapScreen(
    viewModel: DeliveryTrackingViewModel,
    onBack: () -> Unit,
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

    Column(modifier = Modifier.fillMaxSize()) {
        // Top bar
        TopAppBar(
            title = { Text("Delivery Tracking") },
            navigationIcon = {
                IconButton(onClick = onBack) {
                    Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                }
            },
            colors = TopAppBarDefaults.topAppBarColors(
                containerColor = MaterialTheme.colorScheme.surface,
            ),
        )

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

        // Map
        Box(modifier = Modifier.fillMaxSize()) {
            if (uiState.isLoading && uiState.orders.isEmpty()) {
                Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator()
                }
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
                if (visibleOrders.isNotEmpty()) {
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
                                "${visibleOrders.size} active",
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
                    Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                        Text(
                            text = "No active deliveries with driver location",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun OrderInfoCard(order: TrackingOrder) {
    Column(
        modifier = Modifier
            .padding(16.dp)
            .fillMaxWidth()
            .background(
                MaterialTheme.colorScheme.surfaceContainerHigh,
                MaterialTheme.shapes.large,
            )
            .padding(16.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Box(
                modifier = Modifier
                    .size(8.dp)
                    .background(
                        if (order.isApproaching || order.state == "ARRIVED") Color(0xFF34C759)
                        else MaterialTheme.colorScheme.onSurface,
                        CircleShape,
                    ),
            )
            Spacer(modifier = Modifier.width(8.dp))
            Text(
                text = order.supplierName.ifEmpty { "Unknown Supplier" },
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold,
                color = MaterialTheme.colorScheme.onSurface,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f),
            )
            Text(
                text = order.state.replace("_", " "),
                style = MaterialTheme.typography.labelSmall,
                color = if (order.isApproaching) Color(0xFF34C759) else MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        Spacer(modifier = Modifier.height(8.dp))
        Text(
            text = order.items.joinToString { "${it.productName} ×${it.quantity}" }.ifEmpty { "No items" },
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            maxLines = 2,
            overflow = TextOverflow.Ellipsis,
        )
        Spacer(modifier = Modifier.height(4.dp))
        Text(
            text = formatAmount(order.totalAmount),
            style = MaterialTheme.typography.labelMedium,
            fontWeight = FontWeight.Medium,
            color = MaterialTheme.colorScheme.onSurface,
        )
    }
}

private fun formatAmount(amount: Long): String {
    val formatted = String.format("%,d", amount).replace(',', ' ')
    return "$formatted"
}
