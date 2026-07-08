package com.pegasusx.warehouse.ui.screens.scanner

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
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
    // private val warehouseApi: WarehouseApi // TODO: Inject API when available
) : ViewModel() {

    private val _state = MutableStateFlow(ScannerState.IDLE)
    val state: StateFlow<ScannerState> = _state.asStateFlow()

    private val _lastScannedBinId = MutableStateFlow<String?>(null)
    val lastScannedBinId: StateFlow<String?> = _lastScannedBinId.asStateFlow()

    private val _errorMessage = MutableStateFlow<String?>(null)
    val errorMessage: StateFlow<String?> = _errorMessage.asStateFlow()

    fun startScanning() {
        _state.value = ScannerState.SCANNING
        _errorMessage.value = null
    }

    fun onBarcodeScanned(barcode: String) {
        if (_state.value != ScannerState.SCANNING) return
        
        _state.value = ScannerState.PROCESSING
        _lastScannedBinId.value = barcode

        viewModelScope.launch {
            try {
                // TODO: Dispatch telemetry event to backend
                // val response = warehouseApi.scanBin(barcode)

                _state.value = ScannerState.SUCCESS
            } catch (e: Exception) {
                _state.value = ScannerState.ERROR
                _errorMessage.value = e.message ?: "Failed to process bin scan"
            }
        }
    }

    fun reset() {
        _state.value = ScannerState.IDLE
        _lastScannedBinId.value = null
        _errorMessage.value = null
    }
}
