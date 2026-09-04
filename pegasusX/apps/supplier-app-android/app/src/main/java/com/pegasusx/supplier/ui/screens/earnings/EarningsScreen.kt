package com.pegasusx.supplier.ui.screens.earnings

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SupplierEarnings
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasus.design.ui.PegasusLoadingState
import com.pegasus.design.ui.PegasusStateKind
import com.pegasus.design.ui.PegasusStatePane
import com.pegasus.design.network.moneyCurrency
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import com.pegasusx.supplier.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun EarningsScreen(api: SupplierApi, ops: SupplierOperationsRepository) {
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var earnings by remember { mutableStateOf<SupplierEarnings?>(null) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(Unit) {
        scope.launch {
            try {
                val resp = api.getEarnings()
                if (resp.isSuccessful && resp.body() != null) {
                    earnings = resp.body()
                } else {
                    val ledger = ops.getPaymentLedger()
                    if (ledger.isSuccessful && ledger.body() != null) {
                        val items = ledger.body()!!.items
                        val total = items.sumOf { it.amountMinor }
                        val currency = moneyCurrency(items.firstOrNull()?.currency)
                        earnings = SupplierEarnings(
                            currency = currency,
                            todayMinor = 0,
                            weekMinor = total,
                            monthMinor = total,
                            authoritative = false,
                        )
                    } else {
                        error = "Failed (${resp.code()})"
                    }
                }
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    Scaffold(topBar = { TopAppBar(title = { Text("Earnings") }) }) { padding ->
        Box(Modifier.padding(padding)) {
            when {
                loading -> PegasusLoadingState("Loading earnings…", "Treasury summary")
                error != null -> PegasusStatePane(
                    kind = PegasusStateKind.Error,
                    headline = "Earnings unavailable",
                    body = error!!,
                )
                earnings == null -> PegasusStatePane(
                    kind = PegasusStateKind.Empty,
                    headline = "No data",
                    body = "Earnings not available.",
                )
                else -> {
                    val e = earnings!!
                    Column(
                        Modifier.padding(PegasusSpacing.lg),
                        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                    ) {
                        Text(stringResource(R.string.mobile_supplier_ui_today_currency_todayminor, e.currency, e.todayMinor), style = MaterialTheme.typography.titleLarge)
                        Text(stringResource(R.string.mobile_supplier_ui_week_currency_weekminor, e.currency, e.weekMinor))
                        Text(stringResource(R.string.mobile_supplier_ui_month_currency_monthminor, e.currency, e.monthMinor))
                        if (!e.authoritative) {
                            Text("Indicative only", style = MaterialTheme.typography.labelSmall)
                        }
                    }
                }
            }
        }
    }
}
