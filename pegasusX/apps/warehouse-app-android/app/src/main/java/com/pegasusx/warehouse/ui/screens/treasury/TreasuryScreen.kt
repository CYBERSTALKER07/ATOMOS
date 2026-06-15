package com.pegasusx.warehouse.ui.screens.treasury

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.warehouse.data.model.Invoice
import com.pegasusx.warehouse.data.model.TreasuryOverview
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.ui.components.WarehouseLoadingState
import com.pegasusx.warehouse.ui.components.WarehouseMetricTile
import com.pegasusx.warehouse.ui.components.WarehouseOpsListCard
import com.pegasusx.warehouse.ui.components.WarehouseSectionTitle
import com.pegasusx.warehouse.ui.components.WarehouseStateKind
import com.pegasusx.warehouse.ui.components.WarehouseStatePane
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import java.text.NumberFormat
import java.util.Locale

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TreasuryScreen(
    api: WarehouseApi,
    onBack: (() -> Unit)? = null,
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
                navigationIcon = { if (onBack != null) { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") } } },
                actions = { IconButton(onClick = { load() }) { Icon(Icons.Default.Refresh, "Refresh") } },
            )
        },
    ) { innerPadding ->
        when {
            loading -> WarehouseLoadingState(
                title = "Loading treasury…",
                body = "Financial overview and invoices",
                modifier = Modifier.padding(innerPadding),
            )
            error != null -> WarehouseStatePane(
                kind = WarehouseStateKind.Error,
                headline = "Treasury unavailable",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier.padding(innerPadding),
            )
            else -> Column(modifier = Modifier.fillMaxSize().padding(innerPadding)) {
                TabRow(selectedTabIndex = tab) {
                    Tab(selected = tab == 0, onClick = { tab = 0 }, text = { Text("Overview") })
                    Tab(selected = tab == 1, onClick = { tab = 1 }, text = { Text("Invoices (${invoices.size})") })
                }
                when (tab) {
                    0 -> overview?.let { o ->
                        LazyColumn(
                            contentPadding = PaddingValues(PegasusSpacing.lg),
                            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                        ) {
                            item { WarehouseSectionTitle("Financial snapshot") }
                            item {
                                Row(
                                    horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                                    modifier = Modifier.fillMaxWidth(),
                                ) {
                                    WarehouseMetricTile(
                                        label = "Outstanding",
                                        value = "${fmt.format(o.totalOutstanding)} UZS",
                                        modifier = Modifier.weight(1f),
                                    )
                                    WarehouseMetricTile(
                                        label = "Invoiced",
                                        value = "${fmt.format(o.totalInvoiced)} UZS",
                                        modifier = Modifier.weight(1f),
                                    )
                                }
                            }
                            item {
                                WarehouseMetricTile(
                                    label = "Paid",
                                    value = "${fmt.format(o.totalPaid)} UZS",
                                    modifier = Modifier.fillMaxWidth(0.5f),
                                )
                            }
                        }
                    }
                    1 -> {
                        if (invoices.isEmpty()) {
                            WarehouseStatePane(
                                kind = WarehouseStateKind.Empty,
                                headline = "No invoices",
                                body = "Retailer invoices will appear here when issued.",
                            )
                        } else {
                            LazyColumn(
                                contentPadding = PaddingValues(PegasusSpacing.lg),
                                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                            ) {
                                items(invoices, key = { it.invoiceId }) { inv ->
                                    val displayAmount = if (inv.amount > 0) inv.amount else inv.amountUzs
                                    val displayCurrency = if (inv.currency.isBlank()) "UZS" else inv.currency.uppercase()
                                    val payoutOwner = buildString {
                                        append(if (inv.payoutOwnerType.isBlank()) "SUPPLIER" else inv.payoutOwnerType)
                                        if (inv.payoutOwnerId.isNotBlank()) {
                                            append(":")
                                            append(inv.payoutOwnerId.take(8))
                                        }
                                    }
                                    WarehouseOpsListCard(
                                        headline = inv.retailerName,
                                        supporting = "${fmt.format(displayAmount)} $displayCurrency · due ${inv.dueDate} · Owner $payoutOwner · Net ${fmt.format(inv.netPayoutAmount)}",
                                        status = inv.status,
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
