package com.pegasusx.retailer.ui.screens.settings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.outlined.Refresh
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.pegasusx.retailer.R
import com.pegasusx.retailer.util.OrderLineContractHelper
import com.pegasusx.retailer.util.RetailerW2ContractHelper

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OwnerApprovalsScreen(
    onNavigateBack: () -> Unit,
    viewModel: OwnerGovernanceViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    val reasonDrafts = remember { mutableStateMapOf<String, String>() }

    LaunchedEffect(Unit) {
        viewModel.loadRoleAndApprovals()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(stringResource(R.string.retailer_approval_inbox_title)) },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(
                            Icons.AutoMirrored.Filled.ArrowBack,
                            contentDescription = stringResource(R.string.common_action_back),
                        )
                    }
                },
                actions = {
                    IconButton(onClick = viewModel::refreshApprovals) {
                        Icon(Icons.Outlined.Refresh, contentDescription = "Refresh")
                    }
                },
            )
        },
    ) { padding ->
        if (uiState.isLoadingRole) {
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(padding),
                contentAlignment = Alignment.Center,
            ) {
                CircularProgressIndicator()
            }
            return@Scaffold
        }

        if (!uiState.isOwner) {
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(padding)
                    .padding(20.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Card(modifier = Modifier.fillMaxWidth()) {
                    Column(
                        modifier = Modifier.padding(16.dp),
                        verticalArrangement = Arrangement.spacedBy(6.dp),
                    ) {
                        Text(
                            text = "Owner access required",
                            style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold),
                        )
                        Text(
                            text = "Only the store owner can approve or reject mandate-exceeding orders.",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                }
            }
            return@Scaffold
        }

        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(horizontal = 16.dp),
            contentPadding = PaddingValues(vertical = 12.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item {
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    listOf(
                        "PENDING_OWNER" to "Pending",
                        "APPROVED" to "Approved",
                        "REJECTED" to "Rejected",
                    ).forEach { (status, label) ->
                        FilterChip(
                            selected = uiState.statusFilter == status,
                            onClick = { viewModel.selectStatusFilter(status) },
                            label = { Text(label) },
                        )
                    }
                }
            }

            uiState.actionBanner?.let { banner ->
                item {
                    Text(
                        text = banner,
                        style = MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.Medium),
                        color = MaterialTheme.colorScheme.primary,
                    )
                }
            }

            uiState.approvalsError?.let { err ->
                item {
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text(err, color = MaterialTheme.colorScheme.error)
                        OutlinedButton(onClick = viewModel::refreshApprovals) { Text("Retry") }
                    }
                }
            }

            if (uiState.approvalsLoading) {
                item {
                    Box(modifier = Modifier.fillMaxWidth().padding(vertical = 24.dp), contentAlignment = Alignment.Center) {
                        CircularProgressIndicator()
                    }
                }
            } else if (uiState.approvals.isEmpty()) {
                item {
                    Text(
                        text = "No approval requests in this state.",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            } else {
                items(uiState.approvals, key = { it.requestId }) { req ->
                    val isBusy = uiState.decidingRequestId == req.requestId
                    val isPending = req.status.equals("PENDING_OWNER", ignoreCase = true)
                    Card(modifier = Modifier.fillMaxWidth()) {
                        Column(
                            modifier = Modifier.padding(14.dp),
                            verticalArrangement = Arrangement.spacedBy(8.dp),
                        ) {
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                Text(
                                    text = req.requestedByName.ifBlank { req.requestedByUser.ifBlank { "Seller" } },
                                    style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.Bold),
                                )
                                Text(
                                    text = OrderLineContractHelper.formatWholeSom(req.amountMinor),
                                    style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.Bold),
                                    color = MaterialTheme.colorScheme.primary,
                                )
                            }

                            Text(
                                text = RetailerW2ContractHelper.formatTrippedRuleLabel(req.trippedRule),
                                style = MaterialTheme.typography.bodySmall,
                            )

                            val metaParts = buildList {
                                if (req.paymentMethod.isNotBlank()) add("Method: ${req.paymentMethod}")
                                if (req.payloadKind.isNotBlank()) add(req.payloadKind)
                                add(req.status)
                            }
                            Text(
                                text = metaParts.joinToString(" · "),
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )

                            if (!req.decisionReason.isNullOrBlank()) {
                                Text(
                                    text = "Note: ${req.decisionReason}",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            }

                            if (isPending) {
                                OutlinedTextField(
                                    value = reasonDrafts[req.requestId] ?: "",
                                    onValueChange = { reasonDrafts[req.requestId] = it },
                                    label = { Text("Decision note (optional)") },
                                    singleLine = true,
                                    enabled = !isBusy,
                                    modifier = Modifier.fillMaxWidth(),
                                )
                                Row(
                                    modifier = Modifier.fillMaxWidth(),
                                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                                ) {
                                    Button(
                                        onClick = { viewModel.approveRequest(req.requestId, reasonDrafts[req.requestId]) },
                                        enabled = !isBusy,
                                        modifier = Modifier.weight(1f),
                                    ) {
                                        Text(if (isBusy) "…" else stringResource(R.string.retailer_approval_approve))
                                    }
                                    Spacer(modifier = Modifier.width(4.dp))
                                    OutlinedButton(
                                        onClick = { viewModel.rejectRequest(req.requestId, reasonDrafts[req.requestId]) },
                                        enabled = !isBusy,
                                        modifier = Modifier.weight(1f),
                                    ) {
                                        Text(stringResource(R.string.retailer_approval_reject))
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
