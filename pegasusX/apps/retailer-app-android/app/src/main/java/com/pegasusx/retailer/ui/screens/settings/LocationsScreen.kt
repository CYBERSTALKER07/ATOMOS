package com.pegasusx.retailer.ui.screens.settings

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
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
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.ViewModel
import com.pegasusx.retailer.data.api.PegasusApi
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.launch
import javax.inject.Inject
import com.pegasusx.retailer.R

data class LocationUi(
    val id: String,
    val name: String,
    val address: String,
    val isPrimary: Boolean,
    val isActive: Boolean,
)

@HiltViewModel
class LocationsViewModel @Inject constructor(val api: PegasusApi) : ViewModel()

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LocationsScreen(
    onNavigateBack: () -> Unit,
    viewModel: LocationsViewModel = hiltViewModel(),
) {
    val scope = rememberCoroutineScope()
    var items by remember { mutableStateOf<List<LocationUi>>(emptyList()) }
    var activeId by remember { mutableStateOf("") }
    var loading by remember { mutableStateOf(true) }
    var banner by remember { mutableStateOf<String?>(null) }
    var name by remember { mutableStateOf("") }
    var address by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }

    fun reload() {
        scope.launch {
            loading = true
            try {
                val el = viewModel.api.getLocations().asJsonObject
                activeId = el.get("active_location_id")?.asString.orEmpty()
                items = el.getAsJsonArray("items")?.map { item ->
                    val o = item.asJsonObject
                    LocationUi(
                        id = o.get("location_id")?.asString.orEmpty(),
                        name = o.get("name")?.asString.orEmpty(),
                        address = o.get("delivery_address")?.asString.orEmpty(),
                        isPrimary = o.get("is_primary")?.asBoolean == true,
                        isActive = o.get("is_active")?.asBoolean != false,
                    )
                }.orEmpty()
            } catch (e: Exception) {
                banner = e.message
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) { reload() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Locations") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
            )
        },
    ) { padding ->
        LazyColumn(
            Modifier.fillMaxSize().padding(padding).padding(horizontal = 16.dp),
            contentPadding = PaddingValues(vertical = 12.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item {
                Text(
                    "Primary store is created automatically from your shop profile. Add branches for multi-location.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            banner?.let { item { Text(it, color = MaterialTheme.colorScheme.primary) } }
            item {
                Card {
                    Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text("Add branch", style = MaterialTheme.typography.titleMedium)
                        OutlinedTextField(value = name, onValueChange = { name = it }, label = { Text("Name") }, modifier = Modifier.fillMaxWidth())
                        OutlinedTextField(value = address, onValueChange = { address = it }, label = { Text("Address") }, modifier = Modifier.fillMaxWidth())
                        Button(enabled = !busy, onClick = {
                            scope.launch {
                                busy = true
                                try {
                                    viewModel.api.createLocation(
                                        body = mapOf(
                                            "name" to name,
                                            "delivery_address" to address,
                                        ),
                                        idempotencyKey = "loc-${System.currentTimeMillis()}",
                                    )
                                    name = ""; address = ""
                                    banner = "Created"
                                    reload()
                                } catch (e: Exception) {
                                    banner = e.message
                                } finally {
                                    busy = false
                                }
                            }
                        }) { Text(if (busy) "…" else "Create") }
                    }
                }
            }
            items(items, key = { it.id }) { loc ->
                Card {
                    Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                        Text(loc.name, style = MaterialTheme.typography.titleSmall)
                        Text(loc.address.ifBlank { "No address" }, style = MaterialTheme.typography.bodySmall)
                        if (loc.isPrimary) Text("Primary", style = MaterialTheme.typography.labelSmall)
                        if (activeId == loc.id) Text("Active for checkout", color = MaterialTheme.colorScheme.primary)
                        if (loc.isActive && activeId != loc.id) {
                            OutlinedButton(onClick = {
                                scope.launch {
                                    try {
                                        viewModel.api.switchLocation(mapOf("location_id" to loc.id))
                                        activeId = loc.id
                                        banner = "Switched active branch"
                                    } catch (e: Exception) {
                                        banner = e.message
                                    }
                                }
                            }) { Text("Use for checkout") }
                        }
                        if (loc.isActive && !loc.isPrimary) {
                            OutlinedButton(onClick = {
                                scope.launch {
                                    try {
                                        viewModel.api.setPrimaryLocation(
                                            locationId = loc.id,
                                            idempotencyKey = "prim-${loc.id}-${System.currentTimeMillis()}",
                                        )
                                        banner = "Primary updated"
                                        reload()
                                    } catch (e: Exception) {
                                        banner = e.message
                                    }
                                }
                            }) { Text("Set primary") }
                        }
                    }
                }
            }
        }
    }
}
