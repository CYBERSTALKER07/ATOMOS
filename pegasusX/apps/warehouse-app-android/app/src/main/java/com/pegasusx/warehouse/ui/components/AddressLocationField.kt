package com.pegasusx.warehouse.ui.components

import androidx.compose.ui.res.stringResource

import android.Manifest
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.MaterialTheme
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
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.dp
import com.google.accompanist.permissions.ExperimentalPermissionsApi
import com.google.accompanist.permissions.isGranted
import com.google.accompanist.permissions.rememberPermissionState
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import com.google.android.gms.tasks.CancellationTokenSource
import com.pegasusx.warehouse.data.model.ForwardGeocodeRequest
import com.pegasusx.warehouse.data.remote.GeocodeApi
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import com.pegasusx.warehouse.util.GeocodeLocationSupport
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.tasks.await
import com.pegasusx.warehouse.R

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
    val context = LocalContext.current
    val focusManager = LocalFocusManager.current
    var query by remember(value.address) { mutableStateOf(value.address) }
    var suggestions by remember { mutableStateOf(emptyList<Pair<String, String>>()) }
    var error by remember { mutableStateOf<String?>(null) }
    var resolving by remember { mutableStateOf(false) }
    var debounceJob by remember { mutableStateOf<Job?>(null) }
    val locationPermission = rememberPermissionState(Manifest.permission.ACCESS_FINE_LOCATION)

    fun applyResolved(address: String, lat: Double, lng: Double, placeId: String? = null) {
        query = address
        suggestions = emptyList()
        error = null
        onValueChange(AddressLocationValue(address = address, lat = lat, lng = lng, placeId = placeId))
    }

    suspend fun resolveText(text: String): Boolean {
        val trimmed = text.trim()
        if (trimmed.isEmpty()) return false
        val top = runCatching { geocodeApi.autocomplete(trimmed).predictions.firstOrNull() }.getOrNull()
        if (!top?.placeId.isNullOrBlank()) {
            val byPlace = runCatching { geocodeApi.resolvePlace(top!!.placeId) }.getOrNull()
            if (byPlace != null && GeocodeLocationSupport.hasValidCoordinates(byPlace.lat, byPlace.lng)) {
                applyResolved(byPlace.address.ifBlank { trimmed }, byPlace.lat, byPlace.lng, byPlace.placeId)
                return true
            }
        }
        val byAddress = runCatching { geocodeApi.forward(ForwardGeocodeRequest(trimmed)) }.getOrNull()
        if (byAddress != null && GeocodeLocationSupport.hasValidCoordinates(byAddress.lat, byAddress.lng)) {
            applyResolved(byAddress.address.ifBlank { trimmed }, byAddress.lat, byAddress.lng, byAddress.placeId)
            return true
        }
        return false
    }

    Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
        OutlinedTextField(
            value = query,
            onValueChange = { text ->
                query = text
                error = null
                onValueChange(value.copy(address = text))
                debounceJob?.cancel()
                debounceJob = scope.launch {
                    delay(250)
                    if (text.trim().length < 3) {
                        suggestions = emptyList()
                        return@launch
                    }
                    runCatching {
                        geocodeApi.autocomplete(text.trim()).predictions.map { it.placeId to it.description }
                    }.onSuccess { suggestions = it }
                }
            },
            label = { Text(label) },
            modifier = Modifier
                .fillMaxWidth()
                .onFocusChanged { state ->
                    if (!state.isFocused && query.trim().isNotEmpty()) {
                        scope.launch {
                            if (value.address == query.trim() && GeocodeLocationSupport.hasValidCoordinates(value.lat, value.lng)) {
                                return@launch
                            }
                            resolving = true
                            if (!resolveText(query)) {
                                error = "Pick an address from the list or refine your search."
                            }
                            resolving = false
                        }
                    }
                },
            singleLine = true,
            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
            keyboardActions = KeyboardActions(onDone = { focusManager.clearFocus() }),
        )
        suggestions.take(5).forEach { (placeId, description) ->
            Text(
                text = description,
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable {
                        scope.launch {
                            resolving = true
                            error = null
                            if (!placeId.isNullOrBlank()) {
                                runCatching { geocodeApi.resolvePlace(placeId) }
                                    .onSuccess { loc ->
                                        applyResolved(loc.address.ifBlank { description }, loc.lat, loc.lng, loc.placeId)
                                        resolving = false
                                        return@launch
                                    }
                            }
                            if (!resolveText(description)) {
                                error = "Could not resolve that address."
                            }
                            resolving = false
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
                    resolving = true
                    error = null
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
                    resolving = false
                }
            },
            enabled = !resolving,
        ) { Text(if (resolving) "Resolving…" else "Share my location") }
        when {
            GeocodeLocationSupport.hasValidCoordinates(value.lat, value.lng) -> {
                Text(
                    text = stringResource(R.string.supplier_portal_location_picker_text_pinned_for_dispatch_routing),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            resolving -> {
                Text(
                    text = stringResource(R.string.factory_portal_location_picker_text_resolving_address),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
        if (!error.isNullOrBlank()) {
            Text(text = error!!, color = MaterialTheme.colorScheme.error)
        }
    }
}
