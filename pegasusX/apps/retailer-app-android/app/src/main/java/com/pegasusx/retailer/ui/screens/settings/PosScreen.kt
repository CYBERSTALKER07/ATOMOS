package com.pegasusx.retailer.ui.screens.settings

import androidx.compose.ui.res.stringResource

import android.content.Context
import android.net.ConnectivityManager
import android.net.NetworkCapabilities
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
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
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.ViewModel
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.local.PendingPosSaleEntity
import com.pegasusx.retailer.data.local.PendingPosSaleSync
import dagger.hilt.android.lifecycle.HiltViewModel
import java.util.UUID
import javax.inject.Inject
import kotlinx.coroutines.launch
import com.pegasusx.retailer.R
import com.pegasusx.retailer.data.json.*

data class PosCartLine(
    val sku: String,
    val name: String,
    val qty: Long,
    val unitPriceMinor: Long,
)

@HiltViewModel
class PosViewModel @Inject constructor(
    val api: PegasusApi,
    val posSync: PendingPosSaleSync,
) : ViewModel()

private fun Context.isOnline(): Boolean {
    val cm = getSystemService(Context.CONNECTIVITY_SERVICE) as? ConnectivityManager ?: return true
    val net = cm.activeNetwork ?: return false
    val caps = cm.getNetworkCapabilities(net) ?: return false
    return caps.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PosScreen(
    onNavigateBack: () -> Unit,
    viewModel: PosViewModel = hiltViewModel(),
) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    var registerId by remember { mutableStateOf<String?>(null) }
    var sessionId by remember { mutableStateOf<String?>(null) }
    var banner by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    var sku by remember { mutableStateOf("") }
    var priceMajor by remember { mutableStateOf("0") }
    var qty by remember { mutableStateOf("1") }
    var lastSaleId by remember { mutableStateOf<String?>(null) }
    var pending by remember { mutableStateOf<List<PendingPosSaleEntity>>(emptyList()) }
    val cart = remember { mutableStateListOf<PosCartLine>() }

    val totalMinor = cart.sumOf { it.qty * it.unitPriceMinor }
    val online = context.isOnline()

    fun refreshPending() {
        scope.launch {
            pending = viewModel.posSync.listActive()
        }
    }

    fun ensureRegister() {
        scope.launch {
            try {
                val regs = viewModel.api.getRegisters().asJsonObject.getAsJsonArray("items")
                if (regs != null && regs.size > 0) {
                    registerId = regs[0].asJsonObject.get("register_id")?.asString
                } else {
                    val created = viewModel.api.createRegister(
                        body = mapOf("label" to "Register 1"),
                        idempotencyKey = "reg-${System.currentTimeMillis()}",
                    ).asJsonObject
                    registerId = created.get("register_id")?.asString
                    banner = "Register created"
                }
            } catch (e: Exception) {
                banner = e.message
            }
        }
    }

    LaunchedEffect(Unit) {
        ensureRegister()
        refreshPending()
        if (online) {
            val (flushed, failed) = viewModel.posSync.flush()
            if (flushed > 0) banner = "Synced $flushed offline sale(s)"
            if (failed > 0 && flushed == 0) banner = "$failed offline sale(s) failed to sync"
            refreshPending()
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("POS") },
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
                    "Open till online. Cash sales work offline and sync on reconnect. Card requires network.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            if (!online) {
                item {
                    Text(
                        "Offline · cash queue active" +
                            if (pending.isNotEmpty()) " · ${pending.size} pending" else "",
                        color = MaterialTheme.colorScheme.tertiary,
                        style = MaterialTheme.typography.labelLarge,
                    )
                }
            }
            banner?.let { item { Text(it, color = MaterialTheme.colorScheme.primary) } }
            item {
                Card {
                    Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text(
                            if (sessionId == null) "Session closed" else "Session open",
                            style = MaterialTheme.typography.titleMedium,
                        )
                        if (sessionId == null) {
                            Button(
                                enabled = !busy && registerId != null && online,
                                onClick = {
                                    scope.launch {
                                        busy = true
                                        try {
                                            val sess = viewModel.api.openPosSession(
                                                body = mapOf(
                                                    "register_id" to (registerId ?: ""),
                                                    "opening_float_minor" to 0L,
                                                    "currency" to "UZS",
                                                ),
                                                idempotencyKey = "open-${System.currentTimeMillis()}",
                                            ).asJsonObject
                                            sessionId = sess.get("session_id")?.asString
                                            cart.clear()
                                            banner = "Session open"
                                        } catch (e: Exception) {
                                            banner = e.message
                                        } finally {
                                            busy = false
                                        }
                                    }
                                },
                            ) { Text(if (online) "Open session" else "Open needs network") }
                        } else {
                            OutlinedButton(
                                enabled = !busy && online,
                                onClick = {
                                    scope.launch {
                                        busy = true
                                        try {
                                            val active = viewModel.posSync.countActiveForSession(sessionId!!)
                                            if (active > 0) {
                                                banner = "Sync $active offline sale(s) before close"
                                                return@launch
                                            }
                                            viewModel.api.closePosSession(
                                                sessionId = sessionId!!,
                                                body = mapOf("closing_cash_minor" to 0L),
                                            )
                                            sessionId = null
                                            banner = "Session closed"
                                        } catch (e: Exception) {
                                            banner = e.message
                                        } finally {
                                            busy = false
                                        }
                                    }
                                },
                            ) { Text("Close session") }
                            if (pending.isNotEmpty() && online) {
                                Button(
                                    enabled = !busy,
                                    onClick = {
                                        scope.launch {
                                            busy = true
                                            try {
                                                val (flushed, failed) = viewModel.posSync.flush()
                                                banner =
                                                    "Synced $flushed" + if (failed > 0) ", $failed failed" else ""
                                                refreshPending()
                                            } finally {
                                                busy = false
                                            }
                                        }
                                    },
                                ) { Text(stringResource(R.string.mobile_retailer_ui_sync_size_offline_sale_s, pending.size)) }
                            }
                        }
                    }
                }
            }
            if (pending.isNotEmpty()) {
                item {
                    Text("Offline queue", style = MaterialTheme.typography.titleSmall)
                }
                items(pending, key = { it.id }) { p ->
                    Text(stringResource(R.string.mobile_retailer_ui_clientreceipt_status, p.clientReceipt, p.status) +
                            (p.lastError?.let { " · $it" } ?: ""),
                        style = MaterialTheme.typography.labelSmall,
                        color = if (p.status == "FAILED") {
                            MaterialTheme.colorScheme.error
                        } else {
                            MaterialTheme.colorScheme.onSurfaceVariant
                        },
                    )
                }
            }
            if (sessionId != null) {
                item {
                    Card {
                        Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                            OutlinedTextField(value = sku, onValueChange = { sku = it }, label = { Text("SKU") }, modifier = Modifier.fillMaxWidth())
                            OutlinedTextField(value = qty, onValueChange = { qty = it }, label = { Text("Qty") }, modifier = Modifier.fillMaxWidth())
                            OutlinedTextField(value = priceMajor, onValueChange = { priceMajor = it }, label = { Text("Price (major)") }, modifier = Modifier.fillMaxWidth())
                            Button(onClick = {
                                val unit = ((priceMajor.toDoubleOrNull() ?: 0.0) * 100).toLong()
                                val q = qty.toLongOrNull() ?: 1L
                                if (sku.isBlank()) return@Button
                                val existing = cart.indexOfFirst { it.sku == sku.trim() }
                                if (existing >= 0) {
                                    val old = cart[existing]
                                    cart[existing] = old.copy(qty = old.qty + q)
                                } else {
                                    cart.add(PosCartLine(sku.trim(), sku.trim(), q, unit))
                                }
                                sku = ""; qty = "1"
                            }) { Text("Add line") }
                        }
                    }
                }
                items(cart) { line ->
                    Card {
                        Row(Modifier.padding(14.dp).fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                            Text(stringResource(R.string.mobile_retailer_ui_sku_qty, line.sku, line.qty))
                            Text("${line.qty * line.unitPriceMinor / 100.0}")
                        }
                    }
                }
                item {
                    Text(stringResource(R.string.mobile_retailer_ui_total_n_0, totalMinor / 100.0), style = MaterialTheme.typography.titleLarge)
                    Button(
                        enabled = !busy && cart.isNotEmpty(),
                        onClick = {
                            scope.launch {
                                busy = true
                                try {
                                    val lines = cart.map {
                                        mapOf(
                                            "sku" to it.sku,
                                            "name" to it.name,
                                            "qty" to it.qty,
                                            "unit_price_minor" to it.unitPriceMinor,
                                        )
                                    }
                                    if (!context.isOnline()) {
                                        val entity = viewModel.posSync.enqueue(
                                            sessionId = sessionId!!,
                                            lines = lines,
                                            totalMinor = totalMinor,
                                        )
                                        lastSaleId = entity.clientSaleId
                                        banner = "Offline ${entity.clientReceipt} · will sync"
                                        cart.clear()
                                        refreshPending()
                                    } else {
                                        val clientSaleId = UUID.randomUUID().toString()
                                        val sale = viewModel.api.createPosSale(
                                            body = mapOf(
                                                "session_id" to (sessionId ?: ""),
                                                "stock_bin" to "FLOOR",
                                                "origin" to "online",
                                                "client_sale_id" to clientSaleId,
                                                "lines" to lines,
                                                "tenders" to listOf(
                                                    mapOf("method" to "CASH", "amount_minor" to totalMinor),
                                                ),
                                            ),
                                            idempotencyKey = "pos-sale:$clientSaleId",
                                        ).asJsonObject
                                        lastSaleId = sale.get("sale_id")?.asString
                                        banner = "Sale ${sale.get("receipt_number")?.asString}"
                                        cart.clear()
                                    }
                                } catch (e: Exception) {
                                    // Network fail → queue cash offline
                                    try {
                                        val lines = cart.map {
                                            mapOf(
                                                "sku" to it.sku,
                                                "name" to it.name,
                                                "qty" to it.qty,
                                                "unit_price_minor" to it.unitPriceMinor,
                                            )
                                        }
                                        val entity = viewModel.posSync.enqueue(
                                            sessionId = sessionId!!,
                                            lines = lines,
                                            totalMinor = totalMinor,
                                        )
                                        lastSaleId = entity.clientSaleId
                                        banner = "Queued offline ${entity.clientReceipt}"
                                        cart.clear()
                                        refreshPending()
                                    } catch (_: Exception) {
                                        banner = e.message
                                    }
                                } finally {
                                    busy = false
                                }
                            }
                        },
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Text(if (online) "Complete cash sale" else "Complete cash sale offline")
                    }
                    if (lastSaleId != null && online) {
                        OutlinedButton(onClick = {
                            scope.launch {
                                try {
                                    viewModel.api.voidPosSale(
                                        saleId = lastSaleId!!,
                                        body = mapOf("reason" to "mobile_void"),
                                    )
                                    banner = "Voided"
                                    lastSaleId = null
                                } catch (e: Exception) {
                                    banner = e.message
                                }
                            }
                        }) { Text("Void last sale") }
                    }
                }
            }
        }
    }
}
