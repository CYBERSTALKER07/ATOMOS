package com.pegasus.retailer.ui.screens.auth

import android.Manifest
import android.annotation.SuppressLint
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
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.MyLocation
import androidx.compose.material.icons.filled.Place
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.google.accompanist.permissions.ExperimentalPermissionsApi
import com.google.accompanist.permissions.isGranted
import com.google.accompanist.permissions.rememberPermissionState
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import com.google.android.gms.maps.CameraUpdateFactory
import com.google.android.gms.maps.model.CameraPosition
import com.google.android.gms.maps.model.LatLng
import com.google.android.gms.tasks.CancellationTokenSource
import com.google.maps.android.compose.GoogleMap
import com.google.maps.android.compose.MapProperties
import com.google.maps.android.compose.MapUiSettings
import com.google.maps.android.compose.rememberCameraPositionState
import com.pegasus.retailer.ui.theme.PillShape
import kotlinx.coroutines.launch
import kotlinx.coroutines.tasks.await

// Default: Tashkent city center
private val TASHKENT = LatLng(41.2995, 69.2401)

data class PickedLocation(
    val latitude: Double,
    val longitude: Double,
    val displayText: String,
)

@OptIn(ExperimentalPermissionsApi::class)
@SuppressLint("MissingPermission")
@Composable
fun LocationPickerScreen(
    initialLat: Double = 0.0,
    initialLng: Double = 0.0,
    onConfirm: (PickedLocation) -> Unit,
    onBack: () -> Unit,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()

    val locationPermission = rememberPermissionState(Manifest.permission.ACCESS_FINE_LOCATION)
    val fusedClient = remember { LocationServices.getFusedLocationProviderClient(context) }

    val startPosition = if (initialLat != 0.0 || initialLng != 0.0) {
        LatLng(initialLat, initialLng)
    } else {
        TASHKENT
    }

    val cameraPositionState = rememberCameraPositionState {
        position = CameraPosition.fromLatLngZoom(startPosition, 15f)
    }

    var selectedPosition by remember { mutableStateOf(startPosition) }

    // Track camera center as the selected position
    LaunchedEffect(cameraPositionState.isMoving) {
        if (!cameraPositionState.isMoving) {
            selectedPosition = cameraPositionState.position.target
        }
    }

    Box(modifier = Modifier.fillMaxSize()) {
        // Map
        GoogleMap(
            modifier = Modifier.fillMaxSize(),
            cameraPositionState = cameraPositionState,
            properties = MapProperties(
                isMyLocationEnabled = locationPermission.status.isGranted,
            ),
            uiSettings = MapUiSettings(
                myLocationButtonEnabled = false,
                zoomControlsEnabled = false,
                mapToolbarEnabled = false,
            ),
        )

        // Center pin overlay (always centered on screen)
        Box(
            modifier = Modifier.fillMaxSize(),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                imageVector = Icons.Default.Place,
                contentDescription = "Selected location",
                tint = Color.Black,
                modifier = Modifier
                    .size(48.dp)
                    .padding(bottom = 24.dp), // offset so pin tip is at center
            )
        }

        // Top bar
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .background(Color.White.copy(alpha = 0.92f))
                .padding(horizontal = 8.dp, vertical = 12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = onBack) {
                Icon(
                    imageVector = Icons.AutoMirrored.Filled.ArrowBack,
                    contentDescription = "Back",
                    tint = Color.Black,
                )
            }
            Text(
                text = "Pick Store Location",
                style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.SemiBold),
                color = Color.Black,
                modifier = Modifier.weight(1f),
                textAlign = TextAlign.Center,
            )
            // Spacer to balance the row
            Spacer(modifier = Modifier.size(48.dp))
        }

        // Bottom panel
        Column(
            modifier = Modifier
                .align(Alignment.BottomCenter)
                .fillMaxWidth()
                .background(Color.White.copy(alpha = 0.95f))
                .padding(horizontal = 24.dp, vertical = 16.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text(
                text = "%.5f, %.5f".format(selectedPosition.latitude, selectedPosition.longitude),
                style = MaterialTheme.typography.bodyMedium,
                color = Color.Black.copy(alpha = 0.6f),
            )
            Spacer(Modifier.height(4.dp))
            Text(
                text = "Drag map to adjust pin position",
                style = MaterialTheme.typography.bodySmall,
                color = Color.Black.copy(alpha = 0.4f),
            )
            Spacer(Modifier.height(12.dp))
            Button(
                onClick = {
                    onConfirm(
                        PickedLocation(
                            latitude = selectedPosition.latitude,
                            longitude = selectedPosition.longitude,
                            displayText = "%.5f, %.5f".format(
                                selectedPosition.latitude,
                                selectedPosition.longitude,
                            ),
                        ),
                    )
                },
                modifier = Modifier
                    .fillMaxWidth()
                    .height(48.dp),
                shape = PillShape,
                colors = ButtonDefaults.buttonColors(
                    containerColor = Color.Black,
                    contentColor = Color.White,
                ),
            ) {
                Icon(
                    imageVector = Icons.Default.Check,
                    contentDescription = null,
                    modifier = Modifier.size(18.dp),
                )
                Spacer(Modifier.size(8.dp))
                Text("Confirm Location", fontWeight = FontWeight.SemiBold, fontSize = 15.sp)
            }
        }

        // My-location FAB
        FloatingActionButton(
            onClick = {
                if (!locationPermission.status.isGranted) {
                    locationPermission.launchPermissionRequest()
                    return@FloatingActionButton
                }
                scope.launch {
                    try {
                        val loc = fusedClient.getCurrentLocation(
                            Priority.PRIORITY_HIGH_ACCURACY,
                            CancellationTokenSource().token,
                        ).await()
                        if (loc != null) {
                            val target = LatLng(loc.latitude, loc.longitude)
                            cameraPositionState.animate(
                                CameraUpdateFactory.newLatLngZoom(target, 17f),
                                durationMs = 600,
                            )
                        }
                    } catch (_: Exception) {
                        // Location unavailable — ignore
                    }
                }
            },
            modifier = Modifier
                .align(Alignment.BottomEnd)
                .padding(end = 16.dp, bottom = 140.dp),
            containerColor = Color.White,
            contentColor = Color.Black,
            shape = CircleShape,
        ) {
            Icon(
                imageVector = Icons.Default.MyLocation,
                contentDescription = "Use my location",
            )
        }
    }
}
