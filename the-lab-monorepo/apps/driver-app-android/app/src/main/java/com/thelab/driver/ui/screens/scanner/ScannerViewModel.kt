package com.thelab.driver.ui.screens.scanner

import android.app.Application
import android.os.Build
import android.os.VibrationEffect
import android.os.Vibrator
import android.os.VibratorManager
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.thelab.driver.data.model.ValidateQRRequest
import com.thelab.driver.data.model.ValidateQRResponse
import com.thelab.driver.data.remote.DriverApi
import dagger.hilt.android.lifecycle.HiltViewModel
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.TimeoutCancellationException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import javax.inject.Inject

data class ScannerUiState(
    val isScanning: Boolean = true,
    val scannedToken: String? = null,
    val isSubmitting: Boolean = false,
    val validated: ValidateQRResponse? = null,
    val error: String? = null
)

@HiltViewModel
class ScannerViewModel @Inject constructor(
    private val app: Application,
    private val api: DriverApi
) : ViewModel() {

    private val _state = MutableStateFlow(ScannerUiState())
    val state: StateFlow<ScannerUiState> = _state.asStateFlow()

    fun onQrDetected(rawValue: String) {
        if (!_state.value.isScanning) return
        _state.value = _state.value.copy(isScanning = false, scannedToken = rawValue)
        vibrateOnDetect()
        validateQR(rawValue)
    }

    private fun vibrateOnDetect() {
        val vibrator = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
            val mgr = app.getSystemService(VibratorManager::class.java)
            mgr?.defaultVibrator
        } else {
            @Suppress("DEPRECATION")
            app.getSystemService(Vibrator::class.java)
        }
        vibrator?.vibrate(VibrationEffect.createOneShot(150, VibrationEffect.DEFAULT_AMPLITUDE))
    }

    private fun validateQR(qrToken: String) {
        viewModelScope.launch {
            _state.value = _state.value.copy(isSubmitting = true, error = null)
            try {
                val parts = qrToken.split(":")
                val orderId = if (parts.size >= 2) parts[0] else qrToken

                val response = withTimeout(30_000L) {
                    api.validateQR(
                        ValidateQRRequest(orderId = orderId, scannedToken = qrToken)
                    )
                }
                _state.value = _state.value.copy(
                    isSubmitting = false,
                    validated = response
                )
            } catch (e: TimeoutCancellationException) {
                _state.value = _state.value.copy(
                    isSubmitting = false,
                    error = "QR validation timed out. Please retry."
                )
            } catch (e: Exception) {
                _state.value = _state.value.copy(
                    isSubmitting = false,
                    error = e.message ?: "QR validation failed"
                )
            }
        }
    }

    fun resetScanner() {
        _state.value = ScannerUiState()
    }
}
