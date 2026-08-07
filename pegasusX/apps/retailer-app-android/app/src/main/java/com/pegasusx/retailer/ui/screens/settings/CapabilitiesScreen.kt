package com.pegasusx.retailer.ui.screens.settings

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
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
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.google.gson.JsonObject
import com.pegasusx.retailer.data.api.PegasusApi
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.launch
import javax.inject.Inject
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.ViewModel

data class CapabilityPackUi(
    val id: String,
    val name: String,
    val description: String,
    val enabled: Boolean,
    val alwaysOn: Boolean,
    val hardDeps: List<String>,
    val softDeps: List<String>,
)

@HiltViewModel
class CapabilitiesViewModel @Inject constructor(
    val api: PegasusApi,
) : ViewModel()

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CapabilitiesScreen(
    onNavigateBack: () -> Unit,
    viewModel: CapabilitiesViewModel = hiltViewModel(),
) {
    val scope = rememberCoroutineScope()
    var packs by remember { mutableStateOf<List<CapabilityPackUi>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var busyId by remember { mutableStateOf<String?>(null) }
    var banner by remember { mutableStateOf<String?>(null) }

    fun reload() {
        scope.launch {
            loading = true
            error = null
            try {
                val el = viewModel.api.getCapabilities()
                val arr = el.asJsonObject.getAsJsonArray("packs")
                packs = arr.mapNotNull { item ->
                    val o = item.asJsonObject
                    CapabilityPackUi(
                        id = o.get("id")?.asString.orEmpty(),
                        name = o.get("name")?.asString.orEmpty(),
                        description = o.get("description")?.asString.orEmpty(),
                        enabled = o.get("enabled")?.asBoolean == true,
                        alwaysOn = o.get("always_on")?.asBoolean == true,
                        hardDeps = o.getAsJsonArray("hard_deps")?.map { it.asString }.orEmpty(),
                        softDeps = o.getAsJsonArray("soft_deps")?.map { it.asString }.orEmpty(),
                    )
                }
            } catch (e: Exception) {
                error = e.message ?: "Failed to load"
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) { reload() }

    fun enable(packId: String, acceptSoft: Boolean = false, enableDeps: Boolean = false) {
        scope.launch {
            busyId = packId
            banner = null
            try {
                val body = mapOf(
                    "accept_soft_deps" to acceptSoft,
                    "enable_deps" to enableDeps,
                    "config" to emptyMap<String, Any>(),
                )
                val res = viewModel.api.enableCapability(
                    packId = packId,
                    body = body,
                    idempotencyKey = "cap-en-$packId-${System.currentTimeMillis()}",
                )
                if (res is JsonObject && res.get("status")?.asString == "BLOCKED") {
                    banner = res.get("message")?.asString ?: "Blocked"
                } else if (res is JsonObject && res.get("status")?.asString == "WARN") {
                    banner = (res.get("message")?.asString ?: "Recommended deps missing") +
                        " — tap Enable again after reviewing, or enable with deps from desktop."
                } else {
                    banner = "$packId enabled"
                    reload()
                }
            } catch (e: Exception) {
                // Retrofit may throw on 409 — surface message
                banner = e.message ?: "Enable failed (check hard deps)"
            } finally {
                busyId = null
            }
        }
    }

    fun disable(packId: String) {
        scope.launch {
            busyId = packId
            banner = null
            try {
                viewModel.api.disableCapability(
                    packId = packId,
                    idempotencyKey = "cap-dis-$packId-${System.currentTimeMillis()}",
                )
                banner = "$packId disabled"
                reload()
            } catch (e: Exception) {
                banner = e.message ?: "Disable failed"
            } finally {
                busyId = null
            }
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Store capabilities") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
            )
        },
    ) { padding ->
        Column(
            Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(horizontal = 16.dp),
        ) {
            Text(
                "Solo shops need only Core. Enable packs as you grow — dependencies are enforced.",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Spacer(Modifier.height(8.dp))
            banner?.let {
                Text(it, color = MaterialTheme.colorScheme.primary, style = MaterialTheme.typography.bodySmall)
                Spacer(Modifier.height(8.dp))
            }
            if (loading && packs.isEmpty()) {
                CircularProgressIndicator(Modifier.align(Alignment.CenterHorizontally))
            }
            error?.let {
                Text(it, color = MaterialTheme.colorScheme.error)
                Button(onClick = { reload() }) { Text("Retry") }
            }
            LazyColumn(
                contentPadding = PaddingValues(vertical = 8.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                items(packs, key = { it.id }) { pack ->
                    Card(colors = CardDefaults.cardColors()) {
                        Column(Modifier.padding(14.dp)) {
                            Text(pack.name, style = MaterialTheme.typography.titleMedium)
                            Text(pack.id, style = MaterialTheme.typography.labelSmall)
                            Spacer(Modifier.height(4.dp))
                            Text(pack.description, style = MaterialTheme.typography.bodySmall)
                            if (pack.hardDeps.isNotEmpty()) {
                                Text(
                                    stringResource(R.string.mobile_retailer_ui_requires_jointostring, pack.hardDeps.joinToString()),
                                    style = MaterialTheme.typography.labelSmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            }
                            Spacer(Modifier.height(10.dp))
                            Row(
                                Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.End,
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                when {
                                    pack.alwaysOn -> Text("Always on", style = MaterialTheme.typography.labelMedium)
                                    pack.enabled -> OutlinedButton(
                                        onClick = { disable(pack.id) },
                                        enabled = busyId != pack.id,
                                    ) { Text(if (busyId == pack.id) "…" else "Disable") }
                                    else -> Button(
                                        onClick = { enable(pack.id, acceptSoft = true, enableDeps = true) },
                                        enabled = busyId != pack.id,
                                    ) { Text(if (busyId == pack.id) "…" else "Enable") }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
