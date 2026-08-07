package com.pegasusx.warehouse.ui.screens.setup

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import com.pegasusx.warehouse.data.model.WarehouseLocationPatchRequest
import com.pegasusx.warehouse.data.model.WarehouseSetupRequest
import com.pegasusx.warehouse.data.remote.GeocodeApi
import com.pegasusx.warehouse.data.remote.TokenHolder
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.ui.components.AddressLocationField
import com.pegasusx.warehouse.ui.components.AddressLocationValue
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import com.pegasusx.warehouse.util.GeocodeLocationSupport
import com.pegasusx.warehouse.util.WarehouseIdempotencyKeys
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LocationSetupScreen(
    api: WarehouseApi,
    geocodeApi: GeocodeApi,
    onComplete: () -> Unit,
) {
    val hasAssignedWarehouse = TokenHolder.hasAssignedWarehouse
    var warehouseName by remember { mutableStateOf("") }
    var location by remember { mutableStateOf(AddressLocationValue()) }
    var loading by remember { mutableStateOf(hasAssignedWarehouse) }
    var submitting by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(hasAssignedWarehouse) {
        if (!hasAssignedWarehouse) return@LaunchedEffect
        loading = true
        runCatching { api.getWarehouseLocation() }
            .onSuccess { resp ->
                val body = resp.body()
                if (resp.isSuccessful && body != null) {
                    warehouseName = body.name
                    location = AddressLocationValue(
                        address = body.address,
                        lat = body.lat,
                        lng = body.lng,
                        placeId = body.placeId,
                    )
                }
            }
        loading = false
    }

    fun submit() {
        scope.launch {
            submitting = true
            error = null
            try {
                val resolved = GeocodeLocationSupport.resolveLocationValue(geocodeApi, location)
                    ?: run {
                        error = "Select an address from the suggestions or share your location."
                        submitting = false
                        return@launch
                    }
                location = resolved

                if (hasAssignedWarehouse) {
                    val resp = api.patchWarehouseLocation(
                        WarehouseLocationPatchRequest(
                            address = resolved.address,
                            placeId = resolved.placeId,
                            lat = resolved.lat,
                            lng = resolved.lng,
                        ),
                        WarehouseIdempotencyKeys.opsLocation(resolved.lat, resolved.lng, resolved.placeId),
                    )
                    if (!resp.isSuccessful) {
                        error = "Setup failed (${resp.code()})"
                        submitting = false
                        return@launch
                    }
                    TokenHolder.refreshToken?.let { refresh ->
                        runCatching { api.refreshToken(com.pegasusx.warehouse.data.model.RefreshTokenRequest(refresh)) }
                            .onSuccess { refreshResp ->
                                val auth = refreshResp.body()
                                if (refreshResp.isSuccessful && auth != null) {
                                    TokenHolder.token = auth.token
                                    TokenHolder.refreshToken = auth.refreshToken
                                    if (auth.warehouseId.isNotBlank()) TokenHolder.warehouseId = auth.warehouseId
                                }
                            }
                    }
                } else {
                    if (warehouseName.trim().length < 3) {
                        error = "Warehouse name is required."
                        submitting = false
                        return@launch
                    }
                    val resp = api.setupWarehouse(
                        WarehouseSetupRequest(
                            name = warehouseName.trim(),
                            address = resolved.address,
                            placeId = resolved.placeId,
                            lat = resolved.lat,
                            lng = resolved.lng,
                        ),
                    )
                    val body = resp.body()
                    if (!resp.isSuccessful || body?.token.isNullOrBlank()) {
                        error = "Setup failed (${resp.code()})"
                        submitting = false
                        return@launch
                    }
                    TokenHolder.token = body.token
                    body.refreshToken?.let { TokenHolder.refreshToken = it }
                    if (body.warehouseId.isNotBlank()) TokenHolder.warehouseId = body.warehouseId
                }
                onComplete()
            } catch (e: Exception) {
                error = e.message
            } finally {
                submitting = false
            }
        }
    }

    Scaffold(
        topBar = { TopAppBar(title = { Text("Warehouse location") }) },
    ) { padding ->
        if (loading) {
            CircularProgressIndicator(Modifier.padding(padding).padding(PegasusSpacing.lg))
            return@Scaffold
        }
        Column(
            modifier = Modifier
                .padding(padding)
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Text(
                if (hasAssignedWarehouse) {
                    "Confirm or update your depot address. Changes sync with dispatch and delivery routing."
                } else {
                    "Name your warehouse and set the depot address to start operations."
                },
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            if (!hasAssignedWarehouse) {
                OutlinedTextField(
                    value = warehouseName,
                    onValueChange = { warehouseName = it },
                    label = { Text("Warehouse name") },
                    modifier = Modifier.fillMaxWidth(),
                    singleLine = true,
                )
            } else if (warehouseName.isNotBlank()) {
                Text(warehouseName, style = MaterialTheme.typography.titleMedium)
            }
            AddressLocationField(
                geocodeApi = geocodeApi,
                value = location,
                onValueChange = { location = it },
                label = stringResource(R.string.factory_portal_settings_location_text_depot_address),
            )
            if (!error.isNullOrBlank()) {
                Text(error!!, color = MaterialTheme.colorScheme.error)
            }
            Button(onClick = { submit() }, enabled = !submitting, modifier = Modifier.fillMaxWidth()) {
                Text(if (submitting) "Saving…" else "Complete setup")
            }
        }
    }
}
