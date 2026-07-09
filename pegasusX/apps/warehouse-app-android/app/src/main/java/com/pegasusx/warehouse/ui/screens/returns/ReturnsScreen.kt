package com.pegasusx.warehouse.ui.screens.returns

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.AssistChip
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.Tab
import androidx.compose.material3.TabRow
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
import com.pegasus.barcode.EanBarcodeScannerPreview
import com.pegasusx.warehouse.data.model.InboundReturnRow
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import com.pegasusx.warehouse.util.WarehouseIdempotencyKeys
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

private enum class ReturnsTab { Queue, History }

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ReturnsScreen(
    api: WarehouseApi,
    onBack: (() -> Unit)? = null,
) {
    var items by remember { mutableStateOf<List<InboundReturnRow>>(emptyList()) }
    var history by remember { mutableStateOf<List<InboundReturnRow>>(emptyList()) }
    var tab by remember { mutableStateOf(ReturnsTab.Queue) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var barcode by remember { mutableStateOf("") }
    var sessionId by remember { mutableStateOf<String?>(null) }
    var scanning by remember { mutableStateOf(false) }
    var scannerEnabled by remember { mutableStateOf(true) }
    var selected by remember { mutableStateOf(setOf<String>()) }
    var statusMessage by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        loading = true
        error = null
        scope.launch {
            try {
                val inbound = api.getInboundReturns()
                if (inbound.isSuccessful && inbound.body() != null) items = inbound.body()!!.data
                else error = "Failed (${inbound.code()})"
                val hist = api.getReturnsHistory()
                if (hist.isSuccessful && hist.body() != null) history = hist.body()!!.data
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                loading = false
            }
        }
    }

    suspend fun ensureSession(): String {
        sessionId?.let { return it }
        val resp = api.createInboundSession()
        if (!resp.isSuccessful) throw IllegalStateException("session_failed")
        val sid = resp.body()?.get("session_id")?.asString ?: throw IllegalStateException("session_failed")
        sessionId = sid
        return sid
    }

    fun submitScan(trimmed: String) {
        if (trimmed.isBlank() || scanning) return
        scanning = true
        scope.launch {
            try {
                val sid = ensureSession()
                val body = JsonObject().apply {
                    addProperty("barcode", trimmed)
                    addProperty("qty", 1)
                    addProperty("session_id", sid)
                }
                val key = WarehouseIdempotencyKeys.inboundScan(trimmed, sid)
                val resp = api.scanInboundReturn(key, body)
                if (!resp.isSuccessful) throw IllegalStateException("scan_failed")
                barcode = ""
                scannerEnabled = false
                statusMessage = "Scan recorded"
                load()
                delay(1500)
                scannerEnabled = true
            } catch (e: Exception) {
                statusMessage = e.message
            } finally {
                scanning = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Inbound Returns") },
                navigationIcon = {
                    if (onBack != null) {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back")
                        }
                    }
                },
                actions = {
                    IconButton(onClick = { load() }) {
                        Icon(Icons.Default.Refresh, "Refresh")
                    }
                },
            )
        },
    ) { innerPadding ->
        when {
            loading -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                CircularProgressIndicator()
            }
            error != null -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(error!!, color = MaterialTheme.colorScheme.error)
                    Spacer(Modifier.height(PegasusSpacing.lg))
                    Button(onClick = { load() }) { Text("Retry") }
                }
            }
            else -> Column(Modifier.fillMaxSize().padding(innerPadding)) {
                TabRow(selectedTabIndex = tab.ordinal) {
                    Tab(
                        selected = tab == ReturnsTab.Queue,
                        onClick = { tab = ReturnsTab.Queue },
                        text = { Text("Gate queue") },
                    )
                    Tab(
                        selected = tab == ReturnsTab.History,
                        onClick = { tab = ReturnsTab.History },
                        text = { Text("History") },
                    )
                }
                if (tab == ReturnsTab.Queue) {
                EanBarcodeScannerPreview(
                    enabled = scannerEnabled && !scanning,
                    onBarcode = { scanned -> submitScan(scanned.trim()) },
                )
                Row(
                    Modifier.fillMaxWidth().padding(PegasusSpacing.lg),
                    horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                ) {
                    OutlinedTextField(
                        value = barcode,
                        onValueChange = { barcode = it },
                        modifier = Modifier.weight(1f),
                        label = { Text("EAN barcode") },
                        singleLine = true,
                    )
                    Button(
                        onClick = { submitScan(barcode.trim()) },
                        enabled = !scanning && barcode.isNotBlank(),
                    ) { Text(if (scanning) "…" else "Scan") }
                }
                statusMessage?.let {
                    Text(
                        it,
                        modifier = Modifier.padding(horizontal = PegasusSpacing.lg),
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
                if (selected.isNotEmpty()) {
                    Row(
                        Modifier.fillMaxWidth().padding(horizontal = PegasusSpacing.lg, vertical = PegasusSpacing.sm),
                        horizontalArrangement = Arrangement.SpaceBetween,
                    ) {
                        TextButton(onClick = {
                            scope.launch {
                                try {
                                    val sid = ensureSession()
                                    val lines = selected.map { id ->
                                        JsonObject().apply {
                                            addProperty("return_id", id)
                                            addProperty("disposition", "RESTOCK")
                                        }
                                    }
                                    val body = JsonObject().apply {
                                        addProperty("session_id", sid)
                                        add("lines", com.google.gson.JsonArray().apply { lines.forEach { add(it) } })
                                    }
                                    val key = WarehouseIdempotencyKeys.inboundConfirm(selected.toList(), "RESTOCK")
                                    api.confirmInboundReturns(key, body)
                                    selected = emptySet()
                                    statusMessage = "RESTOCK confirmed"
                                    load()
                                } catch (e: Exception) {
                                    statusMessage = e.message
                                }
                            }
                        }) { Text("Restock (${selected.size})") }
                        TextButton(onClick = {
                            scope.launch {
                                try {
                                    val sid = ensureSession()
                                    val lines = selected.map { id ->
                                        JsonObject().apply {
                                            addProperty("return_id", id)
                                            addProperty("disposition", "WRITE_OFF")
                                        }
                                    }
                                    val body = JsonObject().apply {
                                        addProperty("session_id", sid)
                                        add("lines", com.google.gson.JsonArray().apply { lines.forEach { add(it) } })
                                    }
                                    val key = WarehouseIdempotencyKeys.inboundConfirm(selected.toList(), "WRITE_OFF")
                                    api.confirmInboundReturns(key, body)
                                    selected = emptySet()
                                    statusMessage = "WRITE_OFF confirmed"
                                    load()
                                } catch (e: Exception) {
                                    statusMessage = e.message
                                }
                            }
                        }) { Text("Write off") }
                    }
                }
                }
                val visible = if (tab == ReturnsTab.Queue) items else history
                if (visible.isEmpty()) {
                    Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                        Text(
                            if (tab == ReturnsTab.Queue) "No returns at gate" else "No completed receives yet",
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                } else {
                    LazyVerticalGrid(
                        columns = GridCells.Adaptive(minSize = 340.dp),
                        contentPadding = PaddingValues(PegasusSpacing.lg),
                        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                        modifier = Modifier.fillMaxSize(),
                    ) {
                        items(visible, key = { it.returnId }) { r ->
                            val checked = selected.contains(r.returnId)
                            ElevatedCard(
                                modifier = Modifier.fillMaxWidth(),
                                onClick = {
                                    if (tab == ReturnsTab.Queue) {
                                        selected = if (checked) selected - r.returnId else selected + r.returnId
                                    }
                                },
                            ) {
                                Column(modifier = Modifier.padding(PegasusSpacing.lg)) {
                                    Row(verticalAlignment = Alignment.CenterVertically) {
                                        Text(
                                            r.productName,
                                            style = MaterialTheme.typography.titleSmall,
                                            modifier = Modifier.weight(1f),
                                        )
                                        AssistChip(onClick = {}, label = { Text(r.physicalStatus) })
                                    }
                                    Spacer(Modifier.height(PegasusSpacing.xs))
                                    Text(
                                        "Qty: ${r.receivedQty}/${r.expectedQty} · ${r.reason} · ${r.driverName}",
                                        style = MaterialTheme.typography.bodySmall,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                                    )
                                    if (r.barcode.isNotBlank()) {
                                        Text(
                                            "EAN ${r.barcode}",
                                            style = MaterialTheme.typography.labelSmall,
                                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                                        )
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
