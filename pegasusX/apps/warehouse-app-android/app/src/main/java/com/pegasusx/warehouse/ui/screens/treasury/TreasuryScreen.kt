package com.pegasusx.warehouse.ui.screens.treasury

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
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.Invoice
import com.pegasusx.warehouse.data.model.TreasuryOverview
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import java.text.NumberFormat
import java.util.Locale

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TreasuryScreen(
    api: WarehouseApi,
    onBack: () -> Unit,
) {
    var overview by remember { mutableStateOf<TreasuryOverview?>(null) }
    var invoices by remember { mutableStateOf<List<Invoice>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var tab by remember { mutableIntStateOf(0) }
    val scope = rememberCoroutineScope()
    val fmt = remember { NumberFormat.getInstance(Locale("uz", "UZ")) }

    fun load() {
        loading = true; error = null
        scope.launch {
            try {
                val oResp = api.getTreasuryOverview()
                val iResp = api.getInvoices()
                if (oResp.isSuccessful && oResp.body() != null) {
                    overview = oResp.body()!!
                } else {
                    val fResp = api.getOpsFinancials()
                    if (fResp.isSuccessful && fResp.body() != null) {
                        val financials = fResp.body()!!
                        overview = TreasuryOverview(
                            totalInvoiced = financials.totalRevenue,
                            totalPaid = financials.netPayout,
                            totalOutstanding = financials.cashPending,
                        )
                    }
                }
                if (iResp.isSuccessful && iResp.body() != null) invoices = iResp.body()!!.invoices
                if (overview == null) error = "Failed to load"
            } catch (e: Exception) { error = e.message ?: "Network error" }
            finally { loading = false }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Treasury") },
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
            else -> Column(modifier = Modifier.fillMaxSize().padding(innerPadding)) {
                TabRow(selectedTabIndex = tab) {
                    Tab(selected = tab == 0, onClick = { tab = 0 }, text = { Text("Overview") })
                    Tab(selected = tab == 1, onClick = { tab = 1 }, text = { Text("Invoices") })
                }
                when (tab) {
                    0 -> overview?.let { o ->
                        LazyColumn(contentPadding = PaddingValues(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md)) {
                            item {
                                Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md), modifier = Modifier.fillMaxWidth()) {
                                    KpiCard("Outstanding", "${fmt.format(o.totalOutstanding)} UZS", Modifier.weight(1f))
                                    KpiCard("Invoiced", "${fmt.format(o.totalInvoiced)} UZS", Modifier.weight(1f))
                                }
                            }
                            item {
                                Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md), modifier = Modifier.fillMaxWidth()) {
                                    KpiCard("Paid", "${fmt.format(o.totalPaid)} UZS", Modifier.weight(1f))
                                    Spacer(Modifier.weight(1f))
                                }
                            }
                        }
                    }
                    1 -> {
                        if (invoices.isEmpty()) {
                            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                                Text("No invoices", color = MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                        } else {
                            LazyColumn(contentPadding = PaddingValues(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md)) {
                                items(invoices, key = { it.invoiceId }) { inv ->
                                    val displayAmount = if (inv.amount > 0) inv.amount else inv.amountUzs
                                    val displayCurrency = if (inv.currency.isBlank()) "UZS" else inv.currency.uppercase()
                                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                                        Row(modifier = Modifier.padding(PegasusSpacing.lg), verticalAlignment = Alignment.CenterVertically) {
                                            Column(modifier = Modifier.weight(1f)) {
                                                Text(inv.retailerName, style = MaterialTheme.typography.titleSmall)
                                                Text("${fmt.format(displayAmount)} $displayCurrency · ${inv.dueDate}", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                                                val payoutOwner = buildString {
                                                    append(if (inv.payoutOwnerType.isBlank()) "SUPPLIER" else inv.payoutOwnerType)
                                                    if (inv.payoutOwnerId.isNotBlank()) {
                                                        append(":")
                                                        append(inv.payoutOwnerId.take(8))
                                                    }
                                                }
                                                Text(
                                                    "Owner $payoutOwner · Fee ${fmt.format(inv.feeAmount)} · Net ${fmt.format(inv.netPayoutAmount)}",
                                                    style = MaterialTheme.typography.bodySmall,
                                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                                )
                                            }
                                            AssistChip(onClick = {}, label = { Text(inv.status, style = MaterialTheme.typography.labelSmall) })
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
}

@Composable
private fun KpiCard(label: String, value: String, modifier: Modifier = Modifier) {
    ElevatedCard(modifier = modifier) {
        Column(modifier = Modifier.padding(PegasusSpacing.md)) {
            Text(value, style = MaterialTheme.typography.titleMedium)
            Spacer(Modifier.height(2.dp))
            Text(label, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}
