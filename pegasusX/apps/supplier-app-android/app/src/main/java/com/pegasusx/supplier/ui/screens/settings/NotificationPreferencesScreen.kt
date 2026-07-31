package com.pegasusx.supplier.ui.screens.settings

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasus.design.PegasusLoadingState
import com.pegasusx.supplier.data.model.NotificationPreferenceRow
import com.pegasusx.supplier.data.model.NotificationPreferencesPatchRequest
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun NotificationPreferencesScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
    var loading by remember { mutableStateOf(true) }
    var prefs by remember { mutableStateOf<List<NotificationPreferenceRow>>(emptyList()) }
    var saved by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            try {
                val resp = ops.getNotificationPreferences()
                if (resp.isSuccessful) {
                    prefs = resp.body()?.preferences ?: emptyList()
                }
            } finally {
                loading = false
            }
        }
    }

    fun save() {
        scope.launch {
            val resp = ops.patchNotificationPreferences(NotificationPreferencesPatchRequest(prefs))
            if (resp.isSuccessful) saved = true
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Notification preferences") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
                actions = {
                    TextButton(onClick = { save() }) { Text("Save") }
                },
            )
        },
    ) { padding ->
        when {
            loading -> PegasusLoadingState("Loading…", modifier = Modifier.padding(padding))
            else -> LazyColumn(
                modifier = Modifier.padding(padding).padding(PegasusSpacing.md),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                if (saved) {
                    item { Text("Saved", color = MaterialTheme.colorScheme.primary) }
                }
                items(prefs.size) { idx ->
                    val p = prefs[idx]
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(PegasusSpacing.md)) {
                            Text(p.eventType, style = MaterialTheme.typography.labelMedium)
                            Text(p.channel, style = MaterialTheme.typography.bodySmall)
                            Row {
                                Checkbox(
                                    checked = p.enabled,
                                    onCheckedChange = { checked ->
                                        prefs = prefs.mapIndexed { i, row ->
                                            if (i == idx) row.copy(enabled = checked) else row
                                        }
                                    },
                                )
                                Text("Enabled")
                            }
                        }
                    }
                }
            }
        }
    }
}
