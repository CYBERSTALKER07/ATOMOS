package com.pegasusx.supplier.ui.screens.crm

import androidx.compose.foundation.clickable
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
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
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
import com.pegasus.design.ui.PegasusLoadingState
import com.pegasus.design.ui.PegasusStateKind
import com.pegasus.design.ui.PegasusStatePane
import com.pegasusx.supplier.data.model.SupplierCRMRetailer
import com.pegasusx.supplier.data.model.SupplierCRMRetailerDetail
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import java.text.NumberFormat
import java.util.Locale

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SupplierCRMScreen(
    ops: SupplierOperationsRepository,
    onBack: () -> Unit,
) {
    var retailers by remember { mutableStateOf<List<SupplierCRMRetailer>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var selected by remember { mutableStateOf<SupplierCRMRetailerDetail?>(null) }
    val scope = rememberCoroutineScope()
    val fmt = remember { NumberFormat.getInstance(Locale("uz", "UZ")) }

    fun load() {
        loading = true
        error = null
        scope.launch {
            try {
                val resp = ops.getCrmRetailers()
                if (resp.isSuccessful) {
                    retailers = resp.body()?.retailers.orEmpty()
                } else {
                    error = if (resp.code() == 503) "CRM unavailable" else "Failed (${resp.code()})"
                    retailers = emptyList()
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
                retailers = emptyList()
            } finally {
                loading = false
            }
        }
    }

    fun loadDetail(id: String) {
        scope.launch {
            try {
                val resp = ops.getCrmRetailer(id)
                selected = if (resp.isSuccessful) resp.body() else null
            } catch (_: Exception) {
                selected = null
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("CRM") },
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
    ) { padding ->
        when {
            loading && retailers.isEmpty() -> PegasusLoadingState(
                title = "Loading retailers",
                body = "Supplier lifetime rollup (TotalMinor)",
                modifier = Modifier.fillMaxSize().padding(padding),
            )
            error != null && retailers.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "CRM unavailable",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier.fillMaxSize().padding(padding),
            )
            retailers.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Empty,
                headline = "No retailer orders yet",
                body = "Retailers appear here after they place orders with this supplier.",
                modifier = Modifier.fillMaxSize().padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier.fillMaxSize().padding(padding),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                items(retailers, key = { it.retailerId }) { row ->
                    ElevatedCard(
                        modifier = Modifier.fillMaxWidth().clickable { loadDetail(row.retailerId) },
                    ) {
                        Column(Modifier.padding(PegasusSpacing.lg)) {
                            Text(row.retailerName.ifBlank { "—" }, style = MaterialTheme.typography.titleSmall)
                            Text(
                                "${row.status} · ${row.orderCount} orders · ${fmt.format(row.lifetime)} minor",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                            if (row.lastOrderDate.isNotBlank()) {
                                Text(
                                    row.lastOrderDate,
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            }
                            if (row.email.isNotBlank()) {
                                Text(
                                    row.email,
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            }
                        }
                    }
                }
                selected?.let { detail ->
                    item {
                        ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                            Column(Modifier.padding(PegasusSpacing.lg)) {
                                Text(detail.retailerName, style = MaterialTheme.typography.titleSmall)
                                if (detail.email.isNotBlank()) {
                                    Text(detail.email, style = MaterialTheme.typography.bodySmall)
                                }
                                Text(
                                    "${detail.orders.size} recent orders",
                                    style = MaterialTheme.typography.bodySmall,
                                )
                                detail.orders.forEach { order ->
                                    Text(
                                        "${order.orderId.take(8)} · ${order.state} · ${order.amount} · ${order.itemCount} items",
                                        style = MaterialTheme.typography.bodySmall,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                                    )
                                    order.lines.forEach { line ->
                                        Text(
                                            "  ${line.productName.ifBlank { line.sku }} × ${line.qty} · ${line.amountMinor}",
                                            style = MaterialTheme.typography.bodySmall,
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
