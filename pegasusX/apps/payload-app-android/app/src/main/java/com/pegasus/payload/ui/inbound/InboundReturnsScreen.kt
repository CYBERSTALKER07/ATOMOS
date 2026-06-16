package com.pegasus.payload.ui.inbound

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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
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
import androidx.compose.material3.Tab
import androidx.compose.material3.TabRow
import androidx.compose.material3.TextButton
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
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.pegasus.barcode.EanBarcodeScannerPreview
import com.pegasus.payload.util.PayloadIdempotencyKeys
import kotlinx.coroutines.launch

private data class InboundRow(
    val returnId: String,
    val productName: String,
    val expectedQty: Int,
    val receivedQty: Int,
    val reason: String,
    val physicalStatus: String,
    val driverName: String,
    val barcode: String = "",
)

private enum class InboundTab { Queue, History }

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun InboundReturnsScreen(
    onBack: () -> Unit,
    viewModel: InboundReturnsViewModel = hiltViewModel(),
) {
    val api = viewModel.api
    val online by viewModel.online.collectAsStateWithLifecycle()
    var rows by remember { mutableStateOf<List<InboundRow>>(emptyList()) }
    var history by remember { mutableStateOf<List<InboundRow>>(emptyList()) }
    var tab by remember { mutableStateOf(InboundTab.Queue) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var barcode by remember { mutableStateOf("") }
    var sessionId by remember { mutableStateOf<String?>(null) }
    var scanning by remember { mutableStateOf(false) }
    var scannerEnabled by remember { mutableStateOf(true) }
    var selected by remember { mutableStateOf(setOf<String>()) }
    var statusMessage by remember { mutableStateOf<String?>(null) }
    var queuedScans by remember { mutableStateOf(0) }
    val scope = rememberCoroutineScope()

    fun parseRows(body: Map<String, Any>?): List<InboundRow> {
        @Suppress("UNCHECKED_CAST")
        val data = body?.get("data") as? List<Map<String, Any>> ?: return emptyList()
        return data.mapNotNull { m ->
            val returnId = m["return_id"] as? String ?: return@mapNotNull null
            InboundRow(
                returnId = returnId,
                productName = m["product_name"] as? String ?: "",
                expectedQty = (m["expected_qty"] as? Number)?.toInt() ?: 0,
                receivedQty = (m["received_qty"] as? Number)?.toInt() ?: 0,
                reason = m["reason"] as? String ?: "",
                physicalStatus = m["physical_status"] as? String ?: "",
                driverName = m["driver_name"] as? String ?: "",
                barcode = m["barcode"] as? String ?: "",
            )
        }
    }

    fun load() {
        loading = true
        error = null
        scope.launch {
            try {
                val resp = api.getInboundReturns()
                if (resp.isSuccessful) rows = parseRows(resp.body())
                else error = "Failed (${resp.code()})"
                val hist = api.getReturnsHistory()
                if (hist.isSuccessful) history = parseRows(hist.body())
                queuedScans = viewModel.queuedScanCount()
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                loading = false
            }
        }
    }

    suspend fun ensureSession(): String {
        sessionId?.let { return it }
        val resp = api.createInboundSession(emptyMap())
        if (!resp.isSuccessful) throw IllegalStateException("session_failed")
        val sid = resp.body()?.get("session_id") as? String ?: throw IllegalStateException("session_failed")
        sessionId = sid
        return sid
    }

    fun submitScan(trimmed: String) {
        if (trimmed.isBlank() || scanning) return
        scanning = true
        scope.launch {
            try {
                if (!online) {
                    viewModel.enqueueOfflineScan(trimmed, sessionId)
                    barcode = ""
                    statusMessage = "Scan queued (offline)"
                    queuedScans = viewModel.queuedScanCount()
                    return@launch
                }
                val sid = ensureSession()
                val key = PayloadIdempotencyKeys.key("inbound-scan", "$trimmed-$sid")
                val resp = api.scanInboundReturn(
                    mapOf("barcode" to trimmed, "qty" to 1, "session_id" to sid),
                    key,
                )
                if (!resp.isSuccessful) throw IllegalStateException("scan_failed")
                barcode = ""
                scannerEnabled = false
                statusMessage = "Scan recorded"
                load()
                kotlinx.coroutines.delay(1500)
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
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
                actions = {
                    IconButton(onClick = { load() }) {
                        Icon(Icons.Default.Refresh, contentDescription = "Refresh")
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
                    Spacer(Modifier.height(16.dp))
                    Button(onClick = { load() }) { Text("Retry") }
                }
            }
            else -> Column(Modifier.fillMaxSize().padding(innerPadding)) {
                TabRow(selectedTabIndex = tab.ordinal) {
                    Tab(
                        selected = tab == InboundTab.Queue,
                        onClick = { tab = InboundTab.Queue },
                        text = { Text("Gate queue") },
                    )
                    Tab(
                        selected = tab == InboundTab.History,
                        onClick = { tab = InboundTab.History },
                        text = { Text("History") },
                    )
                }
                if (tab == InboundTab.Queue) {
                EanBarcodeScannerPreview(
                    enabled = scannerEnabled && !scanning,
                    onBarcode = { scanned -> submitScan(scanned.trim()) },
                )
                Row(Modifier.fillMaxWidth().padding(16.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
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
                if (queuedScans > 0) {
                    Text(
                        "$queuedScans scan(s) queued offline",
                        modifier = Modifier.padding(horizontal = 16.dp),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.tertiary,
                    )
                }
                statusMessage?.let {
                    Text(it, modifier = Modifier.padding(horizontal = 16.dp), style = MaterialTheme.typography.bodySmall)
                }
                if (selected.isNotEmpty()) {
                    Row(
                        Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
                        horizontalArrangement = Arrangement.SpaceBetween,
                    ) {
                        TextButton(onClick = {
                            scope.launch {
                                try {
                                    val sid = ensureSession()
                                    val lines = selected.map { mapOf("return_id" to it, "disposition" to "RESTOCK") }
                                    val key = PayloadIdempotencyKeys.key("inbound-confirm", "RESTOCK-${selected.sorted().joinToString(",")}")
                                    api.confirmInboundReturns(mapOf("lines" to lines, "session_id" to sid), key)
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
                                    val lines = selected.map { mapOf("return_id" to it, "disposition" to "WRITE_OFF") }
                                    val key = PayloadIdempotencyKeys.key("inbound-confirm", "WRITE_OFF-${selected.sorted().joinToString(",")}")
                                    api.confirmInboundReturns(mapOf("lines" to lines, "session_id" to sid), key)
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
                val visible = if (tab == InboundTab.Queue) rows else history
                if (visible.isEmpty()) {
                    Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                        Text(
                            if (tab == InboundTab.Queue) "No returns at gate" else "No completed receives yet",
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                } else {
                    LazyColumn(
                        contentPadding = PaddingValues(16.dp),
                        verticalArrangement = Arrangement.spacedBy(12.dp),
                        modifier = Modifier.fillMaxSize(),
                    ) {
                        items(visible, key = { it.returnId }) { row ->
                            val checked = selected.contains(row.returnId)
                            ElevatedCard(
                                modifier = Modifier.fillMaxWidth(),
                                onClick = {
                                    if (tab == InboundTab.Queue) {
                                        selected = if (checked) selected - row.returnId else selected + row.returnId
                                    }
                                },
                            ) {
                                Column(Modifier.padding(16.dp)) {
                                    Row(verticalAlignment = Alignment.CenterVertically) {
                                        Text(row.productName, style = MaterialTheme.typography.titleSmall, modifier = Modifier.weight(1f))
                                        AssistChip(onClick = {}, label = { Text(row.physicalStatus) })
                                    }
                                    Spacer(Modifier.height(4.dp))
                                    Text(
                                        "Qty ${row.receivedQty}/${row.expectedQty} · ${row.reason}",
                                        style = MaterialTheme.typography.bodySmall,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                                    )
                                    if (row.barcode.isNotBlank()) {
                                        Text(
                                            "EAN ${row.barcode}",
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
