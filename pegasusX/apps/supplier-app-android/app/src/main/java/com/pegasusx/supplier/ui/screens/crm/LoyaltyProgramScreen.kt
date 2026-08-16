package com.pegasusx.supplier.ui.screens.crm

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
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
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.data.model.LoyaltyProgram
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.util.SupplierIdempotencyKeys
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LoyaltyProgramScreen(
    ops: SupplierOperationsRepository,
    onBack: () -> Unit,
) {
    var program by remember { mutableStateOf<LoyaltyProgram?>(null) }
    var earnBps by remember { mutableStateOf("100") }
    var reason by remember { mutableStateOf("") }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var status by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    fun load() {
        loading = true
        error = null
        scope.launch {
            val resp = ops.getLoyaltyProgram()
            if (resp.isSuccessful) {
                program = resp.body()
                earnBps = (resp.body()?.earnBps ?: 100L).toString()
            } else {
                error = "Failed (${resp.code()})"
            }
            loading = false
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Loyalty program") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading && program == null -> PegasusLoadingState(
                title = "Loading loyalty",
                body = "Earn bps and tiers",
                modifier = Modifier.fillMaxSize().padding(padding),
            )
            error != null && program == null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Loyalty unavailable",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier.fillMaxSize().padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier.fillMaxSize().padding(padding),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                item {
                    Text(
                        "Earn on paid orders. Burn is out of scope. Unconfigured retailers see enrolled=false, not a fake Bronze.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                item {
                    Text(
                        "Source: ${program?.source ?: "unconfigured"}",
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
                item {
                    OutlinedTextField(
                        value = earnBps,
                        onValueChange = { earnBps = it },
                        label = { Text("Earn bps") },
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
                item {
                    OutlinedTextField(
                        value = reason,
                        onValueChange = { reason = it },
                        label = { Text("Reason (required)") },
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
                item {
                    Button(
                        onClick = {
                            val why = reason.trim()
                            if (why.isEmpty()) {
                                status = "Typed reason required"
                                return@Button
                            }
                            busy = true
                            scope.launch {
                                val bps = earnBps.toLongOrNull()?.takeIf { it > 0 } ?: 100L
                                val resp = ops.patchLoyaltyProgram(
                                    LoyaltyProgram(
                                        supplierId = SupplierIdempotencyKeys.supplierScopeId(),
                                        earnBps = bps,
                                        tiers = program?.tiers.orEmpty(),
                                        reason = why,
                                    ),
                                    SupplierIdempotencyKeys.loyaltyProgramPatch(
                                        SupplierIdempotencyKeys.supplierScopeId(),
                                        why,
                                    ),
                                )
                                status = if (resp.isSuccessful) {
                                    program = resp.body()
                                    reason = ""
                                    "Saved (${resp.body()?.source ?: "program"})"
                                } else {
                                    "Save failed (${resp.code()})"
                                }
                                busy = false
                            }
                        },
                        enabled = !busy,
                    ) { Text(if (busy) "Saving…" else "Save program") }
                }
                status?.let { item { Text(it, style = MaterialTheme.typography.bodySmall) } }
                item {
                    Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                        program?.tiers.orEmpty().forEach { tier ->
                            Text(
                                "${tier.name} from ${tier.minPoints} lifetime points",
                                style = MaterialTheme.typography.bodySmall,
                            )
                        }
                    }
                }
            }
        }
    }
}
