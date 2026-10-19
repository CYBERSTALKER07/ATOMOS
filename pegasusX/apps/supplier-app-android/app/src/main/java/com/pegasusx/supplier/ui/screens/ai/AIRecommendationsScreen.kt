package com.pegasusx.supplier.ui.screens.ai

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SupplierAIRecommendation
import com.pegasusx.supplier.data.model.SupplierAIRecommendationDecisionRequest
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasus.design.ui.PegasusLoadingState
import com.pegasus.design.ui.PegasusStateKind
import com.pegasus.design.ui.PegasusStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import java.util.UUID
import com.pegasusx.supplier.R

private val statusFilters = listOf("PENDING", "ACKNOWLEDGED", "OVERRIDDEN", "DISMISSED", "ALL")
private val decisionActions = listOf("ACKNOWLEDGED", "OVERRIDDEN", "DISMISSED", "REOPENED")

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AIRecommendationsScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
    var filter by remember { mutableStateOf("PENDING") }
    var items by remember { mutableStateOf<List<SupplierAIRecommendation>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var pendingDecisionId by remember { mutableStateOf<String?>(null) }
    var feedback by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.getAiRecommendations(status = filter, limit = 50)
                if (resp.isSuccessful) items = resp.body()?.items ?: emptyList()
                else error = "Failed (${resp.code()})"
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    fun record(rec: SupplierAIRecommendation, decision: String) {
        scope.launch {
            pendingDecisionId = rec.recommendationId
            feedback = null
            try {
                val key = "ai-rec-${rec.recommendationId}-$decision-${UUID.randomUUID()}"
                val resp = ops.recordAiRecommendationDecision(
                    SupplierAIRecommendationDecisionRequest(rec.recommendationId, decision, null),
                    key,
                )
                if (resp.isSuccessful) {
                    val updated = resp.body()?.recommendation
                    if (updated != null) {
                        items = items.map { if (it.recommendationId == updated.recommendationId) updated else it }
                    }
                    feedback = "${decision.lowercase()} recorded for ${rec.aggregateType} ${rec.aggregateId}."
                } else {
                    feedback = "Decision failed (${resp.code()})"
                }
            } catch (e: Exception) {
                feedback = "Decision failed: ${e.message}"
            } finally {
                pendingDecisionId = null
            }
        }
    }

    LaunchedEffect(filter) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("AI recommendations") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
            )
        },
    ) { padding ->
        Column(modifier = Modifier.padding(padding).fillMaxSize()) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .horizontalScrollPadding(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                statusFilters.forEach { status ->
                    FilterChip(
                        selected = filter == status,
                        onClick = { filter = status },
                        label = { Text(status) },
                    )
                }
            }
            feedback?.let {
                Text(
                    it,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.outline,
                    modifier = Modifier.padding(horizontal = PegasusSpacing.lg, vertical = PegasusSpacing.sm),
                )
            }
            when {
                loading -> PegasusLoadingState("Loading recommendations…", "AI advisory")
                error != null -> PegasusStatePane(
                    kind = PegasusStateKind.Error,
                    headline = "Recommendations unavailable",
                    body = error!!,
                    actionLabel = "Retry",
                    onAction = { load() },
                )
                items.isEmpty() -> PegasusStatePane(
                    kind = PegasusStateKind.Empty,
                    headline = "No recommendations",
                    body = "No ${filter.lowercase()} advisory rows for this supplier.",
                )
                else -> LazyColumn(
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(PegasusSpacing.lg),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                ) {
                    items(items, key = { it.recommendationId }) { rec ->
                        RecommendationCard(rec, pendingDecisionId != null, ::record)
                    }
                }
            }
        }
    }
}

private fun Modifier.horizontalScrollPadding(): Modifier =
    this.padding(horizontal = PegasusSpacing.lg, vertical = PegasusSpacing.sm)

@Composable
private fun RecommendationCard(
    rec: SupplierAIRecommendation,
    decisionBusy: Boolean,
    onDecision: (SupplierAIRecommendation, String) -> Unit,
) {
    ElevatedCard(Modifier.fillMaxWidth()) {
        Column(
            Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
        ) {
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                Text(rec.action.ifEmpty { "Recommendation" }, style = MaterialTheme.typography.titleMedium)
                Text(rec.status, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.outline)
            }
            Text(
                "Confidence %.0f%% · Score %.2f".format(rec.confidence * 100, rec.score),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.outline,
            )
            if (rec.explanation.isNotEmpty()) {
                Text(rec.explanation, style = MaterialTheme.typography.bodyMedium)
            }
            Text(
                stringResource(R.string.mobile_supplier_ui_aggregatetype_aggregateid_source_source, rec.aggregateType, rec.aggregateId, rec.source),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.outline,
            )
            if (rec.reasonCodes.isNotEmpty()) {
                Text(stringResource(R.string.mobile_supplier_ui_reasons_jointostring, rec.reasonCodes.joinToString(", ")),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.outline,
                )
            }
            rec.evidence.forEach { ev ->
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                    Text(ev.label, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.outline)
                    Text(ev.value, style = MaterialTheme.typography.bodySmall)
                }
            }
            val decided = !rec.decision.isNullOrEmpty()
            if (decided) {
                Text(
                    "Decision: ${rec.decision} by ${rec.decidedBy ?: "operator"}",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.outline,
                )
            } else {
                FlowRowActions(decisionBusy) { decision -> onDecision(rec, decision) }
            }
        }
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun FlowRowActions(busy: Boolean, onDecision: (String) -> Unit) {
    FlowRow(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
        decisionActions.forEach { decision ->
            OutlinedButton(onClick = { onDecision(decision) }, enabled = !busy) {
                Text(decision.lowercase().replaceFirstChar { it.uppercase() })
            }
        }
    }
}
