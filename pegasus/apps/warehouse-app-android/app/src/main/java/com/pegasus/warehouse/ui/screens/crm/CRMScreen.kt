package com.pegasus.warehouse.ui.screens.crm

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.pegasus.warehouse.data.model.Retailer
import com.pegasus.warehouse.data.model.UpdateRetailerReceivingWindowRequest
import com.pegasus.warehouse.data.remote.WarehouseApi
import com.pegasus.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import java.text.NumberFormat
import java.util.Locale

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CRMScreen(
    api: WarehouseApi,
    onBack: () -> Unit,
) {
    var retailers by remember { mutableStateOf<List<Retailer>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var editing by remember { mutableStateOf<Retailer?>(null) }
    var windowOpen by remember { mutableStateOf("") }
    var windowClose by remember { mutableStateOf("") }
    var saving by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val fmt = remember { NumberFormat.getInstance(Locale("uz", "UZ")) }

    fun load() {
        loading = true; error = null
        scope.launch {
            try {
                val resp = api.getRetailers()
                if (resp.isSuccessful && resp.body() != null) retailers = resp.body()!!.retailers
                else error = "Failed (${resp.code()})"
            } catch (e: Exception) { error = e.message ?: "Network error" }
            finally { loading = false }
        }
    }

    LaunchedEffect(Unit) { load() }

    if (editing != null) {
        AlertDialog(
            onDismissRequest = { if (!saving) editing = null },
            title = { Text("Receiving window") },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                    Text(editing!!.displayName, style = MaterialTheme.typography.bodySmall)
                    OutlinedTextField(
                        value = windowOpen,
                        onValueChange = { windowOpen = it },
                        label = { Text("Open (HH:MM)") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                    )
                    OutlinedTextField(
                        value = windowClose,
                        onValueChange = { windowClose = it },
                        label = { Text("Close (HH:MM)") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
            },
            confirmButton = {
                TextButton(
                    enabled = !saving,
                    onClick = {
                        val target = editing ?: return@TextButton
                        saving = true
                        scope.launch {
                            try {
                                val resp = api.updateRetailerReceivingWindow(
                                    target.retailerId,
                                    UpdateRetailerReceivingWindowRequest(windowOpen, windowClose),
                                )
                                if (resp.isSuccessful) {
                                    editing = null
                                    load()
                                } else error = "Save failed (${resp.code()})"
                            } catch (e: Exception) {
                                error = e.message ?: "Network error"
                            } finally {
                                saving = false
                            }
                        }
                    },
                ) { Text(if (saving) "Saving…" else "Save") }
            },
            dismissButton = {
                TextButton(enabled = !saving, onClick = { editing = null }) { Text("Cancel") }
            },
        )
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Retailers") },
                navigationIcon = { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") } },
                actions = { IconButton(onClick = { load() }) { Icon(Icons.Default.Refresh, "Refresh") } },
            )
        },
    ) { innerPadding ->
        when {
            loading -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
            error != null -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(error!!, color = MaterialTheme.colorScheme.error)
                    Spacer(Modifier.height(PegasusSpacing.lg))
                    Button(onClick = { load() }) { Text("Retry") }
                }
            }
            retailers.isEmpty() -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                Text("No retailers", color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            else -> LazyColumn(
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            ) {
                items(retailers, key = { it.retailerId }) { r ->
                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                        Column(modifier = Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                            Text(r.displayName, style = MaterialTheme.typography.titleSmall)
                            Text(
                                "${r.totalOrders} orders · ${fmt.format(r.totalRevenue)} UZS",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                            Text(
                                "Receiving: ${r.receivingWindowLabel}",
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                            TextButton(onClick = {
                                editing = r
                                windowOpen = r.receivingWindowOpen
                                windowClose = r.receivingWindowClose
                            }) { Text("Edit window") }
                        }
                    }
                }
            }
        }
    }
}
