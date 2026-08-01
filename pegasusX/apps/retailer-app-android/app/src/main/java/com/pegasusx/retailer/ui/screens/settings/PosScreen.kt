package com.pegasusx.retailer.ui.screens.settings

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
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.ViewModel
import com.pegasusx.retailer.data.api.PegasusApi
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.launch
import javax.inject.Inject

data class PosCartLine(
    val sku: String,
    val name: String,
    val qty: Long,
    val unitPriceMinor: Long,
)

@HiltViewModel
class PosViewModel @Inject constructor(val api: PegasusApi) : ViewModel()

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PosScreen(
    onNavigateBack: () -> Unit,
    viewModel: PosViewModel = hiltViewModel(),
) {
    val scope = rememberCoroutineScope()
    var registerId by remember { mutableStateOf<String?>(null) }
    var sessionId by remember { mutableStateOf<String?>(null) }
    var banner by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    var sku by remember { mutableStateOf("") }
    var priceMajor by remember { mutableStateOf("0") }
    var qty by remember { mutableStateOf("1") }
    var lastSaleId by remember { mutableStateOf<String?>(null) }
    val cart = remember { mutableStateListOf<PosCartLine>() }

    val totalMinor = cart.sumOf { it.qty * it.unitPriceMinor }

    fun ensureRegister() {
        scope.launch {
            try {
                val regs = viewModel.api.getRegisters().asJsonObject.getAsJsonArray("items")
                if (regs != null && regs.size() > 0) {
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

    LaunchedEffect(Unit) { ensureRegister() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("POS") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
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
                    "Open till → add lines → complete sale (FLOOR stock). Manager void for large refunds.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            banner?.let { item { Text(it, color = MaterialTheme.colorScheme.primary) } }
            item {
                Card {
                    Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text(if (sessionId == null) "Session closed" else "Session open", style = MaterialTheme.typography.titleMedium)
                        if (sessionId == null) {
                            Button(enabled = !busy && registerId != null, onClick = {
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
                            }) { Text("Open session") }
                        } else {
                            OutlinedButton(enabled = !busy, onClick = {
                                scope.launch {
                                    busy = true
                                    try {
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
                            }) { Text("Close session") }
                        }
                    }
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
                            Text("${line.sku} × ${line.qty}")
                            Text("${line.qty * line.unitPriceMinor / 100.0}")
                        }
                    }
                }
                item {
                    Text("Total: ${totalMinor / 100.0}", style = MaterialTheme.typography.titleLarge)
                    Button(
                        enabled = !busy && cart.isNotEmpty(),
                        onClick = {
                            scope.launch {
                                busy = true
                                try {
                                    val sale = viewModel.api.createPosSale(
                                        body = mapOf(
                                            "session_id" to (sessionId ?: ""),
                                            "stock_bin" to "FLOOR",
                                            "lines" to cart.map {
                                                mapOf(
                                                    "sku" to it.sku,
                                                    "name" to it.name,
                                                    "qty" to it.qty,
                                                    "unit_price_minor" to it.unitPriceMinor,
                                                )
                                            },
                                            "tenders" to listOf(
                                                mapOf("method" to "CASH", "amount_minor" to totalMinor),
                                            ),
                                        ),
                                        idempotencyKey = "sale-${System.currentTimeMillis()}",
                                    ).asJsonObject
                                    lastSaleId = sale.get("sale_id")?.asString
                                    banner = "Sale ${sale.get("receipt_number")?.asString}"
                                    cart.clear()
                                } catch (e: Exception) {
                                    banner = e.message
                                } finally {
                                    busy = false
                                }
                            }
                        },
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text("Complete cash sale") }
                    if (lastSaleId != null) {
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
