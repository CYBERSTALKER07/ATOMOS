package com.pegasus.retailer.ui.screens.profile

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.ArrowBack
import androidx.compose.material.icons.rounded.AutoAwesome
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasus.retailer.ui.components.PegasusEmptyState

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AutoOrderScreen(
    onNavigateBack: () -> Unit,
    viewModel: AutoOrderViewModel = hiltViewModel()
) {
    val uiState by viewModel.uiState.collectAsState()

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Auto-Ordering") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(imageVector = Icons.AutoMirrored.Rounded.ArrowBack, contentDescription = "Back")
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.surface,
                    titleContentColor = MaterialTheme.colorScheme.onSurface
                )
            )
        }
    ) { padding ->
        Box(modifier = Modifier.fillMaxSize().padding(padding)) {
            if (uiState.isLoading && uiState.settings == null) {
                CircularProgressIndicator(modifier = Modifier.align(Alignment.Center))
            } else if (uiState.error \!= null && uiState.settings == null) {
                Text(
                    text = uiState.error\!\!,
                    color = MaterialTheme.colorScheme.error,
                    modifier = Modifier.align(Alignment.Center).padding(16.dp)
                )
            } else if (uiState.settings \!= null) {
                val settings = uiState.settings\!\!
                Column(
                    modifier = Modifier.fillMaxSize().padding(16.dp),
                    verticalArrangement = Arrangement.spacedBy(16.dp)
                ) {
                    Card(
                        modifier = Modifier.fillMaxWidth(),
                        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainer)
                    ) {
                        Row(
                            modifier = Modifier.fillMaxWidth().padding(16.dp),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Column(modifier = Modifier.weight(1f)) {
                                Text("Global Auto-Ordering", style = MaterialTheme.typography.titleMedium)
                                Spacer(modifier = Modifier.height(4.dp))
                                Text(
                                    "When enabled, AI will predict replenishment needs and create draft orders automatically.",
                                    style = MaterialTheme.typography.bodyMedium,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant
                                )
                            }
                            Switch(
                                checked = settings.globalEnabled,
                                onCheckedChange = { isChecked ->
                                    viewModel.toggleGlobalEnabled(isChecked)
                                }
                            )
                        }
                    }

                    if (settings.supplierOverrides.isNotEmpty() || settings.categoryOverrides.isNotEmpty()) {
                        Text("Active Overrides", style = MaterialTheme.typography.titleSmall)
                        // Extra detail logic can be expanded here for overrides.
                        Text(
                            "You have configured specific overrides for certain suppliers or categories.",
                            style = MaterialTheme.typography.bodySmall, 
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    } else if (settings.globalEnabled) {
                        PegasusEmptyState(
                            icon = Icons.Rounded.AutoAwesome,
                            title = "Auto-Ordering Active",
                            description = "The AI agent is now optimizing your inventory restocking schedule.",
                        )
                    }
                }
            }
        }
    }
}
