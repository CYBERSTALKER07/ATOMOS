package com.pegasus.retailer.ui.screens.profile

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasus.retailer.data.api.PegasusApi
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import javax.inject.Inject

data class SavedCard(
    val id: String,
    val pan: String,
    val isDefault: Boolean,
    val type: String
)

data class SavedCardsUiState(
    val isLoading: Boolean = true,
    val cards: List<SavedCard> = emptyList(),
    val error: String? = null,
    val isAddingCard: Boolean = false,
    val initiateSession: String? = null,
    val otpPhone: String? = null,
    val addError: String? = null
)

@HiltViewModel
class SavedCardsViewModel @Inject constructor(
    private val api: PegasusApi
) : ViewModel() {

    private val _uiState = MutableStateFlow(SavedCardsUiState())
    val uiState: StateFlow<SavedCardsUiState> = _uiState.asStateFlow()

    init {
        loadCards()
    }

    fun loadCards() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }
            try {
                val element = api.getCards()
                val cardsArray = when {
                    element is JsonObject -> element["cards"]?.let { json ->
                        runCatching { json.jsonArray }.getOrNull()
                    } ?: emptyList()
                    else -> runCatching { element.jsonArray }.getOrElse { emptyList() }
                }
                val cardsList = cardsArray.mapNotNull { item ->
                    val obj = runCatching { item.jsonObject }.getOrNull() ?: return@mapNotNull null
                    SavedCard(
                        id = obj["id"]?.jsonPrimitive?.contentOrNull
                            ?: obj["token_id"]?.jsonPrimitive?.contentOrNull
                            ?: "",
                        pan = obj["pan"]?.jsonPrimitive?.contentOrNull
                            ?: obj["pan_mask"]?.jsonPrimitive?.contentOrNull
                            ?: "",
                        isDefault = obj["is_default"]?.jsonPrimitive?.booleanOrNull ?: false,
                        type = obj["type"]?.jsonPrimitive?.contentOrNull ?: "UNKNOWN",
                    )
                }
                _uiState.update { it.copy(isLoading = false, cards = cardsList) }
            } catch (e: Exception) {
                _uiState.update { it.copy(isLoading = false, error = e.message) }
            }
        }
    }

    fun initiateCard(cardNumber: String, expire: String) {
        viewModelScope.launch {
            _uiState.update { it.copy(addError = null) }
            try {
                val res = api.initiateCard(mapOf(
                    "card_number" to cardNumber,
                    "expire" to expire
                ))
                if (res is JsonObject) {
                    val session = res["session"]?.jsonPrimitive?.contentOrNull
                    val phone = res["phone"]?.jsonPrimitive?.contentOrNull
                    _uiState.update { it.copy(initiateSession = session, otpPhone = phone, isAddingCard = true) }
                }
            } catch (e: Exception) {
                _uiState.update { it.copy(addError = e.message) }
            }
        }
    }

    fun confirmCard(otp: String) {
        val session = _uiState.value.initiateSession ?: return
        viewModelScope.launch {
            _uiState.update { it.copy(addError = null) }
            try {
                api.confirmCard(mapOf(
                    "session" to session,
                    "otp" to otp
                ))
                _uiState.update { it.copy(isAddingCard = false, initiateSession = null, otpPhone = null) }
                loadCards()
            } catch (e: Exception) {
                _uiState.update { it.copy(addError = e.message) }
            }
        }
    }

    fun cancelAdd() {
        _uiState.update { it.copy(isAddingCard = false, initiateSession = null, otpPhone = null, addError = null) }
    }

    fun setDefault(cardId: String) {
        viewModelScope.launch {
            try {
                api.setDefaultCard(mapOf("card_id" to cardId))
                loadCards()
            } catch (e: Exception) {
                _uiState.update { it.copy(error = e.message) }
            }
        }
    }

    fun deleteCard(cardId: String) {
        viewModelScope.launch {
            try {
                api.deactivateCard(mapOf("card_id" to cardId))
                loadCards()
            } catch (e: Exception) {
                _uiState.update { it.copy(error = e.message) }
            }
        }
    }
}
