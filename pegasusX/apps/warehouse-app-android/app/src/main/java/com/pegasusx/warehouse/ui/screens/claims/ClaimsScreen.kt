package com.pegasusx.warehouse.ui.screens.claims

import androidx.compose.ui.res.stringResource

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
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
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
import com.pegasusx.warehouse.data.model.WarehouseClaim
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import com.pegasusx.warehouse.R

private val STATUS_FILTERS = listOf("OPEN", "UNDER_REVIEW", "RESOLVED", "REJECTED", "")

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ClaimsScreen(
    api: WarehouseApi,
    onOrderClick: (String) -> Unit,
    onOpenReturns: () -> Unit,
    onOpenExceptions: () -> Unit,
    onBack: (() -> Unit)? = null,
) {
    var status by remember { mutableStateOf("OPEN") }
    var claims by remember { mutableStateOf<List<WarehouseClaim>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        loading = true
        error = null
        scope.launch {
            try {
                val resp = api.getSupplierClaims(
                    status = status.takeIf { it.isNotBlank() },
                    limit = 50,
                )
                if (resp.isSuccessful && resp.body() != null) {
                    claims = resp.body()!!.claims
                } else {
                    error = "Failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(status) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Claims (reverse logistics)") },
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
        Column(Modifier.fillMaxSize().padding(innerPadding)) {
            Row(
                Modifier
                    .fillMaxWidth()
                    .padding(horizontal = PegasusSpacing.lg, vertical = PegasusSpacing.sm),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                STATUS_FILTERS.forEach { s ->
                    FilterChip(
                        selected = status == s,
                        onClick = { status = s },
                        label = { Text(if (s.isBlank()) "ALL" else s) },
                    )
                }
            }
            Row(
                Modifier.padding(horizontal = PegasusSpacing.lg),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                TextButton(onClick = onOpenReturns) { Text("Returns inbound") }
                TextButton(onClick = onOpenExceptions) { Text("Exception triage") }
            }
            Text(
                "Read-only prep queue. Approve/reject stays supplier HQ.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(horizontal = PegasusSpacing.lg, vertical = PegasusSpacing.sm),
            )
            when {
                loading -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator()
                }
                error != null -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Text(error!!, color = MaterialTheme.colorScheme.error)
                        Spacer(Modifier.height(PegasusSpacing.lg))
                        Button(onClick = { load() }) { Text("Retry") }
                    }
                }
                claims.isEmpty() -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    Text("No claims in this filter.", color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
                else -> LazyColumn(
                    contentPadding = PaddingValues(PegasusSpacing.lg),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                ) {
                    items(claims, key = { it.claimId }) { c ->
                        ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                            Column(Modifier.padding(PegasusSpacing.lg)) {
                                Text(stringResource(R.string.mobile_warehouse_ui_claimtype_status, c.claimType, c.status),
                                    style = MaterialTheme.typography.labelSmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                                Text(c.claimId, style = MaterialTheme.typography.titleSmall)
                                if (c.orderId.isNotBlank()) {
                                    TextButton(onClick = { onOrderClick(c.orderId) }) {
                                        Text(stringResource(R.string.mobile_warehouse_ui_order_orderid, c.orderId))
                                    }
                                }
                                Text(
                                    stringResource(R.string.mobile_warehouse_ui_retailer_retailerid_amountminor_currency, c.retailerId, c.amountMinor, c.currency),
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                                c.lineItems.forEach { li ->
                                    Text(stringResource(R.string.mobile_warehouse_ui_sku_quantity, li.sku, li.quantity) +
                                            if (li.reason.isNotBlank()) " (${li.reason})" else "",
                                        style = MaterialTheme.typography.bodySmall,
                                    )
                                }
                                if (c.description.isNotBlank()) {
                                    Text(c.description, style = MaterialTheme.typography.bodySmall)
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
