package com.pegasusx.driver.ui.screens.offline

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.driver.data.local.RouteManifestDao
import com.pegasusx.driver.data.model.RouteManifestEntity
import com.pegasusx.driver.data.remote.DriverApi
import com.pegasusx.driver.util.sha256Hex
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import javax.inject.Inject

@Serializable
private data class QrPayload(
    @kotlinx.serialization.SerialName("order_id") val orderId: String,
    val token: String,
)

sealed class VerificationState {
    data object Idle : VerificationState()
    data object Syncing : VerificationState()
    data class Ready(val manifest: CachedManifest) : VerificationState()
    data object Scanning : VerificationState()
    data class Verified(val orderId: String) : VerificationState()
    data class Fraud(val reason: String) : VerificationState()
    data class Error(val reason: String) : VerificationState()
}

data class CachedManifest(
    val driverId: String,
    val date: String,
    val expiresAt: Long,
    val hashes: Map<String, String>,
) {
    val isValid: Boolean get() = System.currentTimeMillis() / 1000 < expiresAt
}

data class OfflineVerifierUiState(
    val verificationState: VerificationState = VerificationState.Idle,
    val isSyncing: Boolean = false,
    val orderCount: Int = 0,
    val syncedAt: String? = null,
    val error: String? = null,
) {
    val statusLabel: String
        get() = when (verificationState) {
            VerificationState.Idle -> "IDLE"
            VerificationState.Syncing -> "SYNCING"
            is VerificationState.Ready -> "READY"
            VerificationState.Scanning -> "SCANNING"
            is VerificationState.Verified -> "VERIFIED"
            is VerificationState.Fraud -> "FRAUD"
            is VerificationState.Error -> "ERROR"
        }
}

@HiltViewModel
class OfflineVerifierViewModel @Inject constructor(
    private val api: DriverApi,
    private val manifestDao: RouteManifestDao,
    private val json: Json,
) : ViewModel() {

    private var cachedManifest: CachedManifest? = null
    private var scanLocked = false

    private val _state = MutableStateFlow(OfflineVerifierUiState())
    val state: StateFlow<OfflineVerifierUiState> = _state.asStateFlow()

    init {
        viewModelScope.launch { restoreCachedManifest() }
    }

    private suspend fun restoreCachedManifest() {
        val today = SimpleDateFormat("yyyy-MM-dd", Locale.US).format(Date())
        val entity = manifestDao.getForDate(today) ?: return
        val hashes = runCatching {
            json.decodeFromString<Map<String, String>>(entity.hashesJson)
        }.getOrDefault(emptyMap())
        val manifest = CachedManifest(
            driverId = entity.driverId,
            date = entity.date,
            expiresAt = entity.expiresAt,
            hashes = hashes,
        )
        cachedManifest = manifest
        _state.update {
            it.copy(
                verificationState = VerificationState.Ready(manifest),
                orderCount = hashes.size,
                syncedAt = entity.date,
            )
        }
    }

    fun syncManifest() {
        viewModelScope.launch {
            _state.update {
                it.copy(
                    isSyncing = true,
                    error = null,
                    verificationState = VerificationState.Syncing,
                )
            }
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
                val cached = CachedManifest(
                    driverId = manifest.driverId,
                    date = manifest.date.ifBlank { today },
                    expiresAt = manifest.expiresAt,
                    hashes = manifest.hashes,
                )
                cachedManifest = cached
                _state.update {
                    it.copy(
                        isSyncing = false,
                        orderCount = manifest.hashes.size,
                        syncedAt = today,
                        verificationState = VerificationState.Ready(cached),
                    )
                }
            } catch (e: Exception) {
                _state.update {
                    it.copy(
                        isSyncing = false,
                        error = e.message ?: "Manifest sync failed",
                        verificationState = VerificationState.Error(
                            e.message ?: "Manifest sync failed",
                        ),
                    )
                }
            }
        }
    }

    fun activateScanner() {
        val manifest = cachedManifest
        if (manifest == null) {
            _state.update {
                it.copy(verificationState = VerificationState.Error("No manifest loaded."))
            }
            return
        }
        if (!manifest.isValid) {
            _state.update {
                it.copy(verificationState = VerificationState.Error("Manifest expired. Re-sync required."))
            }
            return
        }
        _state.update { it.copy(verificationState = VerificationState.Scanning) }
    }

    fun cancelScanner() {
        val manifest = cachedManifest
        _state.update {
            it.copy(
                verificationState = if (manifest != null) {
                    VerificationState.Ready(manifest)
                } else {
                    VerificationState.Idle
                },
            )
        }
    }

    fun handleBarcodeScan(rawValue: String) {
        if (scanLocked) return
        scanLocked = true
        viewModelScope.launch {
            kotlinx.coroutines.delay(2_000)
            scanLocked = false
        }

        val manifest = cachedManifest
        if (manifest == null) {
            _state.update {
                it.copy(verificationState = VerificationState.Error("No manifest loaded."))
            }
            return
        }
        if (!manifest.isValid) {
            _state.update {
                it.copy(verificationState = VerificationState.Error("Manifest expired. Re-sync required."))
            }
            return
        }

        val payload = runCatching { json.decodeFromString<QrPayload>(rawValue) }.getOrNull()
        if (payload == null) {
            _state.update {
                it.copy(verificationState = VerificationState.Error("Invalid QR code format."))
            }
            return
        }

        val expectedHash = manifest.hashes[payload.orderId]
        if (expectedHash == null) {
            _state.update {
                it.copy(verificationState = VerificationState.Fraud("ORDER NOT FOUND IN ROUTE MANIFEST"))
            }
            return
        }

        val computedHash = sha256Hex(payload.token)
        if (computedHash == expectedHash) {
            _state.update {
                it.copy(verificationState = VerificationState.Verified(payload.orderId))
            }
        } else {
            _state.update {
                it.copy(verificationState = VerificationState.Fraud("CRYPTOGRAPHIC MISMATCH"))
            }
        }
    }

    fun resetTerminal() {
        cachedManifest = null
        _state.update {
            it.copy(
                verificationState = VerificationState.Idle,
                orderCount = 0,
                syncedAt = null,
            )
        }
    }

    fun nextDelivery() {
        val manifest = cachedManifest
        _state.update {
            it.copy(
                verificationState = if (manifest != null) {
                    VerificationState.Ready(manifest)
                } else {
                    VerificationState.Idle
                },
            )
        }
    }
}
