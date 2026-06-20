package com.pegasusx.factory.ui.screens.setup

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
import com.pegasusx.factory.data.model.FactoryLocationPatchRequest
import com.pegasusx.factory.data.model.FactorySetupRequest
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.data.remote.GeocodeApi
import com.pegasusx.factory.data.remote.TokenHolder
import com.pegasusx.factory.ui.components.AddressLocationField
import com.pegasusx.factory.ui.components.AddressLocationValue
import com.pegasusx.factory.ui.theme.PegasusSpacing
import com.pegasusx.factory.util.GeocodeLocationSupport
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LocationSetupScreen(
    api: FactoryApi,
    geocodeApi: GeocodeApi,
    onComplete: () -> Unit,
) {
    val hasAssignedFactory = TokenHolder.hasAssignedFactory
    var factoryName by remember { mutableStateOf("") }
    var facilityType by remember { mutableStateOf("MANUFACTURING") }
    var location by remember { mutableStateOf(AddressLocationValue()) }
    var loading by remember { mutableStateOf(hasAssignedFactory) }
    var submitting by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(hasAssignedFactory) {
        if (!hasAssignedFactory) return@LaunchedEffect
        loading = true
        runCatching { api.getFactoryLocation() }
            .onSuccess { resp ->
                val body = resp.body()
                if (resp.isSuccessful && body != null) {
                    factoryName = body.name
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

                if (hasAssignedFactory) {
                    val resp = api.patchFactoryLocation(
                        FactoryLocationPatchRequest(
                            address = resolved.address,
                            placeId = resolved.placeId,
                            lat = resolved.lat,
                            lng = resolved.lng,
                        ),
                    )
                    if (!resp.isSuccessful) {
                        error = "Setup failed (${resp.code()})"
                        submitting = false
                        return@launch
                    }
                    runCatching { api.refreshToken() }
                        .onSuccess { refreshResp ->
                            val auth = refreshResp.body()
                            if (refreshResp.isSuccessful && auth != null) {
                                TokenHolder.token = auth.token
                                TokenHolder.refreshToken = auth.refreshToken
                                if (auth.factoryId.isNotBlank()) TokenHolder.factoryId = auth.factoryId
                            }
                        }
                } else {
                    if (factoryName.trim().length < 3) {
                        error = "Factory name is required."
                        submitting = false
                        return@launch
                    }
                    val resp = api.setupFactory(
                        FactorySetupRequest(
                            factoryName = factoryName.trim(),
                            address = resolved.address,
                            placeId = resolved.placeId,
                            lat = resolved.lat,
                            lng = resolved.lng,
                            facilityType = facilityType,
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
                    if (body.factoryId.isNotBlank()) TokenHolder.factoryId = body.factoryId
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
        topBar = { TopAppBar(title = { Text("Factory location") }) },
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
                if (hasAssignedFactory) {
                    "Confirm or update your facility address. Changes sync with supply routing and loading bay operations."
                } else {
                    "Name your factory and set the facility address to start operations."
                },
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            if (!hasAssignedFactory) {
                OutlinedTextField(
                    value = factoryName,
                    onValueChange = { factoryName = it },
                    label = { Text("Factory name") },
                    modifier = Modifier.fillMaxWidth(),
                    singleLine = true,
                )
            } else if (factoryName.isNotBlank()) {
                Text(factoryName, style = MaterialTheme.typography.titleMedium)
            }
            AddressLocationField(
                geocodeApi = geocodeApi,
                value = location,
                onValueChange = { location = it },
                label = "Factory address",
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
