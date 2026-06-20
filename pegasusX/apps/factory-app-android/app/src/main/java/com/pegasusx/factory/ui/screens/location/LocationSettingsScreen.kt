package com.pegasusx.factory.ui.screens.location

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import com.pegasusx.factory.data.model.FactoryLocationPatchRequest
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.data.remote.GeocodeApi
import com.pegasusx.factory.ui.components.AddressLocationField
import com.pegasusx.factory.ui.components.AddressLocationValue
import com.pegasusx.factory.ui.theme.PegasusSpacing
import com.pegasusx.factory.util.GeocodeLocationSupport
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LocationSettingsScreen(
    api: FactoryApi,
    geocodeApi: GeocodeApi,
    onBack: (() -> Unit)? = null,
) {
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var saveMessage by remember { mutableStateOf<String?>(null) }
    var factoryName by remember { mutableStateOf("") }
    var location by remember { mutableStateOf(AddressLocationValue()) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = api.getFactoryLocation()
                if (resp.isSuccessful && resp.body() != null) {
                    val body = resp.body()!!
                    factoryName = body.name
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
                val resp = api.patchFactoryLocation(
                    FactoryLocationPatchRequest(
                        address = resolved.address,
                        placeId = resolved.placeId,
                        lat = resolved.lat,
                        lng = resolved.lng,
                    ),
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
                title = { Text("Factory location") },
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
                if (factoryName.isNotBlank()) {
                    Text(factoryName, style = MaterialTheme.typography.titleMedium)
                }
                Text(
                    "Supply routing uses this address. Coordinates stay hidden from daily ops screens.",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                AddressLocationField(
                    geocodeApi = geocodeApi,
                    value = location,
                    onValueChange = { location = it },
                    label = "Factory address",
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
