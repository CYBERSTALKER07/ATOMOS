package com.pegasus.retailer.ui.screens.profile

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.google.gson.JsonElement
import com.pegasus.retailer.data.api.PegasusApi
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
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
                val cardsList = mutableListOf<SavedCard>()
                if (element.isJsonObject && element.asJsonObject.has("cards")) {
                    element.asJsonObject.getAsJsonArray("cards").forEach {
                        val obj = it.asJsonObject
                        cardsList.add(
                            SavedCard(
                                id = obj.get("id")?.asString ?: "",
                                pan = obj.get("pan")?.asString ?: "",
                                isDefault = obj.get("is_default")?.asBoolean ?: false,
                                type = obj.get("type")?.asString ?: "UNKNOWN"
                            )
                        )
                    }
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
                if (res.isJsonObject) {
                    val session = res.asJsonObject.get("session")?.asString
                    val phone = res.asJsonObject.get("phone")?.asString
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
