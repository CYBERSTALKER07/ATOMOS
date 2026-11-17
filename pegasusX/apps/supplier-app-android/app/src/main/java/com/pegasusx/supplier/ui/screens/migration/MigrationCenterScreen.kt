package com.pegasusx.supplier.ui.screens.migration

import android.content.Intent
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.Undo
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.CloudUpload
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Share
import androidx.compose.material.icons.filled.Tune
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.pegasusx.supplier.R
import com.pegasusx.supplier.data.model.*
import com.pegasusx.supplier.ui.components.SupplierStatusChip
import com.pegasusx.supplier.ui.components.formatMinorAmount
import com.pegasusx.supplier.ui.design.PegasusRuntimeBanner
import com.pegasusx.supplier.ui.design.PegasusRuntimeTone
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.ui.viewmodel.MigrationUiState
import com.pegasusx.supplier.ui.viewmodel.MigrationViewModel
import kotlin.math.roundToInt

@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)
@Composable
fun MigrationCenterScreen(
    onBack: () -> Unit,
    showRedirectBanner: Boolean = false,
    viewModel: MigrationViewModel = hiltViewModel(),
) {
    val context = LocalContext.current
    val state by viewModel.state.collectAsStateWithLifecycle()
    val progressBySession by viewModel.migrationProgressBySession.collectAsStateWithLifecycle()

    val filePicker = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.OpenMultipleDocuments(),
    ) { uris ->
        if (uris.isNotEmpty()) {
            viewModel.uploadPickedUris(context, uris)
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(stringResource(R.string.mobile_supplier_migrations_title)) },
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
                    IconButton(
                        onClick = {
                            viewModel.update { copy(showCreateProjectDialog = true) }
                        },
                    ) {
                        Icon(
                            imageVector = Icons.Default.Add,
                            contentDescription = stringResource(R.string.supplier_portal_migrations_create_project),
                        )
                    }
                },
            )
        },
    ) { padding ->
        if (state.loading && state.projects.isEmpty()) {
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(padding),
                contentAlignment = Alignment.Center,
            ) {
                CircularProgressIndicator()
            }
        } else {
            LazyColumn(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(padding)
                    .padding(horizontal = PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                contentPadding = PaddingValues(vertical = PegasusSpacing.lg),
            ) {
                if (showRedirectBanner) {
                    item {
                        PegasusRuntimeBanner(
                            tone = PegasusRuntimeTone.Refreshing,
                            message = stringResource(R.string.supplier_portal_migrations_catalog_redirect_notice),
                        )
                    }
                }

                state.error?.let { err ->
                    item {
                        PegasusRuntimeBanner(
                            tone = PegasusRuntimeTone.Warning,
                            message = err,
                            onRetry = {
                                viewModel.clearBanner()
                                viewModel.load(silent = false)
                            },
                        )
                    }
                }

                state.notice?.let { msg ->
                    item {
                        PegasusRuntimeBanner(
                            tone = PegasusRuntimeTone.Live,
                            message = msg,
                        )
                    }
                }

                // Projects Switcher
                item {
                    ProjectsHeaderCard(
                        state = state,
                        onSelectProject = viewModel::selectProject,
                        onOpenCreate = { viewModel.update { copy(showCreateProjectDialog = true) } },
                    )
                }

                // Upload Dropzone
                if (state.selectedProjectId != null) {
                    item {
                        UploadBundleCard(
                            state = state,
                            onPickFiles = { filePicker.launch(arrayOf("*/*")) },
                        )
                    }
                }

                // Bundles & Auto-Sort Decisions
                val bundles = state.tree?.bundles.orEmpty()
                if (bundles.isNotEmpty()) {
                    item {
                        Text(
                            text = stringResource(R.string.supplier_portal_migrations_sorter_title),
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold,
                        )
                    }
                    items(bundles, key = { it.bundle.bundleId }) { bundleTree ->
                        BundleSummaryCard(
                            bundleTree = bundleTree,
                            onReview = {
                                viewModel.syncSortDrafts(bundleTree.bundle.sortSummary)
                                viewModel.update { copy(activeReviewBundle = bundleTree) }
                            },
                        )
                    }
                }

                // DAG Ordered Sessions
                val sessions = state.sortedSessions
                if (sessions.isNotEmpty()) {
                    item {
                        Text(
                            text = stringResource(R.string.supplier_portal_migrations_mapper_title),
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold,
                        )
                    }
                    items(sessions, key = { it.sessionId }) { session ->
                        val packBlockedBy = state.tree?.dag?.blockedBy?.get(session.pack.orEmpty()).orEmpty()
                        SessionCard(
                            session = session,
                            selected = state.selectedSession?.sessionId == session.sessionId,
                            blockedBy = packBlockedBy,
                            progress = progressBySession[session.sessionId],
                            onSelect = { viewModel.update { copy(selectedSessionId = session.sessionId) } },
                            onOpenMapper = { viewModel.openMapper(session.sessionId) },
                        )
                    }
                }

                // Dry-Run, Apply & 72h Rollback
                val selectedSession = state.selectedSession
                if (state.selectedProjectId != null && selectedSession != null) {
                    item {
                        DryRunAndCutoverSection(
                            state = state,
                            session = selectedSession,
                            progress = progressBySession[selectedSession.sessionId],
                            onApplyModeChange = { mode ->
                                viewModel.update { copy(applyMode = mode) }
                            },
                            onConfirmSetOnHandChange = { confirmed ->
                                viewModel.update { copy(setOnHandConfirmed = confirmed) }
                            },
                            onRunProjectDryRun = { viewModel.runDryRun(sessionOnly = false) },
                            onRunSessionDryRun = { viewModel.runDryRun(sessionOnly = true) },
                            onBuildQuarantineCsv = viewModel::buildQuarantinedCSV,
                            onShareCsv = { csv ->
                                val intent = Intent(Intent.ACTION_SEND).apply {
                                    type = "text/csv"
                                    putExtra(Intent.EXTRA_SUBJECT, "quarantined_${selectedSession.sessionId}.csv")
                                    putExtra(Intent.EXTRA_TEXT, csv)
                                }
                                context.startActivity(Intent.createChooser(intent, "Share Quarantined Errors CSV"))
                            },
                            onApprove = viewModel::approveSession,
                            onApply = viewModel::applySession,
                            onRollback = viewModel::rollbackSession,
                        )
                    }

                    // Held Open Orders Releaser
                    item {
                        OpenOrdersReleaseCard(
                            state = state,
                            onOrderIdsChange = { text ->
                                viewModel.update { copy(openOrderIdsText = text) }
                            },
                            onTargetStatusChange = { status ->
                                viewModel.update { copy(releaseTargetStatus = status) }
                            },
                            onRelease = viewModel::releaseOpenOrders,
                        )
                    }

                    // Project Cutover Readiness & Sign-Off
                    item {
                        CutoverCard(
                            state = state,
                            onCutover = viewModel::cutoverProject,
                        )
                    }
                }
            }
        }
    }

    if (state.showCreateProjectDialog) {
        CreateProjectDialog(
            state = state,
            onNameChange = { name -> viewModel.update { copy(newProjectName = name) } },
            onWarehouseChange = { whId -> viewModel.update { copy(newProjectWarehouseId = whId) } },
            onDismiss = { viewModel.update { copy(showCreateProjectDialog = false) } },
            onConfirm = viewModel::createProject,
        )
    }

    state.activeReviewBundle?.let { reviewBundle ->
        AutoSortReviewDialog(
            state = state,
            bundleTree = reviewBundle,
            onDismiss = { viewModel.update { copy(activeReviewBundle = null) } },
            onModeChange = { mode -> viewModel.update { copy(resolvedInventoryMode = mode) } },
            onPackOverride = { streamId, pack ->
                viewModel.update { copy(packOverrides = packOverrides + (streamId to pack)) }
            },
            onEncodingOverride = { streamId, enc ->
                viewModel.update { copy(encodingOverrides = encodingOverrides + (streamId to enc)) }
            },
            onIgnoreChange = { streamId, ignored ->
                viewModel.update { copy(ignoredStreams = ignoredStreams + (streamId to ignored)) }
            },
            onSave = { viewModel.saveSortOverride(reviewBundle) },
        )
    }

    if (state.mappingSessionId != null) {
        ColumnMapperDialog(
            state = state,
            samplePreview = viewModel::samplePreview,
            onSelectTarget = viewModel::updateProposalTarget,
            onSaveTemplateToggle = { enabled ->
                viewModel.update { copy(saveMappingTemplate = enabled) }
            },
            onAcceptHighConfidence = viewModel::acceptHighConfidenceMappings,
            onConfirm = viewModel::confirmMappings,
            onDismiss = viewModel::closeMapper,
        )
    }
}

@Composable
private fun ProjectsHeaderCard(
    state: MigrationUiState,
    onSelectProject: (String) -> Unit,
    onOpenCreate: () -> Unit,
) {
    OutlinedCard(
        shape = RoundedCornerShape(16.dp),
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.5f)),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = stringResource(R.string.mobile_supplier_migrations_active_projects),
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                )
                OutlinedButton(onClick = onOpenCreate) {
                    Icon(Icons.Default.Add, contentDescription = null, modifier = Modifier.size(16.dp))
                    Spacer(Modifier.width(4.dp))
                    Text(stringResource(R.string.supplier_portal_migrations_create_project))
                }
            }
            Text(
                text = stringResource(R.string.supplier_portal_migrations_subtitle),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )

            if (state.projects.isEmpty()) {
                Text(
                    text = stringResource(R.string.mobile_supplier_migrations_no_projects),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            } else {
                LazyRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    items(state.projects, key = { it.projectId }) { proj ->
                        FilterChip(
                            selected = proj.projectId == state.selectedProjectId,
                            onClick = { onSelectProject(proj.projectId) },
                            label = {
                                Column {
                                    Text(proj.name.ifBlank { proj.projectId.take(8) })
                                    if (proj.status.isNotBlank()) {
                                        Text(
                                            proj.status,
                                            style = MaterialTheme.typography.labelSmall,
                                            fontFamily = FontFamily.Monospace,
                                        )
                                    }
                                }
                            },
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun UploadBundleCard(
    state: MigrationUiState,
    onPickFiles: () -> Unit,
) {
    OutlinedCard(
        shape = RoundedCornerShape(16.dp),
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.5f)),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
        ) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Icon(Icons.Default.CloudUpload, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
                Text(
                    text = stringResource(R.string.supplier_portal_migrations_dropzone_title),
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.Bold,
                )
            }
            Text(
                text = stringResource(R.string.supplier_portal_migrations_dropzone_subtitle),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Text(
                text = stringResource(R.string.supplier_portal_migrations_resumable_hint),
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            if (state.uploading) {
                val progress = if (state.totalUploadCount > 0) {
                    state.uploadedCount.toFloat() / state.totalUploadCount.toFloat()
                } else {
                    0f
                }
                LinearProgressIndicator(
                    progress = { progress.coerceIn(0f, 1f) },
                    modifier = Modifier.fillMaxWidth(),
                )
                Text(
                    text = "${state.uploadedCount} / ${state.totalUploadCount}",
                    style = MaterialTheme.typography.labelSmall,
                    fontFamily = FontFamily.Monospace,
                )
            }
            Button(
                onClick = onPickFiles,
                enabled = !state.uploading && !state.busy,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Icon(Icons.Default.CloudUpload, contentDescription = null, modifier = Modifier.size(18.dp))
                Spacer(Modifier.width(8.dp))
                Text(stringResource(R.string.supplier_portal_migrations_select_files))
            }
        }
    }
}

@Composable
private fun BundleSummaryCard(
    bundleTree: MigrationBundleTree,
    onReview: () -> Unit,
) {
    val bundle = bundleTree.bundle
    val summary = bundle.sortSummary
    val needsReview = summary?.needsReview == true
    OutlinedCard(
        shape = RoundedCornerShape(14.dp),
        border = BorderStroke(
            width = 1.dp,
            color = if (needsReview) {
                MaterialTheme.colorScheme.error.copy(alpha = 0.6f)
            } else {
                MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.45f)
            },
        ),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.md),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = bundle.bundleId.take(12),
                        style = MaterialTheme.typography.titleSmall,
                        fontFamily = FontFamily.Monospace,
                        fontWeight = FontWeight.Bold,
                    )
                    Text(
                        text = "${bundleTree.artifacts.size} files · ${summary?.streams?.size ?: 0} streams",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                SupplierStatusChip(status = bundle.status)
            }
            if (summary?.reviewReasons?.isNotEmpty() == true) {
                Text(
                    text = summary.reviewReasons.joinToString(" · "),
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.error,
                )
            }
            OutlinedButton(
                onClick = onReview,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Icon(Icons.Default.Tune, contentDescription = null, modifier = Modifier.size(16.dp))
                Spacer(Modifier.width(6.dp))
                Text(stringResource(R.string.supplier_portal_migrations_reassign_stream))
            }
        }
    }
}

@Composable
private fun SessionCard(
    session: MigrationSession,
    selected: Boolean,
    blockedBy: List<String>,
    progress: MigrationApplyProgressEvent?,
    onSelect: () -> Unit,
    onOpenMapper: () -> Unit,
) {
    OutlinedCard(
        onClick = onSelect,
        shape = RoundedCornerShape(14.dp),
        border = BorderStroke(
            width = if (selected) 2.dp else 1.dp,
            color = if (selected) {
                MaterialTheme.colorScheme.primary
            } else {
                MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.45f)
            },
        ),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.md),
            verticalArrangement = Arrangement.spacedBy(6.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = stringResource(MigrationPack.labelResFor(session.pack)),
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = FontWeight.Bold,
                    )
                    Text(
                        text = "${session.fileName}${session.sheetName?.let { " [$it]" } ?: ""}",
                        style = MaterialTheme.typography.bodySmall,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
                SupplierStatusChip(status = session.status)
            }
            Text(
                text = "${session.totalRows} rows · ${session.sessionId.take(10)}",
                style = MaterialTheme.typography.labelSmall,
                fontFamily = FontFamily.Monospace,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            if (blockedBy.isNotEmpty()) {
                Text(
                    text = "Blocked by: ${blockedBy.joinToString(", ")}",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.error,
                )
            }
            if (progress != null && progress.totalRows > 0) {
                LinearProgressIndicator(
                    progress = { (progress.processedRows.toFloat() / progress.totalRows.toFloat()).coerceIn(0f, 1f) },
                    modifier = Modifier.fillMaxWidth(),
                )
                Text(
                    text = "${progress.processedRows} / ${progress.totalRows}",
                    style = MaterialTheme.typography.labelSmall,
                    fontFamily = FontFamily.Monospace,
                )
            }
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                OutlinedButton(
                    onClick = onOpenMapper,
                    modifier = Modifier.weight(1f),
                ) {
                    Text(stringResource(R.string.supplier_portal_migrations_mapper_title))
                }
                FilledTonalButton(onClick = onSelect) {
                    Text(if (selected) stringResource(R.string.common_action_selected) else "Select")
                }
            }
        }
    }
}

@Composable
private fun DryRunAndCutoverSection(
    state: MigrationUiState,
    session: MigrationSession,
    progress: MigrationApplyProgressEvent?,
    onApplyModeChange: (String) -> Unit,
    onConfirmSetOnHandChange: (Boolean) -> Unit,
    onRunProjectDryRun: () -> Unit,
    onRunSessionDryRun: () -> Unit,
    onBuildQuarantineCsv: () -> Unit,
    onShareCsv: (String) -> Unit,
    onApprove: () -> Unit,
    onApply: () -> Unit,
    onRollback: () -> Unit,
) {
    OutlinedCard(
        shape = RoundedCornerShape(16.dp),
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.5f)),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
        ) {
            Text(
                text = stringResource(R.string.mobile_supplier_migrations_dry_run_summary),
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.Bold,
            )
            Text(
                text = "${session.fileName} (${session.status})",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )

            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                FilterChip(
                    selected = state.applyMode == "MERGE",
                    onClick = { onApplyModeChange("MERGE") },
                    label = { Text("MERGE") },
                )
                FilterChip(
                    selected = state.applyMode == "SET_ON_HAND",
                    onClick = { onApplyModeChange("SET_ON_HAND") },
                    label = { Text("SET_ON_HAND") },
                )
            }

            if (state.applyMode == "SET_ON_HAND") {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    Checkbox(
                        checked = state.setOnHandConfirmed,
                        onCheckedChange = onConfirmSetOnHandChange,
                    )
                    Text(
                        text = stringResource(R.string.supplier_portal_migrations_set_on_hand_ack),
                        style = MaterialTheme.typography.bodySmall,
                        modifier = Modifier.weight(1f),
                    )
                }
            }

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Button(
                    onClick = onRunProjectDryRun,
                    enabled = !state.busy,
                    modifier = Modifier.weight(1f),
                ) {
                    Text(stringResource(R.string.supplier_portal_migrations_run_dry_run))
                }
                OutlinedButton(
                    onClick = onRunSessionDryRun,
                    enabled = !state.busy,
                ) {
                    Text("Session Only")
                }
            }

            state.dryRunResult?.let { dryRun ->
                val summary = dryRun.summary
                HorizontalDivider()
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        text = stringResource(R.string.supplier_portal_migrations_dry_run_title),
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = FontWeight.Bold,
                    )
                    SupplierStatusChip(
                        status = if (summary.balancedJournalCheck) {
                            stringResource(R.string.supplier_portal_migrations_balance_status_balanced)
                        } else {
                            stringResource(R.string.supplier_portal_migrations_balance_status_unbalanced)
                        },
                    )
                }
                Text(
                    text = "${stringResource(R.string.supplier_portal_migrations_debits_total)}: ${formatMinorAmount(summary.journalDebitMinor, dryRun.report.currency)} · " +
                        "${stringResource(R.string.supplier_portal_migrations_credits_total)}: ${formatMinorAmount(summary.journalCreditMinor, dryRun.report.currency)}",
                    style = MaterialTheme.typography.labelMedium,
                    fontFamily = FontFamily.Monospace,
                )
                if (summary.coldLotWarningCount > 0L) {
                    PegasusRuntimeBanner(
                        tone = PegasusRuntimeTone.Warning,
                        message = "Cold-chain notice: ${summary.coldLotWarningCount} lots use cold_lot_migrated_without_live_baseline.",
                    )
                }
                if (summary.onHandBelowReservedHits > 0L) {
                    PegasusRuntimeBanner(
                        tone = PegasusRuntimeTone.Warning,
                        message = "Stock notice: ${summary.onHandBelowReservedHits} rows fell below active reserved quantities.",
                    )
                }
                summary.packs.forEach { packReport ->
                    Text(
                        text = "${stringResource(MigrationPack.labelResFor(packReport.pack))}: +${packReport.wouldCreate} create / ~${packReport.wouldUpdate} update / ${packReport.wouldSkip} skip (${packReport.blockingCount} blocking)",
                        style = MaterialTheme.typography.labelSmall,
                        fontFamily = FontFamily.Monospace,
                    )
                }
                if (summary.conflicts.isNotEmpty()) {
                    Text(
                        text = "${stringResource(R.string.supplier_portal_migrations_errors_count)}: ${summary.conflicts.size}",
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.error,
                        fontWeight = FontWeight.Bold,
                    )
                    summary.conflicts.take(8).forEach { conflict ->
                        Text(
                            text = "Row ${conflict.rowIndex} [${conflict.severity}] ${conflict.code}: ${conflict.message}",
                            style = MaterialTheme.typography.labelSmall,
                            color = if (conflict.severity.equals("BLOCKING", ignoreCase = true)) {
                                MaterialTheme.colorScheme.error
                            } else {
                                MaterialTheme.colorScheme.onSurfaceVariant
                            },
                        )
                    }
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        OutlinedButton(onClick = onBuildQuarantineCsv) {
                            Text(stringResource(R.string.supplier_portal_migrations_download_errors_csv))
                        }
                        state.quarantinedErrorsCSV?.let { csv ->
                            FilledTonalButton(onClick = { onShareCsv(csv) }) {
                                Icon(Icons.Default.Share, contentDescription = null, modifier = Modifier.size(16.dp))
                                Spacer(Modifier.width(4.dp))
                                Text("Share CSV")
                            }
                        }
                    }
                }
            }

            if (progress != null && progress.totalRows > 0) {
                LinearProgressIndicator(
                    progress = { (progress.processedRows.toFloat() / progress.totalRows.toFloat()).coerceIn(0f, 1f) },
                    modifier = Modifier.fillMaxWidth(),
                )
                Text(
                    text = "${stringResource(R.string.supplier_portal_migrations_apply_progress)}: ${progress.processedRows}/${progress.totalRows}",
                    style = MaterialTheme.typography.labelSmall,
                    fontFamily = FontFamily.Monospace,
                )
            }

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                OutlinedButton(
                    onClick = onApprove,
                    enabled = !state.busy && (state.applyMode != "SET_ON_HAND" || state.setOnHandConfirmed),
                    modifier = Modifier.weight(1f),
                ) {
                    Text(stringResource(R.string.mobile_supplier_migrations_approve_action))
                }
                Button(
                    onClick = onApply,
                    enabled = !state.busy,
                    modifier = Modifier.weight(1f),
                ) {
                    Icon(Icons.Default.PlayArrow, contentDescription = null, modifier = Modifier.size(16.dp))
                    Spacer(Modifier.width(4.dp))
                    Text(stringResource(R.string.mobile_supplier_migrations_apply_live))
                }
            }

            if (session.rollbackUntil != null || session.status == "APPLIED") {
                OutlinedButton(
                    onClick = onRollback,
                    enabled = !state.busy,
                    colors = ButtonDefaults.outlinedButtonColors(contentColor = MaterialTheme.colorScheme.error),
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Icon(Icons.AutoMirrored.Filled.Undo, contentDescription = null, modifier = Modifier.size(16.dp))
                    Spacer(Modifier.width(6.dp))
                    Text("${stringResource(R.string.mobile_supplier_migrations_rollback_action)} (${stringResource(R.string.supplier_portal_migrations_rollback_window_hours)})")
                }
            }

            state.applyResult?.let { res ->
                PegasusRuntimeBanner(
                    tone = PegasusRuntimeTone.Live,
                    message = "Applied: +${res.created} created, ~${res.updated} updated, ${res.skipped} skipped.",
                )
            }
            state.rollbackResult?.let { rb ->
                PegasusRuntimeBanner(
                    tone = PegasusRuntimeTone.Warning,
                    message = "Rolled back: ${rb.revertedSnapshots} snapshots, ${rb.reversedInvoices} invoices, ${rb.supersededOrders} orders.",
                )
            }
        }
    }
}

@Composable
private fun OpenOrdersReleaseCard(
    state: MigrationUiState,
    onOrderIdsChange: (String) -> Unit,
    onTargetStatusChange: (String) -> Unit,
    onRelease: () -> Unit,
) {
    OutlinedCard(
        shape = RoundedCornerShape(16.dp),
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.5f)),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
        ) {
            Text(
                text = stringResource(R.string.supplier_portal_migrations_open_orders_title),
                style = MaterialTheme.typography.titleSmall,
                fontWeight = FontWeight.Bold,
            )
            OutlinedTextField(
                value = state.openOrderIdsText,
                onValueChange = onOrderIdsChange,
                label = { Text("Held MIGRATION_OPEN order IDs (comma separated)") },
                modifier = Modifier.fillMaxWidth(),
                minLines = 2,
            )
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                FilterChip(
                    selected = state.releaseTargetStatus == "PENDING",
                    onClick = { onTargetStatusChange("PENDING") },
                    label = { Text("PENDING") },
                )
                FilterChip(
                    selected = state.releaseTargetStatus == "CONFIRMED",
                    onClick = { onTargetStatusChange("CONFIRMED") },
                    label = { Text("CONFIRMED") },
                )
            }
            Button(
                onClick = onRelease,
                enabled = !state.busy && state.openOrderIdsText.isNotBlank(),
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(stringResource(R.string.supplier_portal_migrations_release_orders_button))
            }
            state.releaseResult?.let { rel ->
                PegasusRuntimeBanner(
                    tone = PegasusRuntimeTone.Live,
                    message = "Released ${rel.releasedOrderIds.size}, skipped ${rel.skippedOrderIds.size}",
                )
            }
        }
    }
}

@Composable
private fun CutoverCard(
    state: MigrationUiState,
    onCutover: () -> Unit,
) {
    OutlinedCard(
        shape = RoundedCornerShape(16.dp),
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.5f)),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
        ) {
            Text(
                text = stringResource(R.string.supplier_portal_migrations_cutover_title),
                style = MaterialTheme.typography.titleSmall,
                fontWeight = FontWeight.Bold,
            )
            state.cutoverResult?.checklist?.forEach { item ->
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    Icon(
                        imageVector = if (item.passed) Icons.Default.CheckCircle else Icons.Default.Warning,
                        contentDescription = null,
                        tint = if (item.passed) MaterialTheme.colorScheme.tertiary else MaterialTheme.colorScheme.error,
                        modifier = Modifier.size(18.dp),
                    )
                    Column(modifier = Modifier.weight(1f)) {
                        Text(item.name, style = MaterialTheme.typography.bodySmall, fontWeight = FontWeight.SemiBold)
                        if (item.detail.isNotBlank()) {
                            Text(
                                item.detail,
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                    }
                }
            }
            Button(
                onClick = onCutover,
                enabled = !state.busy,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(stringResource(R.string.supplier_portal_migrations_cutover_button))
            }
            if (state.cutoverResult != null) {
                PegasusRuntimeBanner(
                    tone = PegasusRuntimeTone.Live,
                    message = stringResource(R.string.supplier_portal_migrations_cutover_ready),
                )
            }
        }
    }
}

@Composable
private fun CreateProjectDialog(
    state: MigrationUiState,
    onNameChange: (String) -> Unit,
    onWarehouseChange: (String) -> Unit,
    onDismiss: () -> Unit,
    onConfirm: () -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.supplier_portal_migrations_create_project)) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                OutlinedTextField(
                    value = state.newProjectName,
                    onValueChange = onNameChange,
                    label = { Text(stringResource(R.string.supplier_portal_migrations_project_name)) },
                    placeholder = { Text(stringResource(R.string.supplier_portal_migrations_project_name_placeholder)) },
                    modifier = Modifier.fillMaxWidth(),
                )
                if (state.warehouses.isNotEmpty()) {
                    Text(
                        text = stringResource(R.string.supplier_portal_migrations_warehouse_label),
                        style = MaterialTheme.typography.labelMedium,
                    )
                    LazyRow(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                        items(state.warehouses, key = { it.warehouseId }) { wh ->
                            FilterChip(
                                selected = state.newProjectWarehouseId == wh.warehouseId,
                                onClick = {
                                    onWarehouseChange(
                                        if (state.newProjectWarehouseId == wh.warehouseId) "" else wh.warehouseId,
                                    )
                                },
                                label = { Text(wh.name.ifBlank { wh.warehouseId.take(8) }) },
                            )
                        }
                    }
                }
            }
        },
        confirmButton = {
            Button(
                onClick = onConfirm,
                enabled = !state.busy && state.newProjectName.isNotBlank(),
            ) {
                Text(
                    if (state.busy) {
                        stringResource(R.string.supplier_portal_migrations_creating)
                    } else {
                        stringResource(R.string.supplier_portal_migrations_create_button)
                    },
                )
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text(stringResource(R.string.common_action_cancel))
            }
        },
    )
}

@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)
@Composable
private fun AutoSortReviewDialog(
    state: MigrationUiState,
    bundleTree: MigrationBundleTree,
    onDismiss: () -> Unit,
    onModeChange: (String) -> Unit,
    onPackOverride: (String, String) -> Unit,
    onEncodingOverride: (String, String) -> Unit,
    onIgnoreChange: (String, Boolean) -> Unit,
    onSave: () -> Unit,
) {
    val summary = bundleTree.bundle.sortSummary
    Dialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(usePlatformDefaultWidth = false),
    ) {
        Scaffold(
            topBar = {
                TopAppBar(
                    title = { Text(stringResource(R.string.supplier_portal_migrations_sorter_title)) },
                    navigationIcon = {
                        IconButton(onClick = onDismiss) {
                            Icon(Icons.Default.Close, contentDescription = stringResource(R.string.common_action_close))
                        }
                    },
                    actions = {
                        TextButton(onClick = onSave, enabled = !state.busy) {
                            Text(stringResource(R.string.supplier_portal_migrations_apply_sort_override))
                        }
                    },
                )
            },
        ) { padding ->
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(padding)
                    .verticalScroll(rememberScrollState())
                    .padding(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                if (summary?.conflictingInventoryMode == true) {
                    PegasusRuntimeBanner(
                        tone = PegasusRuntimeTone.Warning,
                        message = stringResource(R.string.supplier_portal_migrations_stock_vs_lots_warning),
                    )
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        FilterChip(
                            selected = state.resolvedInventoryMode == "lots",
                            onClick = { onModeChange("lots") },
                            label = { Text(stringResource(R.string.supplier_portal_migrations_pack_lots)) },
                        )
                        FilterChip(
                            selected = state.resolvedInventoryMode == "stock",
                            onClick = { onModeChange("stock") },
                            label = { Text(stringResource(R.string.supplier_portal_migrations_pack_stock)) },
                        )
                    }
                }

                val streams = summary?.streams.orEmpty()
                streams.forEach { stream ->
                    val chosenPack = state.packOverrides[stream.id] ?: stream.detectedPacks.firstOrNull().orEmpty()
                    val chosenEnc = state.encodingOverrides[stream.id] ?: stream.encoding.ifBlank { "utf-8" }
                    val ignored = state.ignoredStreams[stream.id] ?: stream.ignored
                    OutlinedCard(
                        shape = RoundedCornerShape(14.dp),
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Column(
                            modifier = Modifier.padding(PegasusSpacing.md),
                            verticalArrangement = Arrangement.spacedBy(8.dp),
                        ) {
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                            ) {
                                Text(
                                    text = "${stream.fileName} (${stream.streamKey})",
                                    style = MaterialTheme.typography.titleSmall,
                                    fontWeight = FontWeight.Bold,
                                    modifier = Modifier.weight(1f),
                                )
                                Text(
                                    text = "${(stream.confidence * 100.0).roundToInt()}%",
                                    style = MaterialTheme.typography.labelMedium,
                                    fontFamily = FontFamily.Monospace,
                                )
                            }
                            if (stream.headers.isNotEmpty()) {
                                Text(
                                    text = stream.headers.joinToString(", "),
                                    style = MaterialTheme.typography.labelSmall,
                                    fontFamily = FontFamily.Monospace,
                                    maxLines = 2,
                                    overflow = TextOverflow.Ellipsis,
                                )
                            }
                            FlowRow(
                                horizontalArrangement = Arrangement.spacedBy(6.dp),
                                verticalArrangement = Arrangement.spacedBy(4.dp),
                            ) {
                                MigrationPack.entries.forEach { pack ->
                                    FilterChip(
                                        selected = chosenPack == pack.rawValue,
                                        onClick = { onPackOverride(stream.id, pack.rawValue) },
                                        label = { Text(stringResource(pack.labelRes)) },
                                    )
                                }
                            }
                            Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                                MigrationEncodingOptions.all.forEach { enc ->
                                    FilterChip(
                                        selected = chosenEnc == enc,
                                        onClick = { onEncodingOverride(stream.id, enc) },
                                        label = { Text(enc) },
                                    )
                                }
                            }
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                Text(
                                    text = stringResource(R.string.supplier_portal_migrations_ignore_column),
                                    style = MaterialTheme.typography.bodySmall,
                                )
                                Switch(
                                    checked = ignored,
                                    onCheckedChange = { onIgnoreChange(stream.id, it) },
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ColumnMapperDialog(
    state: MigrationUiState,
    samplePreview: (String) -> String,
    onSelectTarget: (String, String) -> Unit,
    onSaveTemplateToggle: (Boolean) -> Unit,
    onAcceptHighConfidence: () -> Unit,
    onConfirm: () -> Unit,
    onDismiss: () -> Unit,
) {
    var pickingForColumn by remember { mutableStateOf<String?>(null) }
    var showSampleTable by remember { mutableStateOf(false) }

    Dialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(usePlatformDefaultWidth = false),
    ) {
        Scaffold(
            topBar = {
                TopAppBar(
                    title = { Text(stringResource(R.string.supplier_portal_migrations_mapper_title)) },
                    navigationIcon = {
                        IconButton(onClick = onDismiss) {
                            Icon(Icons.Default.Close, contentDescription = stringResource(R.string.common_action_close))
                        }
                    },
                    actions = {
                        TextButton(
                            onClick = onConfirm,
                            enabled = !state.mappingSaving && !state.mappingLoading,
                        ) {
                            Text(stringResource(R.string.supplier_portal_migrations_confirm_mapping))
                        }
                    },
                )
            },
        ) { padding ->
            if (state.mappingLoading) {
                Box(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(padding),
                    contentAlignment = Alignment.Center,
                ) {
                    CircularProgressIndicator()
                }
            } else {
                Column(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(padding)
                        .verticalScroll(rememberScrollState())
                        .padding(PegasusSpacing.lg),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                ) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        OutlinedButton(
                            onClick = onAcceptHighConfidence,
                            enabled = !state.mappingSaving,
                            modifier = Modifier.weight(1f),
                        ) {
                            Text(stringResource(R.string.supplier_portal_migrations_accept_high_confidence))
                        }
                        if (state.sampleRows.isNotEmpty()) {
                            FilledTonalButton(onClick = { showSampleTable = !showSampleTable }) {
                                Text(if (showSampleTable) "Hide Samples" else "20 Rows")
                            }
                        }
                    }

                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text(
                            text = stringResource(R.string.supplier_portal_migrations_save_template_toggle),
                            style = MaterialTheme.typography.bodySmall,
                        )
                        Switch(
                            checked = state.saveMappingTemplate,
                            onCheckedChange = onSaveTemplateToggle,
                        )
                    }

                    state.mappingDoc?.missingRequired?.takeIf { it.isNotEmpty() }?.let { missing ->
                        PegasusRuntimeBanner(
                            tone = PegasusRuntimeTone.Warning,
                            message = "Missing required: ${missing.joinToString(", ")}",
                        )
                    }

                    if (showSampleTable && state.sampleRows.isNotEmpty()) {
                        val columns = state.proposals.map { it.sourceColumn }
                        OutlinedCard(shape = RoundedCornerShape(12.dp)) {
                            Column(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .horizontalScroll(rememberScrollState())
                                    .padding(PegasusSpacing.md),
                                verticalArrangement = Arrangement.spacedBy(6.dp),
                            ) {
                                Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                                    columns.forEach { col ->
                                        Text(
                                            text = col,
                                            style = MaterialTheme.typography.labelMedium,
                                            fontWeight = FontWeight.Bold,
                                            modifier = Modifier.width(120.dp),
                                        )
                                    }
                                }
                                HorizontalDivider()
                                state.sampleRows.take(20).forEach { row ->
                                    Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                                        columns.forEach { col ->
                                            Text(
                                                text = row.textValue(col).ifBlank { "—" },
                                                style = MaterialTheme.typography.labelSmall,
                                                fontFamily = FontFamily.Monospace,
                                                maxLines = 1,
                                                overflow = TextOverflow.Ellipsis,
                                                modifier = Modifier.width(120.dp),
                                            )
                                        }
                                    }
                                }
                            }
                        }
                    }

                    state.proposals.forEach { prop ->
                        val targetMeta = state.taxonomy.firstOrNull { it.key == prop.targetField }
                        OutlinedCard(
                            shape = RoundedCornerShape(14.dp),
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            Column(
                                modifier = Modifier.padding(PegasusSpacing.md),
                                verticalArrangement = Arrangement.spacedBy(6.dp),
                            ) {
                                Row(
                                    modifier = Modifier.fillMaxWidth(),
                                    horizontalArrangement = Arrangement.SpaceBetween,
                                ) {
                                    Text(
                                        text = prop.sourceColumn,
                                        style = MaterialTheme.typography.titleSmall,
                                        fontWeight = FontWeight.Bold,
                                        modifier = Modifier.weight(1f),
                                    )
                                    Text(
                                        text = "${(prop.confidence * 100.0).roundToInt()}%",
                                        style = MaterialTheme.typography.labelMedium,
                                        fontFamily = FontFamily.Monospace,
                                    )
                                }
                                Text(
                                    text = "${stringResource(R.string.supplier_portal_migrations_sample_values)}: ${samplePreview(prop.sourceColumn)}",
                                    style = MaterialTheme.typography.labelSmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                    maxLines = 2,
                                    overflow = TextOverflow.Ellipsis,
                                )
                                OutlinedButton(
                                    onClick = { pickingForColumn = prop.sourceColumn },
                                    modifier = Modifier.fillMaxWidth(),
                                ) {
                                    Text(
                                        text = targetMeta?.let { "${it.localizedLabel()} (${it.key})" }
                                            ?: prop.targetField.ifBlank {
                                                stringResource(R.string.supplier_portal_migrations_ignore_column)
                                            },
                                    )
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    pickingForColumn?.let { column ->
        SearchableTargetFieldDialog(
            sourceColumn = column,
            taxonomy = state.taxonomy,
            currentTarget = state.proposals.firstOrNull { it.sourceColumn == column }?.targetField.orEmpty(),
            onSelect = { key ->
                onSelectTarget(column, key)
                pickingForColumn = null
            },
            onDismiss = { pickingForColumn = null },
        )
    }
}

@Composable
private fun SearchableTargetFieldDialog(
    sourceColumn: String,
    taxonomy: List<MigrationCanonicalField>,
    currentTarget: String,
    onSelect: (String) -> Unit,
    onDismiss: () -> Unit,
) {
    var query by remember { mutableStateOf("") }
    val filtered = remember(query, taxonomy) {
        val needle = query.trim().lowercase()
        if (needle.isEmpty()) {
            taxonomy
        } else {
            taxonomy.filter {
                it.key.lowercase().contains(needle) ||
                    it.localizedLabel().lowercase().contains(needle)
            }
        }
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("${stringResource(R.string.supplier_portal_migrations_target_field)}: $sourceColumn") },
        text = {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .heightIn(max = 420.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                OutlinedTextField(
                    value = query,
                    onValueChange = { query = it },
                    leadingIcon = { Icon(Icons.Default.Search, contentDescription = null) },
                    placeholder = { Text(stringResource(R.string.supplier_portal_migrations_target_field)) },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                )
                LazyColumn(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                    item {
                        TextButton(
                            onClick = { onSelect("") },
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            Text(
                                text = stringResource(R.string.supplier_portal_migrations_ignore_column),
                                modifier = Modifier.fillMaxWidth(),
                            )
                        }
                    }
                    items(filtered, key = { it.key }) { field ->
                        val isSelected = field.key == currentTarget
                        TextButton(
                            onClick = { onSelect(field.key) },
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            Column(modifier = Modifier.fillMaxWidth()) {
                                Text(
                                    text = "${field.localizedLabel()}${if (field.required) " *" else ""}",
                                    fontWeight = if (isSelected) FontWeight.Bold else FontWeight.Normal,
                                )
                                Text(
                                    text = "${field.key} · ${field.type}",
                                    style = MaterialTheme.typography.labelSmall,
                                    fontFamily = FontFamily.Monospace,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            }
                        }
                    }
                }
            }
        },
        confirmButton = {
            TextButton(onClick = onDismiss) {
                Text(stringResource(R.string.common_action_close))
            }
        },
    )
}
