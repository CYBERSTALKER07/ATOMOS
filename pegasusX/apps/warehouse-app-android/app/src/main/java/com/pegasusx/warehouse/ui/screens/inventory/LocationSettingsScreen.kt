package com.pegasusx.warehouse.ui.screens.inventory

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.warehouse.data.model.WarehouseLocationPatchRequest
import com.pegasusx.warehouse.data.remote.GeocodeApi
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.ui.components.AddressLocationField
import com.pegasusx.warehouse.ui.components.AddressLocationValue
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import com.pegasusx.warehouse.util.GeocodeLocationSupport
import com.pegasusx.warehouse.util.WarehouseIdempotencyKeys
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LocationSettingsScreen(
    api: WarehouseApi,
    geocodeApi: GeocodeApi,
    onBack: (() -> Unit)? = null,
) {
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var saveMessage by remember { mutableStateOf<String?>(null) }
    var warehouseName by remember { mutableStateOf("") }
    var location by remember { mutableStateOf(AddressLocationValue()) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = api.getWarehouseLocation()
                if (resp.isSuccessful && resp.body() != null) {
                    val body = resp.body()!!
                    warehouseName = body.name
                    location = AddressLocationValue(
                        address = body.address,
                        lat = body.lat,
                        lng = body.lng,
                        placeId = body.placeId,
                    )
                } else {
                    error = "Failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    fun save() {
        scope.launch {
            saving = true
            saveMessage = null
            try {
                val resolved = GeocodeLocationSupport.resolveLocationValue(geocodeApi, location)
                    ?: run {
                        saveMessage = "Select an address from the suggestions or share your location."
                        saving = false
                        return@launch
                    }
                location = resolved
                val resp = api.patchWarehouseLocation(
                    WarehouseLocationPatchRequest(
                        address = resolved.address,
                        placeId = resolved.placeId,
                        lat = resolved.lat,
                        lng = resolved.lng,
                    ),
                    WarehouseIdempotencyKeys.opsLocation(resolved.lat, resolved.lng, resolved.placeId),
                )
                saveMessage = if (resp.isSuccessful) "Location saved" else "Save failed (${resp.code()})"
                if (resp.isSuccessful) load()
            } catch (e: Exception) {
                saveMessage = e.message
            } finally {
                saving = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Warehouse location") },
                navigationIcon = {
                    if (onBack != null) {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                        }
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> Box(Modifier.padding(padding).fillMaxSize()) {
                CircularProgressIndicator(Modifier.padding(PegasusSpacing.lg))
            }
            error != null -> Column(Modifier.padding(padding).padding(PegasusSpacing.lg)) {
                Text(error!!, color = MaterialTheme.colorScheme.error)
                TextButton(onClick = { load() }) { Text("Retry") }
            }
            else -> Column(
                modifier = Modifier
                    .padding(padding)
                    .verticalScroll(rememberScrollState())
                    .padding(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                if (warehouseName.isNotBlank()) {
                    Text(warehouseName, style = MaterialTheme.typography.titleMedium)
                }
                Text(
                    "Dispatch routing uses this address. Coordinates stay hidden from daily ops screens.",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                AddressLocationField(
                    geocodeApi = geocodeApi,
                    value = location,
                    onValueChange = { location = it },
                    label = "Depot address",
                )
                if (!saveMessage.isNullOrBlank()) {
                    Text(
                        saveMessage!!,
                        color = if (saveMessage!!.contains("saved", ignoreCase = true)) {
                            MaterialTheme.colorScheme.primary
                        } else {
                            MaterialTheme.colorScheme.error
                        },
                    )
                }
                Button(onClick = { save() }, enabled = !saving) {
                    Text(if (saving) "Saving…" else "Save location")
                }
            }
        }
    }
}
