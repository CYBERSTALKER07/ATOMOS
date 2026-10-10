package com.pegasusx.warehouse.ui.screens.scanner

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.warehouse.data.model.WarehouseBinLocation
import com.pegasusx.warehouse.data.remote.WarehouseApi
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

enum class ScannerState {
    IDLE,
    SCANNING,
    PROCESSING,
    SUCCESS,
    ERROR
}

@HiltViewModel
class ScannerViewModel @Inject constructor(
    private val warehouseApi: WarehouseApi,
) : ViewModel() {

    private val _state = MutableStateFlow(ScannerState.IDLE)
    val state: StateFlow<ScannerState> = _state.asStateFlow()

    private val _scannedBinIds = MutableStateFlow<List<String>>(emptyList())
    val scannedBinIds: StateFlow<List<String>> = _scannedBinIds.asStateFlow()

    private val _resolvedBins = MutableStateFlow<List<WarehouseBinLocation>>(emptyList())
    val resolvedBins: StateFlow<List<WarehouseBinLocation>> = _resolvedBins.asStateFlow()

    private val _errorMessage = MutableStateFlow<String?>(null)
    val errorMessage: StateFlow<String?> = _errorMessage.asStateFlow()

    fun startScanning() {
        _state.value = ScannerState.SCANNING
        _errorMessage.value = null
    }

    fun onMatrixBarcodesScanned(barcodes: List<String>) {
        if (_state.value != ScannerState.SCANNING) return

        _state.value = ScannerState.PROCESSING
        _scannedBinIds.value = barcodes

        viewModelScope.launch {
            try {
                val response = warehouseApi.listBins()
                if (!response.isSuccessful) {
                    _state.value = ScannerState.ERROR
                    _errorMessage.value = "Bin lookup failed (${response.code()})"
                    return@launch
                }
                
                val apiBins = response.body()?.bins ?: emptyList()
                val foundBins = mutableListOf<WarehouseBinLocation>()
                val missingOrInactive = mutableListOf<String>()
                
                for (code in barcodes) {
                    val cleanCode = code.trim()
                    val bin = apiBins.firstOrNull { it.locationId.equals(cleanCode, ignoreCase = true) }
                    if (bin == null || !bin.isActive) {
                        missingOrInactive.add(cleanCode)
                    } else {
                        foundBins.add(bin)
                    }
                }
                
                if (missingOrInactive.isNotEmpty()) {
                    _state.value = ScannerState.ERROR
                    _errorMessage.value = "Invalid or inactive bins: ${missingOrInactive.joinToString()}"
                } else {
                    _resolvedBins.value = foundBins
                    _state.value = ScannerState.SUCCESS
                }
            } catch (e: Exception) {
                _state.value = ScannerState.ERROR
                _errorMessage.value = e.message ?: "Failed to process bin scan"
            }
        }
    }

    fun reset() {
        _state.value = ScannerState.IDLE
        _scannedBinIds.value = emptyList()
        _resolvedBins.value = emptyList()
        _errorMessage.value = null
    }
}
