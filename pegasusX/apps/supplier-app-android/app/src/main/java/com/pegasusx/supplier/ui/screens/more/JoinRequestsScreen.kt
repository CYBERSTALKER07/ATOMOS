package com.pegasusx.supplier.ui.screens.more

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.pegasusx.supplier.R
import com.pegasusx.supplier.data.model.JoinRequestItem
import com.pegasusx.supplier.ui.components.SupplierStatusChip
import com.pegasusx.supplier.ui.design.PegasusRuntimeBanner
import com.pegasusx.supplier.ui.design.PegasusRuntimeTone
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.ui.viewmodel.JoinRequestsViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun JoinRequestsScreen(
    onBack: () -> Unit,
    viewModel: JoinRequestsViewModel = hiltViewModel(),
) {
    val state by viewModel.state.collectAsStateWithLifecycle()

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(stringResource(R.string.supplier_portal_join_requests_inbox_title)) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Filled.ArrowBack,
                            contentDescription = stringResource(R.string.common_action_back),
                        )
                    }
                },
                actions = {
                    IconButton(onClick = { viewModel.load(silent = false) }) {
                        Icon(
                            imageVector = Icons.Default.Refresh,
                            contentDescription = stringResource(R.string.common_action_retry),
                        )
                    }
                },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(horizontal = PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Text(
                text = stringResource(R.string.supplier_portal_join_requests_inbox_subtitle),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )

            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                listOf(
                    "PENDING" to R.string.supplier_portal_join_requests_status_pending,
                    "APPROVED" to R.string.supplier_portal_join_requests_status_approved,
                    "REJECTED" to R.string.supplier_portal_join_requests_status_rejected,
                ).forEach { (status, labelRes) ->
                    FilterChip(
                        selected = state.statusFilter == status,
                        onClick = { viewModel.setStatusFilter(status) },
                        label = { Text(stringResource(labelRes)) },
                    )
                }
            }

            state.error?.let { err ->
                PegasusRuntimeBanner(
                    tone = PegasusRuntimeTone.Warning,
                    message = err,
                    onRetry = { viewModel.load(silent = false) },
                )
            }
            state.notice?.let { msg ->
                PegasusRuntimeBanner(
                    tone = PegasusRuntimeTone.Live,
                    message = msg,
                )
            }

            when {
                state.loading && state.requests.isEmpty() -> {
                    Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                        CircularProgressIndicator()
                    }
                }
                state.requests.isEmpty() -> {
                    Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                        Text(
                            text = stringResource(R.string.supplier_portal_join_requests_empty_inbox),
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                }
                else -> {
                    LazyColumn(
                        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                        contentPadding = PaddingValues(bottom = PegasusSpacing.lg),
                    ) {
                        items(state.requests, key = { it.requestId }) { req ->
                            JoinRequestCard(
                                item = req,
                                reason = state.rejectionReasonById[req.requestId].orEmpty(),
                                busy = state.busyId == req.requestId,
                                onReasonChange = { viewModel.setRejectionReason(req.requestId, it) },
                                onApprove = { viewModel.decide(req.requestId, approve = true) },
                                onReject = { viewModel.decide(req.requestId, approve = false) },
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun JoinRequestCard(
    item: JoinRequestItem,
    reason: String,
    busy: Boolean,
    onReasonChange: (String) -> Unit,
    onApprove: () -> Unit,
    onReject: () -> Unit,
) {
    OutlinedCard(
        shape = RoundedCornerShape(14.dp),
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.5f)),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = item.requesterName?.takeIf { it.isNotBlank() } ?: item.requesterPhone,
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = FontWeight.Bold,
                    )
                    Text(
                        text = item.requesterPhone,
                        style = MaterialTheme.typography.bodySmall,
                        fontFamily = FontFamily.Monospace,
                    )
                }
                SupplierStatusChip(status = item.status)
            }
            if (!item.requesterEmail.isNullOrBlank()) {
                Text(
                    text = item.requesterEmail,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            Text(
                text = "${stringResource(R.string.supplier_portal_join_requests_role_sought)}: ${item.roleSought} · INN ${item.normalizedInn}",
                style = MaterialTheme.typography.labelMedium,
            )
            if (item.createdAt.isNotBlank()) {
                Text(
                    text = "${stringResource(R.string.supplier_portal_join_requests_requested_at)}: ${item.createdAt}",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            if (!item.decisionReason.isNullOrBlank()) {
                Text(
                    text = item.decisionReason,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            if (item.status.equals("PENDING", ignoreCase = true)) {
                OutlinedTextField(
                    value = reason,
                    onValueChange = onReasonChange,
                    placeholder = { Text(stringResource(R.string.supplier_portal_join_requests_reject_reason_placeholder)) },
                    modifier = Modifier.fillMaxWidth(),
                )
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    OutlinedButton(
                        onClick = onReject,
                        enabled = !busy,
                        colors = ButtonDefaults.outlinedButtonColors(contentColor = MaterialTheme.colorScheme.error),
                        modifier = Modifier.weight(1f),
                    ) {
                        Text(stringResource(R.string.supplier_portal_join_requests_reject_button))
                    }
                    Button(
                        onClick = onApprove,
                        enabled = !busy,
                        modifier = Modifier.weight(1f),
                    ) {
                        Text(stringResource(R.string.supplier_portal_join_requests_approve_button))
                    }
                }
            }
        }
    }
}
