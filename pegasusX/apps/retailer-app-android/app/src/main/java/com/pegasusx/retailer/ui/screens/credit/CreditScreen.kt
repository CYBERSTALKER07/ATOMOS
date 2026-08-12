package com.pegasusx.retailer.ui.screens.credit

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
import androidx.compose.material3.Card
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
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.ViewModel
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.json.asBoolean
import com.pegasusx.retailer.data.json.asInt
import com.pegasusx.retailer.data.json.asJsonObject
import com.pegasusx.retailer.data.json.asLong
import com.pegasusx.retailer.data.json.asString
import com.pegasusx.retailer.data.json.getAsJsonArray
import dagger.hilt.android.lifecycle.HiltViewModel
import javax.inject.Inject
import kotlinx.coroutines.launch
import kotlinx.serialization.json.jsonObject

data class CreditRelRow(
    val supplierId: String,
    val limitMinor: Long,
    val balanceMinor: Long,
    val termsDays: Int,
    val onHold: Boolean,
)

data class ArInvoiceRow(
    val invoiceId: String,
    val supplierId: String,
    val balanceMinor: Long,
    val status: String,
    val dueAt: String,
)

@HiltViewModel
class CreditViewModel @Inject constructor(val api: PegasusApi) : ViewModel()

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CreditScreen(
    onNavigateBack: () -> Unit,
    viewModel: CreditViewModel = hiltViewModel(),
) {
    val scope = rememberCoroutineScope()
    var rels by remember { mutableStateOf<List<CreditRelRow>>(emptyList()) }
    var invoices by remember { mutableStateOf<List<ArInvoiceRow>>(emptyList()) }
    var banner by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(Unit) {
        scope.launch {
            try {
                val relJson = viewModel.api.getCreditRelationships().asJsonObject
                val relArr = relJson.getAsJsonArray("relationships")
                rels = buildList {
                    if (relArr != null) {
                        for (el in relArr) {
                            val o = el.jsonObject
                            add(
                                CreditRelRow(
                                    supplierId = o["supplier_id"]?.asString ?: "—",
                                    limitMinor = o["credit_limit_minor"]?.asLong ?: 0L,
                                    balanceMinor = o["current_balance_minor"]?.asLong ?: 0L,
                                    termsDays = o["terms_days"]?.asInt ?: 0,
                                    onHold = o["on_hold"]?.asBoolean ?: false,
                                ),
                            )
                        }
                    }
                }
                val invJson = viewModel.api.getArInvoices().asJsonObject
                val invArr = invJson.getAsJsonArray("invoices")
                invoices = buildList {
                    if (invArr != null) {
                        for (el in invArr) {
                            val o = el.jsonObject
                            add(
                                ArInvoiceRow(
                                    invoiceId = o["invoice_id"]?.asString ?: "—",
                                    supplierId = o["supplier_id"]?.asString ?: "—",
                                    balanceMinor = o["balance_minor"]?.asLong ?: 0L,
                                    status = o["status"]?.asString ?: "—",
                                    dueAt = o["due_at"]?.asString ?: "—",
                                ),
                            )
                        }
                    }
                }
                banner = null
            } catch (e: Exception) {
                banner = e.message
            }
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Credit & AR") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding),
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            banner?.let {
                item {
                    Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
                }
            }
            item {
                Text("Trade credit partners", style = MaterialTheme.typography.titleMedium)
            }
            if (rels.isEmpty()) {
                item { Text("No credit relationships.", style = MaterialTheme.typography.bodySmall) }
            } else {
                items(rels, key = { it.supplierId }) { r ->
                    Card(modifier = Modifier.fillMaxWidth()) {
                        Column(Modifier = Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            Text(r.supplierId, style = MaterialTheme.typography.labelMedium)
                            Text("Limit ${r.limitMinor / 100.0} · Balance ${r.balanceMinor / 100.0}")
                            Text("Terms ${r.termsDays}d${if (r.onHold) " · ON HOLD" else ""}")
                        }
                    }
                }
            }
            item {
                Text("Open AR invoices", style = MaterialTheme.typography.titleMedium)
            }
            if (invoices.isEmpty()) {
                item { Text("No open invoices.", style = MaterialTheme.typography.bodySmall) }
            } else {
                items(invoices, key = { it.invoiceId }) { inv ->
                    Card(modifier = Modifier.fillMaxWidth()) {
                        Column(Modifier = Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            Text(inv.invoiceId, style = MaterialTheme.typography.labelMedium)
                            Text("${inv.supplierId} · ${inv.status}")
                            Text("Balance ${inv.balanceMinor / 100.0} · Due ${inv.dueAt}")
                        }
                    }
                }
            }
        }
    }
}
