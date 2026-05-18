package com.pegasus.retailer.ui.screens.profile

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasus.retailer.data.api.PegasusApi
import dagger.hilt.android.lifecycle.HiltViewModel
import java.io.IOException
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
import retrofit2.HttpException
import javax.inject.Inject

enum class SavedCardsLoadIssue {
    RESTRICTED,
    OFFLINE,
    ERROR,
}

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
    val addError: String? = null,
    val loadIssue: SavedCardsLoadIssue? = null,
) {
    val syncMessage: String?
        get() = when (loadIssue) {
            SavedCardsLoadIssue.RESTRICTED -> "Saved cards access is restricted for this account"
            SavedCardsLoadIssue.OFFLINE -> "Offline mode active. Showing latest saved cards"
            SavedCardsLoadIssue.ERROR -> "Saved cards sync degraded. Retry is available"
            null -> null
        }
}

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
            _uiState.update { it.copy(isLoading = true, error = null, loadIssue = null) }
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
                _uiState.update { it.copy(isLoading = false, cards = cardsList, error = null, loadIssue = null) }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        isLoading = false,
                        error = resolveErrorMessage(e, issue, "Could not load saved cards"),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    fun initiateCard() {
        viewModelScope.launch {
            _uiState.update { it.copy(addError = null, loadIssue = null) }
            try {
                val res = api.initiateCard(emptyMap())
                if (res is JsonObject) {
                    val session = res["card_token"]?.jsonPrimitive?.contentOrNull
                    _uiState.update { it.copy(initiateSession = session, otpPhone = null, isAddingCard = true) }
                }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        addError = resolveErrorMessage(e, issue, "Could not initiate card verification"),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    fun confirmCard(otp: String) {
        val session = _uiState.value.initiateSession ?: return
        viewModelScope.launch {
            _uiState.update { it.copy(addError = null, loadIssue = null) }
            try {
                api.confirmCard(mapOf(
                    "card_token" to session,
                    "otp_code" to otp
                ))
                _uiState.update { it.copy(isAddingCard = false, initiateSession = null, otpPhone = null) }
                loadCards()
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        addError = resolveErrorMessage(e, issue, "Could not confirm card"),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    fun cancelAdd() {
        _uiState.update { it.copy(isAddingCard = false, initiateSession = null, otpPhone = null, addError = null, loadIssue = null) }
    }

    fun setDefault(cardId: String) {
        viewModelScope.launch {
            try {
                api.setDefaultCard(mapOf("card_id" to cardId))
                loadCards()
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        error = resolveErrorMessage(e, issue, "Could not set default card"),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    fun deleteCard(cardId: String) {
        viewModelScope.launch {
            try {
                api.deactivateCard(mapOf("card_id" to cardId))
                loadCards()
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        error = resolveErrorMessage(e, issue, "Could not delete card"),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    private fun resolveLoadIssue(error: Exception): SavedCardsLoadIssue {
        return when {
            error is HttpException && (error.code() == 401 || error.code() == 403) -> SavedCardsLoadIssue.RESTRICTED
            error is IOException -> SavedCardsLoadIssue.OFFLINE
            else -> SavedCardsLoadIssue.ERROR
        }
    }

    private fun resolveErrorMessage(error: Exception, issue: SavedCardsLoadIssue, fallback: String): String {
        return when (issue) {
            SavedCardsLoadIssue.RESTRICTED -> "Saved cards access is restricted for this account"
            SavedCardsLoadIssue.OFFLINE -> "Offline mode active. Reconnect and retry"
            SavedCardsLoadIssue.ERROR -> error.message ?: fallback
        }
    }
}
