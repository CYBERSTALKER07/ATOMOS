package com.pegasusx.driver.ui.screens.map

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.location.Location
import android.net.Uri
import android.os.Looper
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
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
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.MyLocation
import androidx.compose.material.icons.filled.Navigation
import androidx.compose.material.icons.filled.Schedule
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.core.content.ContextCompat
import com.google.android.gms.location.LocationCallback
import com.google.android.gms.location.LocationRequest
import com.google.android.gms.location.LocationResult
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import com.google.android.gms.maps.CameraUpdateFactory
import com.google.android.gms.maps.model.BitmapDescriptorFactory
import com.google.android.gms.maps.model.CameraPosition
import com.google.android.gms.maps.model.LatLng
import com.google.android.gms.maps.model.LatLngBounds
import com.google.maps.android.compose.CameraMoveStartedReason
import com.google.maps.android.compose.GoogleMap
import com.google.maps.android.compose.MapProperties
import com.google.maps.android.compose.MapUiSettings
import com.google.maps.android.compose.Marker
import com.google.maps.android.compose.MarkerState
import com.google.maps.android.compose.Polyline
import com.google.maps.android.compose.rememberCameraPositionState
import com.pegasusx.driver.data.telemetry.LocationInterpolator
import com.pegasusx.driver.data.telemetry.MapCameraConfig
import com.pegasusx.driver.data.telemetry.MapCameraMath
import com.pegasusx.driver.data.telemetry.buildPlannedRoutePolyline
import com.pegasusx.driver.data.telemetry.LocationTrail
import com.pegasusx.driver.data.telemetry.formatNavigationDistance
import com.pegasusx.driver.data.model.Order
import com.pegasusx.driver.data.model.OrderState
import com.pegasusx.driver.ui.screens.manifest.ManifestViewModel
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import kotlinx.coroutines.channels.awaitClose
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.callbackFlow
import kotlinx.coroutines.flow.emptyFlow
import java.text.SimpleDateFormat
import java.util.Locale
import java.util.TimeZone

private fun locationFlow(context: android.content.Context): Flow<Location> = callbackFlow {
    val client = LocationServices.getFusedLocationProviderClient(context)
    val request = LocationRequest.Builder(Priority.PRIORITY_HIGH_ACCURACY, 1000L).build()
    val callback = object : LocationCallback() {
        override fun onLocationResult(result: LocationResult) {
            result.lastLocation?.let { trySend(it) }
        }
    }
    if (ContextCompat.checkSelfPermission(context, Manifest.permission.ACCESS_FINE_LOCATION) == PackageManager.PERMISSION_GRANTED) {
        client.requestLocationUpdates(request, callback, Looper.getMainLooper())
    }
    awaitClose { client.removeLocationUpdates(callback) }
}

@Composable
fun MapScreen(
    viewModel: ManifestViewModel,
    onOpenScanner: () -> Unit = {},
    onOpenCorrection: (orderId: String, retailerName: String) -> Unit = { _, _ -> },
) {
    val lab = LocalPegasusColors.current
    val context = LocalContext.current
    val uiState by viewModel.state.collectAsState()

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

    val activeOrders = remember(uiState.orders) {
        uiState.orders.filter {
            it.state != OrderState.COMPLETED && it.state != OrderState.CANCELLED &&
                    it.latitude != null && it.longitude != null
        }
    }

    var selectedOrder by remember { mutableStateOf<Order?>(null) }

    val activeOrder = remember(uiState.orders) { resolveActiveOrder(uiState.orders) }
    val mapPhase = remember(activeOrder, selectedOrder) {
        resolveMapPhase(activeOrder ?: selectedOrder)
    }

    LaunchedEffect(activeOrder?.id) {
        if (activeOrder != null && selectedOrder?.id != activeOrder.id) {
            selectedOrder = activeOrder
        }
    }

    // Default center: Tashkent
    val defaultPosition = LatLng(41.2995, 69.2401)
    val cameraPositionState = rememberCameraPositionState {
        position = CameraPosition.fromLatLngZoom(defaultPosition, 12f)
    }

    var isCameraLocked by remember { mutableStateOf(mapPhase == MapPhase.NAVIGATING) }
    var userPannedAt by remember { mutableStateOf<Long?>(null) }

    val locationInterpolator = remember { LocationInterpolator() }

    val locationFlow = remember(hasLocationPermission) {
        if (hasLocationPermission) locationFlow(context) else emptyFlow()
    }
    val currentLocation by locationFlow.collectAsState(initial = null)
    val trailPoints by LocationTrail.points.collectAsState()

    LaunchedEffect(currentLocation) {
        currentLocation?.let { locationInterpolator.onGps(it) }
    }

    LaunchedEffect(Unit) {
        while (isActive) {
            locationInterpolator.tick()
            delay(MapCameraConfig.INTERPOLATION_FRAME_MS)
        }
    }

    val displayPosition by locationInterpolator.position.collectAsState()
    val displayBearing by locationInterpolator.bearing.collectAsState()
    val displaySpeed by locationInterpolator.speedMps.collectAsState()

    LaunchedEffect(displayPosition, uiState.routeSteps, uiState.routeGeometry, mapPhase, activeOrder?.routeId) {
        val position = displayPosition ?: return@LaunchedEffect
        viewModel.updateNavigationCue(position.latitude, position.longitude)
        if (mapPhase == MapPhase.NAVIGATING) {
            viewModel.checkRouteDeviation(activeOrder?.routeId, position.latitude, position.longitude)
        }
    }

    val plannedRoutePoints = remember(activeOrders, activeOrder?.routeId) {
        buildPlannedRoutePolyline(activeOrders, activeOrder?.routeId)
    }
    val routePolylinePoints = remember(uiState.routeGeometry, plannedRoutePoints) {
        if (uiState.routeGeometry.size >= 2) {
            uiState.routeGeometry.map { LatLng(it.lat, it.lng) }
        } else {
            plannedRoutePoints
        }
    }

    LaunchedEffect(mapPhase) {
        if (mapPhase == MapPhase.NAVIGATING) {
            isCameraLocked = true
            userPannedAt = null
        }
    }

    LaunchedEffect(userPannedAt) {
        val pannedAt = userPannedAt ?: return@LaunchedEffect
        if (mapPhase != MapPhase.NAVIGATING) return@LaunchedEffect
        delay(MapCameraConfig.IDLE_RECENTER_MS)
        if (userPannedAt == pannedAt) {
            isCameraLocked = true
            userPannedAt = null
        }
    }

    // 1. Spatiotemporal Camera Lock (Dynamic Bounding)
    LaunchedEffect(displayPosition, displayBearing, displaySpeed, isCameraLocked) {
        val pos = displayPosition ?: return@LaunchedEffect
        if (!isCameraLocked) return@LaunchedEffect

        val position = MapCameraMath.trackingCameraPosition(
            lat = pos.latitude,
            lng = pos.longitude,
            bearing = displayBearing,
            speedMps = displaySpeed,
            fallbackBearing = cameraPositionState.position.bearing,
        )

        cameraPositionState.animate(
            CameraUpdateFactory.newCameraPosition(position),
            MapCameraConfig.CAMERA_ANIMATION_MS,
        )
    }

    // 2. Gesture Break (Intentional User Override)
    LaunchedEffect(cameraPositionState.isMoving) {
        if (cameraPositionState.isMoving &&
            cameraPositionState.cameraMoveStartedReason == CameraMoveStartedReason.GESTURE
        ) {
            isCameraLocked = false
            userPannedAt = System.currentTimeMillis()
        }
    }

    // Fit camera to orders when they load (if not locked)
    LaunchedEffect(activeOrders) {
        if (!isCameraLocked && activeOrders.isNotEmpty() && mapPhase != MapPhase.NAVIGATING) {
            val boundsBuilder = LatLngBounds.builder()
            activeOrders.forEach { order ->
                boundsBuilder.include(LatLng(order.latitude!!, order.longitude!!))
            }
            try {
                val bounds = boundsBuilder.build()
                cameraPositionState.animate(
                    CameraUpdateFactory.newLatLngBounds(bounds, 80),
                    durationMs = 600
                )
            } catch (_: Exception) {
                // Single point — just zoom to it
                val first = activeOrders.first()
                cameraPositionState.animate(
                    CameraUpdateFactory.newLatLngZoom(
                        LatLng(first.latitude!!, first.longitude!!), 14f
                    )
                )
            }
        }
    }

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(lab.bg)
    ) {
        GoogleMap(
            modifier = Modifier.fillMaxSize(),
            cameraPositionState = cameraPositionState,
            properties = MapProperties(
                isMyLocationEnabled = hasLocationPermission && !(isCameraLocked && mapPhase == MapPhase.NAVIGATING),
                isBuildingEnabled = true
            ),
            uiSettings = MapUiSettings(
                zoomControlsEnabled = false,
                myLocationButtonEnabled = false, // We use custom FAB to handle lock break
                mapToolbarEnabled = false,
                tiltGesturesEnabled = true,
                compassEnabled = false
            ),
            onMapClick = { selectedOrder = null }
        ) {
            if (trailPoints.size >= 2) {
                Polyline(
                    points = trailPoints,
                    color = lab.fg.copy(alpha = 0.35f),
                    width = 6f,
                )
            }
            if (routePolylinePoints.size >= 2) {
                Polyline(
                    points = routePolylinePoints,
                    color = MaterialTheme.colorScheme.primary.copy(alpha = 0.55f),
                    width = 5f,
                )
            }
            val driverPin = displayPosition
            if (isCameraLocked && mapPhase == MapPhase.NAVIGATING && driverPin != null) {
                Marker(
                    state = MarkerState(position = driverPin),
                    title = "You",
                    rotation = displayBearing,
                    flat = true,
                    icon = BitmapDescriptorFactory.defaultMarker(BitmapDescriptorFactory.HUE_AZURE),
                )
            }
            activeOrders.forEach { order ->
                val position = LatLng(order.latitude!!, order.longitude!!)
                val hue = when (order.state) {
                    OrderState.IN_TRANSIT -> BitmapDescriptorFactory.HUE_AZURE
                    OrderState.ARRIVING, OrderState.ARRIVED -> BitmapDescriptorFactory.HUE_GREEN
                    OrderState.LOADED -> BitmapDescriptorFactory.HUE_ORANGE
                    else -> BitmapDescriptorFactory.HUE_RED
                }
                Marker(
                    state = MarkerState(position = position),
                    title = order.retailerName,
                    snippet = order.state.name,
                    icon = BitmapDescriptorFactory.defaultMarker(hue),
                    onClick = {
                        selectedOrder = order
                        true
                    }
                )
            }
        }

        // Order count + phase badge
        if (activeOrders.isNotEmpty()) {
            Column(
                modifier = Modifier
                    .align(Alignment.TopStart)
                    .padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(6.dp),
            ) {
                Box(
                    modifier = Modifier
                        .background(
                            MaterialTheme.colorScheme.primaryContainer,
                            RoundedCornerShape(12.dp)
                        )
                        .padding(horizontal = 12.dp, vertical = 8.dp)
                ) {
                    Text(
                        text = "${activeOrders.size} active stop${if (activeOrders.size != 1) "s" else ""}",
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.onPrimaryContainer
                    )
                }
                Box(
                    modifier = Modifier
                        .background(
                            MaterialTheme.colorScheme.surfaceContainerHigh,
                            RoundedCornerShape(12.dp)
                        )
                        .padding(horizontal = 12.dp, vertical = 6.dp)
                ) {
                    Text(
                        text = mapPhase.name,
                        style = MaterialTheme.typography.labelSmall.copy(
                            fontFamily = FontFamily.Monospace,
                            fontWeight = FontWeight.Bold,
                        ),
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }

        if (mapPhase == MapPhase.NAVIGATING && uiState.navigationCue != null) {
            val cue = uiState.navigationCue!!
            Column(
                modifier = Modifier
                    .align(Alignment.TopCenter)
                    .padding(top = 88.dp, start = 16.dp, end = 16.dp)
                    .fillMaxWidth()
                    .background(
                        MaterialTheme.colorScheme.surfaceContainerHigh,
                        RoundedCornerShape(16.dp),
                    )
                    .padding(horizontal = 16.dp, vertical = 12.dp),
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Icon(
                        imageVector = Icons.Default.Navigation,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.size(20.dp),
                    )
                    Spacer(modifier = Modifier.width(10.dp))
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            text = formatNavigationDistance(cue.distanceM),
                            style = MaterialTheme.typography.labelLarge.copy(fontWeight = FontWeight.Bold),
                            color = MaterialTheme.colorScheme.primary,
                        )
                        Text(
                            text = cue.instruction,
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurface,
                            maxLines = 2,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                }
            }
        }

        // Selected order info card
        selectedOrder?.let { order ->
            Column(
                modifier = Modifier
                    .align(Alignment.BottomCenter)
                    .padding(16.dp)
                    .fillMaxWidth()
                    .background(
                        MaterialTheme.colorScheme.surfaceContainerHigh,
                        RoundedCornerShape(16.dp)
                    )
                    .padding(16.dp)
            ) {
                Text(
                    text = order.retailerName,
                    style = MaterialTheme.typography.titleMedium,
                    color = MaterialTheme.colorScheme.onSurface,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
                Spacer(modifier = Modifier.height(4.dp))
                Text(
                    text = order.deliveryAddress,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis
                )
                Spacer(modifier = Modifier.height(8.dp))

                // ETA row
                val etaText = formatETA(order)
                if (etaText != null) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(6.dp)
                    ) {
                        Icon(
                            imageVector = Icons.Default.Schedule,
                            contentDescription = null,
                            tint = MaterialTheme.colorScheme.primary,
                            modifier = Modifier.size(14.dp)
                        )
                        Text(
                            text = etaText,
                            style = MaterialTheme.typography.labelMedium.copy(
                                fontWeight = FontWeight.Bold,
                                fontFamily = FontFamily.Monospace
                            ),
                            color = MaterialTheme.colorScheme.primary
                        )
                    }
                    Spacer(modifier = Modifier.height(8.dp))
                }

                Text(
                    text = "${order.state.name} — ${order.items.size} item${if (order.items.size != 1) "s" else ""} — ${formatAmount(order.totalAmount)}",
                    style = MaterialTheme.typography.labelMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )

                if (order.latitude != null && order.longitude != null) {
                    Spacer(modifier = Modifier.height(12.dp))
                    FilledTonalButton(
                        onClick = {
                            val uri = Uri.parse("google.navigation:q=${order.latitude},${order.longitude}&mode=d")
                            val intent = Intent(Intent.ACTION_VIEW, uri).apply {
                                setPackage("com.google.android.apps.maps")
                            }
                            if (intent.resolveActivity(context.packageManager) != null) {
                                context.startActivity(intent)
                            } else {
                                val webUri = Uri.parse("https://www.google.com/maps/dir/?api=1&destination=${order.latitude},${order.longitude}&travelmode=driving")
                                context.startActivity(Intent(Intent.ACTION_VIEW, webUri))
                            }
                        },
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Icon(
                            imageVector = Icons.Default.Navigation,
                            contentDescription = null,
                            modifier = Modifier.size(16.dp)
                        )
                        Spacer(modifier = Modifier.width(8.dp))
                        Text("Navigate", style = MaterialTheme.typography.labelLarge)
                    }
                }

                val phaseForCard = resolveMapPhase(activeOrder ?: order)
                if (phaseForCard == MapPhase.ARRIVED || phaseForCard == MapPhase.VERIFYING) {
                    Spacer(modifier = Modifier.height(8.dp))
                    FilledTonalButton(
                        onClick = onOpenScanner,
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Text(
                            if (phaseForCard == MapPhase.VERIFYING) "Scan Proof of Delivery" else "Scan QR",
                            style = MaterialTheme.typography.labelLarge,
                        )
                    }
                    Spacer(modifier = Modifier.height(8.dp))
                    FilledTonalButton(
                        onClick = { onOpenCorrection(order.id, order.retailerName) },
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Text("Delivery Correction", style = MaterialTheme.typography.labelLarge)
                    }
                }
            }
        }

        // Camera Lock FAB
        Box(
            modifier = Modifier
                .align(Alignment.BottomEnd)
                .padding(bottom = if (selectedOrder != null) 180.dp else 16.dp, end = 16.dp)
        ) {
            FloatingActionButton(
                onClick = {
                    isCameraLocked = true
                    userPannedAt = null
                },
                containerColor = if (isCameraLocked) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.surface,
                contentColor = if (isCameraLocked) MaterialTheme.colorScheme.onPrimary else MaterialTheme.colorScheme.onSurface
            ) {
                Icon(Icons.Default.MyLocation, contentDescription = "Lock Camera")
            }
        }

        // Empty state
        if (activeOrders.isEmpty() && !uiState.isLoading) {
            Box(
                modifier = Modifier.fillMaxSize(),
                contentAlignment = Alignment.Center
            ) {
                Text(
                    text = "No active deliveries",
                    fontSize = 14.sp,
                    fontWeight = FontWeight.Medium,
                    color = lab.fgTertiary
                )
            }
        }
    }
}

private fun formatAmount(amount: Long): String {
    val formatted = String.format("%,d", amount).replace(',', ' ')
    return "$formatted"
}

private fun formatETA(order: Order): String? {
    val etaSec = order.etaDurationSec ?: return null
    val distM = order.etaDistanceM

    val parts = mutableListOf<String>()

    // Time part
    if (order.estimatedArrivalAt != null) {
        try {
            val sdf = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss", Locale.US)
            sdf.timeZone = TimeZone.getTimeZone("UTC")
            val date = sdf.parse(order.estimatedArrivalAt)
            if (date != null) {
                val localFmt = SimpleDateFormat("HH:mm", Locale.getDefault())
                parts.add("ETA ${localFmt.format(date)}")
            }
        } catch (_: Exception) {
            // Fall through to duration display
        }
    }

    // Duration part
    val mins = etaSec / 60
    if (mins >= 60) {
        parts.add("${mins / 60}h ${mins % 60}m")
    } else {
        parts.add("${mins}m")
    }

    // Distance part
    if (distM != null && distM > 0) {
        val km = distM / 1000.0
        parts.add(String.format(Locale.US, "%.1f km", km))
    }

    return parts.joinToString(" · ")
}
