package com.pegasusx.supplier.ui.screens.planning

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.KnowledgeGraphNode
import com.pegasusx.supplier.data.model.SupplierKnowledgeGraph
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierOpsListCard
import com.pegasusx.supplier.ui.components.SupplierSectionTitle
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun KnowledgeGraphScreen(
    ops: SupplierOperationsRepository,
    onBack: () -> Unit,
) {
    var graph by remember { mutableStateOf<SupplierKnowledgeGraph?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            val resp = ops.getKnowledgeGraph()
            if (resp.isSuccessful) {
                graph = resp.body()
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
                title = { Text("Knowledge graph") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> SupplierLoadingState("Loading knowledge graph…", modifier = Modifier.padding(padding))
            error != null -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Knowledge graph unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            graph == null -> SupplierStatePane(
                kind = SupplierStateKind.Empty,
                headline = "No graph data",
                body = "Planning entities will appear when the graph is populated.",
                modifier = Modifier.padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier.padding(padding),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                item {
                    SupplierSectionTitle("Nodes (${graph!!.nodes.size})")
                }
                items(graph!!.nodes, key = KnowledgeGraphNode::id) { node ->
                    SupplierOpsListCard(
                        headline = node.name?.ifBlank { node.id } ?: node.id,
                        supporting = "${node.type} · ${node.id}",
                    )
                }
                item {
                    SupplierSectionTitle("Edges (${graph!!.edges.size})")
                }
                items(graph!!.edges, key = { "${it.from}-${it.to}-${it.relation}" }) { edge ->
                    SupplierOpsListCard(
                        headline = edge.relation,
                        supporting = "${edge.from} → ${edge.to}",
                    )
                }
            }
        }
    }
}
