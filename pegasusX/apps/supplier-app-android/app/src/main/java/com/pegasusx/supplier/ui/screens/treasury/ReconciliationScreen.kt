package com.pegasusx.supplier.ui.screens.treasury

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import java.text.NumberFormat
import java.util.Locale
import com.pegasusx.supplier.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ReconciliationScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var netMinor by remember { mutableLongStateOf(0L) }
    var currency by remember { mutableStateOf("UZS") }
    var mismatchCount by remember { mutableIntStateOf(0) }
    val scope = rememberCoroutineScope()
    val fmt = remember { NumberFormat.getInstance(Locale("uz", "UZ")) }

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val authorityResp = ops.getPaymentSettlementAuthority()
                val mismatchResp = ops.getPaymentReconciliationMismatches()
                if (authorityResp.isSuccessful) {
                    val primary = authorityResp.body()?.totalsByCurrency?.firstOrNull()
                    if (primary != null) {
                        currency = primary.currency
                        netMinor = primary.amountMinorTotal
                    }
                }
                if (mismatchResp.isSuccessful) {
                    mismatchCount = mismatchResp.body()?.items?.size ?: 0
                }
                if (!authorityResp.isSuccessful && !mismatchResp.isSuccessful) {
                    error = "Failed to load reconciliation"
                }
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Reconciliation") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> PegasusLoadingState("Loading reconciliation…", "Settlement authority")
            error != null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Reconciliation unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            else -> Column(
                modifier = Modifier.padding(padding).padding(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                ReconciliationKpi("Settlement net (authority)", "${fmt.format(netMinor)} $currency")
                ReconciliationKpi("Open mismatches", mismatchCount.toString())
                Text(
                    "Full ledger detail is on Payment ledger.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.outline,
                )
            }
        }
    }
}

@Composable
private fun ReconciliationKpi(label: String, value: String) {
    ElevatedCard(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(PegasusSpacing.lg)) {
            Text(label, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.outline)
            Text(value, style = MaterialTheme.typography.headlineMedium)
        }
    }
}
