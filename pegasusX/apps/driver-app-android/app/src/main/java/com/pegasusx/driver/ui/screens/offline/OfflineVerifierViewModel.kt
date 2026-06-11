package com.pegasusx.driver.ui.screens.offline

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.driver.data.local.RouteManifestDao
import com.pegasusx.driver.data.model.RouteManifestEntity
import com.pegasusx.driver.data.remote.DriverApi
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import javax.inject.Inject

data class OfflineVerifierUiState(
    val isSyncing: Boolean = false,
    val orderCount: Int = 0,
    val syncedAt: String? = null,
    val error: String? = null,
)

@HiltViewModel
class OfflineVerifierViewModel @Inject constructor(
    private val api: DriverApi,
    private val manifestDao: RouteManifestDao,
    private val json: Json,
) : ViewModel() {

    private val _state = MutableStateFlow(OfflineVerifierUiState())
    val state: StateFlow<OfflineVerifierUiState> = _state.asStateFlow()

    fun syncManifest() {
        viewModelScope.launch {
            _state.update { it.copy(isSyncing = true, error = null) }
            try {
                val today = SimpleDateFormat("yyyy-MM-dd", Locale.US).format(Date())
                val manifest = api.getManifest(today)
                manifestDao.upsert(
                    RouteManifestEntity(
                        date = manifest.date.ifBlank { today },
                        driverId = manifest.driverId,
                        expiresAt = manifest.expiresAt,
                        hashesJson = json.encodeToString(manifest.hashes),
                    )
                )
                _state.update {
                    it.copy(
                        isSyncing = false,
                        orderCount = manifest.hashes.size,
                        syncedAt = today,
                    )
                }
            } catch (e: Exception) {
                _state.update {
                    it.copy(isSyncing = false, error = e.message ?: "Manifest sync failed")
                }
            }
        }
    }
}
