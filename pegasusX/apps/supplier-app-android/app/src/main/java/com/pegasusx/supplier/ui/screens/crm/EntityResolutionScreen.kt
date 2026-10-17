package com.pegasusx.supplier.ui.screens.crm

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
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.EntityResolutionExplainRequest
import com.pegasusx.supplier.data.model.EntityResolutionExplainResponse
import com.pegasusx.supplier.data.model.EntityResolutionResolveRequest
import com.pegasusx.supplier.data.model.EntityResolutionResolveResponse
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

private val ENTITY_TYPES = listOf("ANY", "ORDER", "RETAILER", "WAREHOUSE", "FACTORY", "DRIVER", "VEHICLE", "SUPPLIER")

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun EntityResolutionScreen(
    ops: SupplierOperationsRepository,
    onBack: () -> Unit,
) {
    var entityType by remember { mutableStateOf("ANY") }
    var query by remember { mutableStateOf("") }
    var entityId by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var resolved by remember { mutableStateOf<EntityResolutionResolveResponse?>(null) }
    var explain by remember { mutableStateOf<EntityResolutionExplainResponse?>(null) }
    val scope = rememberCoroutineScope()

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Entity resolution") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        LazyColumn(
            modifier = Modifier.fillMaxSize().padding(padding),
            contentPadding = PaddingValues(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            item {
                Text(
                    "Typed resolve/explain against the supplier graph. ADMIN JWT. Not a search box over raw Spanner.",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            item {
                Text("Type", style = MaterialTheme.typography.labelMedium)
                ENTITY_TYPES.forEach { t ->
                    OutlinedButton(onClick = { entityType = t }) {
                        Text(if (entityType == t) "● $t" else t)
                    }
                }
            }
            item {
                OutlinedTextField(
                    value = query,
                    onValueChange = { query = it },
                    label = { Text("Query") },
                    modifier = Modifier.fillMaxWidth(),
                )
            }
            item {
                OutlinedTextField(
                    value = entityId,
                    onValueChange = { entityId = it },
                    label = { Text("Entity ID") },
                    modifier = Modifier.fillMaxWidth(),
                )
            }
            item {
                Button(
                    onClick = {
                        busy = true
                        error = null
                        explain = null
                        scope.launch {
                            val resp = ops.resolveEntity(
                                EntityResolutionResolveRequest(
                                    entityType = entityType,
                                    query = query.trim().ifBlank { null },
                                    entityId = entityId.trim().ifBlank { null },
                                ),
                            )
                            if (resp.isSuccessful) {
                                resolved = resp.body()
                            } else {
                                error = "Resolve failed (${resp.code()})"
                                resolved = null
                            }
                            busy = false
                        }
                    },
                    enabled = !busy,
                ) { Text(if (busy) "Resolving…" else "Resolve") }
            }
            error?.let { item { Text(it, color = MaterialTheme.colorScheme.error) } }
            resolved?.let { result ->
                item {
                    Text("Requested ${result.requestedType} · ${result.candidates.size} candidates")
                }
                item {
                    val top = result.resolved
                    Text(
                        if (top != null) "Top: ${top.label} (${top.entityId}) score ${top.score}"
                        else "No deterministic match",
                    )
                }
                items(result.candidates, key = { it.nodeId }) { c ->
                    Column(Modifier.fillMaxWidth()) {
                        Text("${c.label} · ${c.entityType}/${c.entityId} · ${c.confidenceClass}")
                        OutlinedButton(onClick = {
                            busy = true
                            scope.launch {
                                val resp = ops.explainEntity(
                                    EntityResolutionExplainRequest(entityType = c.entityType, entityId = c.entityId),
                                )
                                explain = if (resp.isSuccessful) resp.body() else null
                                if (!resp.isSuccessful) error = "Explain failed (${resp.code()})"
                                busy = false
                            }
                        }) { Text("Explain") }
                    }
                }
            }
            explain?.let { exp ->
                item {
                    Text("Lineage for ${exp.source.label}", style = MaterialTheme.typography.titleSmall)
                }
                items(exp.projection.edges, key = { "${it.from}-${it.to}-${it.relation}" }) { e ->
                    Text("${e.relation}: ${e.from} → ${e.to}", style = MaterialTheme.typography.bodySmall)
                }
            }
        }
    }
}
