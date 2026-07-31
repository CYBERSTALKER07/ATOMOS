package com.pegasusx.supplier.ui.screens.treasury

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.pegasus.design.showFullScreenLoading
import com.pegasusx.supplier.ui.components.SupplierKpiTile
import com.pegasusx.supplier.ui.components.SupplierLeadingIcon
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.ui.components.formatMinorAmount
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.ui.viewmodel.TreasuryViewModel

private data class TreasuryLink(
    val title: String,
    val subtitle: String,
    val icon: ImageVector,
    val onClick: () -> Unit,
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TreasuryHubScreen(
    onBack: () -> Unit,
    onLedger: () -> Unit,
    onPayments: () -> Unit,
    onReconciliation: () -> Unit,
    onEarnings: () -> Unit,
    onChargebacks: () -> Unit,
    onClaimChargebacks: () -> Unit = {},
    onClaims: () -> Unit = {},
    onCompliance: () -> Unit = {},
    onCashReconciliations: () -> Unit = {},
    onCreditNotes: () -> Unit = {},
    onCreditProfiles: () -> Unit = {},
    viewModel: TreasuryViewModel = hiltViewModel(),
) {
    val state by viewModel.hubState.collectAsStateWithLifecycle()

    LaunchedEffect(Unit) { viewModel.loadHub() }

    val links = listOf(
        TreasuryLink("Payment ledger", "Treasury entries", Icons.Default.AccountBalance, onLedger),
        TreasuryLink("Payments", "Settlement authority", Icons.Default.CreditCard, onPayments),
        TreasuryLink("Reconciliation", "Settlement mismatches", Icons.Default.Balance, onReconciliation),
        TreasuryLink("Compliance audit", "Open fiscal, force-completes, freezes", Icons.Default.Gavel, onCompliance),
        TreasuryLink("Cash reconciliations", "Driver shift cash discrepancies", Icons.Default.Money, onCashReconciliations),
        TreasuryLink("Credit notes", "Draft and issue credit notes", Icons.Default.Description, onCreditNotes),
        TreasuryLink("Credit profiles", "Limits, balances, freeze / unfreeze", Icons.Default.AccountBox, onCreditProfiles),
        TreasuryLink("Earnings", "Revenue summary", Icons.Default.Payments, onEarnings),
        TreasuryLink("Chargebacks", "Record chargeback or reversal", Icons.Default.Undo, onChargebacks),
        TreasuryLink("Claim chargebacks", "Logistics claim settlements", Icons.Default.Description, onClaimChargebacks),
        TreasuryLink("Claims queue", "Approve / reject OS&D claims", Icons.Default.Warning, onClaims),
    )

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Treasury hub") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        when {
            showFullScreenLoading(state.loading, state.earnings != null) -> PegasusLoadingState(
                title = "Loading treasury…",
                body = "KPIs and links",
                modifier = Modifier.padding(padding),
            )
            state.error != null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Treasury unavailable",
                body = state.error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { viewModel.loadHub() },
            )
            else -> LazyColumn(
                modifier = Modifier.padding(padding),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                item {
                    val earnings = state.earnings
                    Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md)) {
                        SupplierKpiTile(
                            label = "Month revenue",
                            value = formatMinorAmount(earnings?.monthMinor ?: 0, earnings?.currency.orEmpty()),
                            icon = Icons.Default.TrendingUp,
                            modifier = Modifier.weight(1f),
                        )
                        SupplierKpiTile(
                            label = "Ledger entries",
                            value = state.ledgerEntryCount.toString(),
                            icon = Icons.Default.Receipt,
                            modifier = Modifier.weight(1f),
                        )
                    }
                }
                item {
                    SupplierKpiTile(
                        label = "Reconciliation mismatches",
                        value = state.mismatchCount.toString(),
                        icon = Icons.Default.Warning,
                    )
                }
                item {
                    Text(
                        "Treasury modules",
                        style = MaterialTheme.typography.titleSmall,
                        color = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.padding(vertical = PegasusSpacing.sm),
                    )
                }
                items(links.size) { index ->
                    val link = links[index]
                    ListItem(
                        headlineContent = { Text(link.title) },
                        supportingContent = { Text(link.subtitle) },
                        leadingContent = { SupplierLeadingIcon(icon = link.icon) },
                        modifier = Modifier.clickable(onClick = link.onClick),
                    )
                    HorizontalDivider()
                }
            }
        }
    }
}
