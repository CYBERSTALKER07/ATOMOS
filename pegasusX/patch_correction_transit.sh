sed -i '' '/fun closeEditor()/i\
    fun startTransitForPartialOrder() {\
        val stateVal = _state.value\
        if (!stateVal.isPartial) return\
        viewModelScope.launch {\
            _state.update { it.copy(isSubmitting = true, error = null) }\
            try {\
                val ik = DriverIdempotencyKeys.generate()\
                api.transitionState(orderId, mapOf("state" to "IN_TRANSIT"), ik)\
                _state.update { it.copy(isSubmitting = false) }\
            } catch (e: Exception) {\
                _state.update { it.copy(isSubmitting = false, error = e.message) }\
            }\
        }\
    }\
' /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/driver-app-android/app/src/main/java/com/pegasusx/driver/ui/screens/manifest/CorrectionViewModel.kt
