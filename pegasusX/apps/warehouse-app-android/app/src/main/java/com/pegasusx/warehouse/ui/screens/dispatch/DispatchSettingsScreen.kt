package com.pegasusx.warehouse.ui.screens.dispatch

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.pegasusx.warehouse.data.remote.WarehouseOperationsRepository
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import com.pegasusx.warehouse.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DispatchSettingsScreen(
    opsRepository: WarehouseOperationsRepository,
    onBack: (() -> Unit)? = null,
) {
    var autoDispatch by remember { mutableStateOf<Boolean?>(null) }
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var saveMessage by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = opsRepository.getDispatchSettings()
                if (resp.isSuccessful) {
                    autoDispatch = resp.body()?.autoDispatchEnabled
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

    fun save(enabled: Boolean) {
        scope.launch {
            saving = true
            saveMessage = null
            try {
                val resp = opsRepository.patchDispatchSettings(enabled)
                if (resp.isSuccessful) {
                    autoDispatch = enabled
                    saveMessage = if (enabled) "Auto dispatch enabled" else "Auto dispatch disabled"
                } else {
                    saveMessage = "Update failed (${resp.code()})"
                }
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
                title = { Text("Dispatch settings") },
                navigationIcon = { if (onBack != null) { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back)) } } },
                actions = {
                    TextButton(onClick = { load() }) { Text("Refresh") }
                },
            )
        },
    ) { padding ->
        when {
            loading -> Box(
                Modifier.fillMaxSize().padding(padding),
                contentAlignment = Alignment.Center,
            ) { CircularProgressIndicator() }

            error != null -> Box(
                Modifier.fillMaxSize().padding(padding),
                contentAlignment = Alignment.Center,
            ) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(error!!, color = MaterialTheme.colorScheme.error)
                    Spacer(Modifier.height(PegasusSpacing.md))
                    Button(onClick = { load() }) { Text("Retry") }
                }
            }

            else -> Column(
                modifier = Modifier
                    .padding(padding)
                    .padding(PegasusSpacing.lg)
                    .fillMaxSize(),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                Text(
                    "Configure warehouse auto-dispatch policy for this node.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                saveMessage?.let {
                    Text(it, color = MaterialTheme.colorScheme.primary)
                }
                ElevatedCard(Modifier.fillMaxWidth()) {
                    Row(
                        modifier = Modifier.padding(PegasusSpacing.lg),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Column(Modifier.weight(1f)) {
                            Text("Auto dispatch", style = MaterialTheme.typography.titleMedium)
                            Text(
                                "When enabled, the AI worker may auto-assign pending orders.",
                                style = MaterialTheme.typography.bodySmall,
                            )
                        }
                        Switch(
                            checked = autoDispatch == true,
                            onCheckedChange = { save(it) },
                            enabled = !saving && autoDispatch != null,
                        )
                    }
                }
            }
        }
    }
}
