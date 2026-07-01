package com.pegasusx.supplier.ui.screens.planning

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Policy
import androidx.compose.material.icons.filled.Speed
import androidx.compose.material.icons.filled.ToggleOn
import androidx.compose.material.icons.filled.TrendingUp
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SupplierReplenishmentPolicy
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierKpiTile
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ReplenishmentPoliciesScreen(
    ops: SupplierOperationsRepository,
    onBack: () -> Unit,
) {
    var policy by remember { mutableStateOf<SupplierReplenishmentPolicy?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            val resp = ops.getReplenishmentPolicies()
            if (resp.isSuccessful) {
                policy = resp.body()
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
                title = { Text("Replenishment policies") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> SupplierLoadingState("Loading policies…", modifier = Modifier.padding(padding))
            error != null -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Policies unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            policy == null -> SupplierStatePane(
                kind = SupplierStateKind.Empty,
                headline = "No policy",
                body = "Replenishment policy has not been configured.",
                modifier = Modifier.padding(padding),
            )
            else -> Column(
                modifier = Modifier
                    .padding(padding)
                    .padding(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm), modifier = Modifier.fillMaxWidth()) {
                    SupplierKpiTile(
                        "Auto stable",
                        if (policy!!.autoApproveStable) "On" else "Off",
                        Icons.Default.ToggleOn,
                        Modifier.weight(1f),
                    )
                    SupplierKpiTile(
                        "Predictive push",
                        if (policy!!.autoApprovePredictivePush) "On" else "Off",
                        Icons.Default.TrendingUp,
                        Modifier.weight(1f),
                    )
                }
                Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm), modifier = Modifier.fillMaxWidth()) {
                    SupplierKpiTile(
                        "Max daily units",
                        policy!!.maxDailyTransferUnits.toString(),
                        Icons.Default.Speed,
                        Modifier.weight(1f),
                    )
                    SupplierKpiTile(
                        "Min confidence",
                        policy!!.minConfidenceScore.toString(),
                        Icons.Default.Policy,
                        Modifier.weight(1f),
                    )
                }
            }
        }
    }
}
