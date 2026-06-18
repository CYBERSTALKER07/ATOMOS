package com.pegasusx.supplier.ui.components

import android.Manifest
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.google.accompanist.permissions.ExperimentalPermissionsApi
import com.google.accompanist.permissions.isGranted
import com.google.accompanist.permissions.rememberPermissionState
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import com.google.android.gms.tasks.CancellationTokenSource
import com.pegasusx.supplier.data.remote.GeocodeApi
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.tasks.await

data class AddressLocationValue(
    val address: String = "",
    val lat: Double = 0.0,
    val lng: Double = 0.0,
    val placeId: String? = null,
)

@OptIn(ExperimentalPermissionsApi::class)
@Composable
fun AddressLocationField(
    geocodeApi: GeocodeApi,
    value: AddressLocationValue,
    onValueChange: (AddressLocationValue) -> Unit,
    label: String = "Address",
) {
    val scope = rememberCoroutineScope()
    val context = androidx.compose.ui.platform.LocalContext.current
    var query by remember(value.address) { mutableStateOf(value.address) }
    var suggestions by remember { mutableStateOf(emptyList<Pair<String, String>>()) }
    var error by remember { mutableStateOf<String?>(null) }
    var debounceJob by remember { mutableStateOf<Job?>(null) }
    val locationPermission = rememberPermissionState(Manifest.permission.ACCESS_FINE_LOCATION)

    fun applyResolved(address: String, lat: Double, lng: Double, placeId: String? = null) {
        query = address
        suggestions = emptyList()
        onValueChange(AddressLocationValue(address = address, lat = lat, lng = lng, placeId = placeId))
    }

    Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
        OutlinedTextField(
            value = query,
            onValueChange = { text ->
                query = text
                error = null
                debounceJob?.cancel()
                debounceJob = scope.launch {
                    delay(250)
                    if (text.trim().length < 3) {
                        suggestions = emptyList()
                        return@launch
                    }
                    runCatching {
                        geocodeApi.autocomplete(text.trim()).predictions.map {
                            it.placeId to it.description
                        }
                    }.onSuccess { suggestions = it }
                }
            },
            label = { Text(label) },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true,
        )
        suggestions.take(5).forEach { (placeId, description) ->
            Text(
                text = description,
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable {
                        scope.launch {
                            runCatching { geocodeApi.resolvePlace(placeId) }
                                .onSuccess { loc -> applyResolved(loc.address.ifBlank { description }, loc.lat, loc.lng, loc.placeId) }
                                .onFailure { error = it.message }
                        }
                    }
                    .padding(vertical = 6.dp, horizontal = 4.dp),
            )
        }
        TextButton(
            onClick = {
                if (!locationPermission.status.isGranted) {
                    locationPermission.launchPermissionRequest()
                    return@TextButton
                }
                scope.launch {
                    runCatching {
                        val fused = LocationServices.getFusedLocationProviderClient(context)
                        val loc = fused.getCurrentLocation(
                            Priority.PRIORITY_HIGH_ACCURACY,
                            CancellationTokenSource().token,
                        ).await() ?: error("location_unavailable")
                        geocodeApi.reverse(loc.latitude, loc.longitude)
                    }.onSuccess { resolved ->
                        applyResolved(resolved.address, resolved.lat, resolved.lng, resolved.placeId)
                    }.onFailure { error = it.message }
                }
            },
        ) { Text("Share my location") }
        if (value.address.isNotBlank()) {
            Text(
                text = "Saved for dispatch routing",
                style = androidx.compose.material3.MaterialTheme.typography.bodySmall,
            )
        }
        if (!error.isNullOrBlank()) {
            Text(text = error!!, color = androidx.compose.material3.MaterialTheme.colorScheme.error)
        }
    }
}
