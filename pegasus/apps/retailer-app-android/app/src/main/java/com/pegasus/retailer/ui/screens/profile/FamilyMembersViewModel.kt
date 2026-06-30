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
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import retrofit2.HttpException
import javax.inject.Inject

enum class FamilyMembersLoadIssue {
    RESTRICTED,
    OFFLINE,
    ERROR,
}

data class FamilyMember(
    val id: String,
    val name: String,
    val phone: String,
    val spendingLimitUzs: Long = 0,
)

data class FamilyMembersUiState(
    val isLoading: Boolean = true,
    val members: List<FamilyMember> = emptyList(),
    val error: String? = null,
    val loadIssue: FamilyMembersLoadIssue? = null,
) {
    val syncMessage: String?
        get() = when (loadIssue) {
            FamilyMembersLoadIssue.RESTRICTED -> "Family/staff access is restricted for this account"
            FamilyMembersLoadIssue.OFFLINE -> "Offline mode active. Showing latest family/staff data"
            FamilyMembersLoadIssue.ERROR -> "Family/staff sync degraded. Retry is available"
            null -> null
        }
}

@HiltViewModel
class FamilyMembersViewModel @Inject constructor(
    private val api: PegasusApi
) : ViewModel() {

    private val _uiState = MutableStateFlow(FamilyMembersUiState())
    val uiState: StateFlow<FamilyMembersUiState> = _uiState.asStateFlow()

    init {
        loadData()
    }

    fun loadData() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null, loadIssue = null) }
            try {
                val element = api.getFamilyMembers()
                val membersArray = when {
                    element is JsonObject -> element["members"]?.let { json ->
                        runCatching { json.jsonArray }.getOrNull()
                    } ?: emptyList()
                    else -> runCatching { element.jsonArray }.getOrElse { emptyList() }
                }
                val membersList = membersArray.mapNotNull { item ->
                    val obj = runCatching { item.jsonObject }.getOrNull() ?: return@mapNotNull null
                    FamilyMember(
                        id = obj["member_id"]?.jsonPrimitive?.contentOrNull
                            ?: obj["id"]?.jsonPrimitive?.contentOrNull
                            ?: "",
                        name = obj["nickname"]?.jsonPrimitive?.contentOrNull
                            ?: obj["name"]?.jsonPrimitive?.contentOrNull
                            ?: "Unknown",
                        phone = obj["phone"]?.jsonPrimitive?.contentOrNull ?: "",
                        spendingLimitUzs = obj["spending_limit_uzs"]?.jsonPrimitive?.contentOrNull?.toLongOrNull() ?: 0L,
                    )
                }
                _uiState.update { it.copy(isLoading = false, members = membersList, error = null, loadIssue = null) }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        isLoading = false,
                        error = resolveErrorMessage(e, issue, "Could not load family members"),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    fun addMember(name: String, phone: String, spendingLimitUzs: Long = 0) {
        viewModelScope.launch {
            try {
                val map = buildMap {
                    put("nickname", name)
                    if (phone.isNotBlank()) put("phone", phone)
                    if (spendingLimitUzs > 0) put("spending_limit_uzs", spendingLimitUzs)
                }
                api.createFamilyMember(map)
                loadData()
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        error = resolveErrorMessage(e, issue, "Could not add member"),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    fun updateSpendingLimit(memberId: String, spendingLimitUzs: Long) {
        viewModelScope.launch {
            try {
                api.updateFamilyMember(memberId, mapOf("spending_limit_uzs" to spendingLimitUzs))
                loadData()
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        error = resolveErrorMessage(e, issue, "Could not update spending limit"),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    fun deleteMember(id: String) {
        viewModelScope.launch {
            try {
                api.deleteFamilyMember(id)
                loadData()
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        error = resolveErrorMessage(e, issue, "Could not delete member"),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    private fun resolveLoadIssue(error: Exception): FamilyMembersLoadIssue {
        return when {
            error is HttpException && (error.code() == 401 || error.code() == 403) -> FamilyMembersLoadIssue.RESTRICTED
            error is IOException -> FamilyMembersLoadIssue.OFFLINE
            else -> FamilyMembersLoadIssue.ERROR
        }
    }

    private fun resolveErrorMessage(error: Exception, issue: FamilyMembersLoadIssue, fallback: String): String {
        return when (issue) {
            FamilyMembersLoadIssue.RESTRICTED -> "Family/staff access is restricted for this account"
            FamilyMembersLoadIssue.OFFLINE -> "Offline mode active. Reconnect and retry"
            FamilyMembersLoadIssue.ERROR -> error.message ?: fallback
        }
    }
}
