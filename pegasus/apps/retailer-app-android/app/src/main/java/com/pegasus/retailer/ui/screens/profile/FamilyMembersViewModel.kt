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
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import javax.inject.Inject

data class FamilyMember(
    val id: String,
    val name: String,
    val phone: String
)

data class FamilyMembersUiState(
    val isLoading: Boolean = true,
    val members: List<FamilyMember> = emptyList(),
    val error: String? = null
)

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
            _uiState.update { it.copy(isLoading = true, error = null) }
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
                    )
                }
                _uiState.update { it.copy(isLoading = false, members = membersList) }
            } catch (e: Exception) {
                _uiState.update { it.copy(isLoading = false, error = e.message) }
            }
        }
    }

    fun addMember(name: String, phone: String) {
        viewModelScope.launch {
            try {
                val map = mapOf("nickname" to name, "phone" to phone)
                api.createFamilyMember(map)
                loadData()
            } catch (e: Exception) {
                _uiState.update { it.copy(error = e.message) }
            }
        }
    }

    fun deleteMember(id: String) {
        viewModelScope.launch {
            try {
                api.deleteFamilyMember(id)
                loadData()
            } catch (e: Exception) {
                _uiState.update { it.copy(error = e.message) }
            }
        }
    }
}
