package com.pegasusx.supplier.ui.screens.ops

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
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.ElevatedCard
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
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.data.model.ControlTowerPlaybook
import com.pegasusx.supplier.data.model.ControlTowerPlaybooksResponse
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import retrofit2.Response

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PlaybooksScreen(
    onBack: () -> Unit,
    loader: suspend () -> Response<ControlTowerPlaybooksResponse>,
) {
    var rows by remember { mutableStateOf<List<ControlTowerPlaybook>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        loading = true
        error = null
        scope.launch {
            try {
                val resp = loader()
                if (resp.isSuccessful) {
                    rows = resp.body()?.playbooks.orEmpty()
                } else {
                    error = if (resp.code() == 503) {
                        "Playbooks unavailable. Enable CONTROL_TOWER_PLAYBOOKS_ENABLED."
                    } else {
                        "Failed (${resp.code()})"
                    }
                    rows = emptyList()
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
                rows = emptyList()
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Playbooks") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
                actions = {
                    IconButton(onClick = { load() }) {
                        Icon(Icons.Default.Refresh, contentDescription = "Refresh")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading && rows.isEmpty() -> PegasusLoadingState(
                title = "Loading playbooks",
                body = "GET /v1/control-tower/playbooks",
                modifier = Modifier.fillMaxSize().padding(padding),
            )
            error != null && rows.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Playbooks unavailable",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier.fillMaxSize().padding(padding),
            )
            rows.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Empty,
                headline = "No playbooks",
                body = "Empty list is {playbooks:[]} when none are configured.",
                modifier = Modifier.fillMaxSize().padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier.fillMaxSize().padding(padding),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                items(rows, key = { it.playbookId }) { row ->
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(
                            Modifier.padding(PegasusSpacing.lg),
                            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
                        ) {
                            Text(row.name.ifBlank { row.playbookId }, style = MaterialTheme.typography.titleSmall)
                            Text(
                                "${if (row.isActive) "active" else "inactive"} · priority ${row.priority}" +
                                    if (row.autoExecute) " · auto" else "",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                            if (row.description.isNotBlank()) {
                                Text(row.description, style = MaterialTheme.typography.bodySmall)
                            }
                        }
                    }
                }
            }
        }
    }
}
