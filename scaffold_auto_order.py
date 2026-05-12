import os

pkg_dir = "pegasus/apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/profile"

vm_code = """package com.pegasus.retailer.ui.screens.profile

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasus.retailer.data.api.PegasusApi
import com.pegasus.retailer.data.model.AutoOrderSettings
import com.pegasus.retailer.data.model.UpdateGlobalSettingsRequest
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class AutoOrderUiState(
    val isLoading: Boolean = true,
    val settings: AutoOrderSettings? = null,
    val error: String? = null
)

@HiltViewModel
class AutoOrderViewModel @Inject constructor(
    private val api: PegasusApi
) : ViewModel() {

    private val _uiState = MutableStateFlow(AutoOrderUiState())
    val uiState: StateFlow<AutoOrderUiState> = _uiState.asStateFlow()

    init {
        loadSettings()
    }

    fun loadSettings() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }
            try {
                val settings = api.getAutoOrderSettings()
                _uiState.update { it.copy(isLoading = false, settings = settings) }
            } catch (e: Exception) {
                _uiState.update { it.copy(isLoading = false, error = e.message) }
            }
        }
    }

    fun toggleGlobalEnabled(enabled: Boolean) {
        viewModelScope.launch {
            try {
                val request = UpdateGlobalSettingsRequest(globalAutoOrderEnabled = enabled, useHistory = true)
                api.updateGlobalAutoOrder(request)
                
                // Optimistically update
                _uiState.update { state -> 
                    state.copy(settings = state.settings?.copy(globalEnabled = enabled))
                }
            } catch (e: Exception) {
                _uiState.update { it.copy(error = e.message) }
                loadSettings() // rollback on error
            }
        }
    }
}
"""

screen_code = """package com.pegasus.retailer.ui.screens.profile

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
"""

with open(f"{pkg_dir}/AutoOrderViewModel.kt", "w") as f:
    f.write(vm_code)

with open(f"{pkg_dir}/AutoOrderScreen.kt", "w") as f:
    f.write(screen_code)

print("AutoOrderViewModel and Screen created.")
