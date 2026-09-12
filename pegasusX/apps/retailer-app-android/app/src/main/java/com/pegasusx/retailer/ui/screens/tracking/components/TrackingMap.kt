package com.pegasusx.retailer.ui.screens.tracking.components

import androidx.compose.ui.res.stringResource

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material.icons.filled.MyLocation
import androidx.compose.material3.FloatingActionButtonDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.SmallFloatingActionButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.google.android.gms.maps.CameraUpdateFactory
import com.google.android.gms.maps.model.BitmapDescriptorFactory
import com.google.android.gms.maps.model.LatLng
import com.google.maps.android.compose.CameraPositionState
import com.google.maps.android.compose.GoogleMap
import com.google.maps.android.compose.MapProperties
import com.google.maps.android.compose.MapUiSettings
import com.google.maps.android.compose.Marker
import com.google.maps.android.compose.MarkerState
import com.google.maps.android.compose.Polyline
import androidx.compose.ui.graphics.Color
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.retailer.data.model.TrackingOrder
import kotlinx.coroutines.launch
import com.pegasusx.retailer.R

@Composable
fun TrackingMap(
    isLoading: Boolean,
    visibleOrders: List<TrackingOrder>,
    hasLocationPermission: Boolean,
    cameraPositionState: CameraPositionState,
    selectedOrder: TrackingOrder?,
    onOrderSelected: (TrackingOrder?) -> Unit,
    activeDeliveryCount: Int,
    emptyStateMessage: String,
    onRefresh: () -> Unit,
    modifier: Modifier = Modifier
) {
    val scope = rememberCoroutineScope()

    Box(modifier = modifier.fillMaxSize()) {
        if (isLoading && visibleOrders.isEmpty()) {
            PegasusLoadingState(
                title = stringResource(R.string.mobile_retailer_ui_loading_deliveries),
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
                onMapClick = { onOrderSelected(null) },
            ) {
                for (order in visibleOrders) {
                    val routePoints = order.routeGeometry?.coordinates
                        ?.map { LatLng(it.lat, it.lng) }
                        .orEmpty()
                    if (routePoints.size >= 2) {
                        Polyline(
                            points = routePoints,
                            color = Color(0xFF2563EB),
                            width = 10f,
                        )
                    }

                    val driverLat = order.driverLatitude ?: continue
                    val driverLng = order.driverLongitude ?: continue
                    val position = LatLng(driverLat, driverLng)

                    val isGreen = order.isApproaching || order.state == "ARRIVED"
                    val markerColor = if (isGreen) BitmapDescriptorFactory.HUE_GREEN
                    else BitmapDescriptorFactory.HUE_VIOLET

                    Marker(
                        state = MarkerState(position = position),
                        title = order.supplierName,
                        snippet = "${order.state} — ${order.items.size} item${if (order.items.size != 1) "s" else ""}",
                        icon = BitmapDescriptorFactory.defaultMarker(markerColor),
                        onClick = {
                            onOrderSelected(order)
                            true
                        },
                    )
                }
            }

            if (hasLocationPermission) {
                SmallFloatingActionButton(
                    onClick = {
                        scope.launch {
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
                    Icon(Icons.Default.MyLocation, contentDescription = stringResource(R.string.mobile_retailer_ui_my_location), modifier = Modifier.size(20.dp))
                }
            }

            if (activeDeliveryCount > 0) {
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
                            stringResource(R.string.mobile_retailer_ui_activedeliverycount_active, activeDeliveryCount),
                            style = MaterialTheme.typography.labelMedium,
                            color = MaterialTheme.colorScheme.onPrimaryContainer,
                        )
                    }
                }
            }

            AnimatedVisibility(
                visible = selectedOrder != null,
                modifier = Modifier.align(Alignment.BottomCenter),
                enter = slideInVertically(initialOffsetY = { it }) + fadeIn(),
                exit = slideOutVertically(targetOffsetY = { it }) + fadeOut(),
            ) {
                selectedOrder?.let { order ->
                    OrderInfoCard(order = order)
                }
            }

            if (visibleOrders.isEmpty() && !isLoading) {
                PegasusStatePane(
                    kind = PegasusStateKind.Empty,
                    headline = "No active deliveries",
                    body = emptyStateMessage,
                    actionLabel = "Refresh",
                    onAction = onRefresh,
                )
            }
        }
    }
}
