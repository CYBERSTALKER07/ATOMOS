package com.pegasusx.retailer.ui.screens.profile

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.model.FamilyMigrateResult
import dagger.hilt.android.lifecycle.HiltViewModel
import java.io.IOException
import java.util.UUID
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import retrofit2.HttpException
import javax.inject.Inject

enum class FamilyMembersLoadIssue {
    RESTRICTED,
    OFFLINE,
    ERROR,
    GONE,
}

data class FamilyMember(
    val id: String,
    val name: String,
    val phone: String,
)

data class FamilyMembersUiState(
    val isLoading: Boolean = true,
    val members: List<FamilyMember> = emptyList(),
    val error: String? = null,
    val loadIssue: FamilyMembersLoadIssue? = null,
    val familyWrites: String = "open",
    val migrating: Boolean = false,
    val migrateResult: FamilyMigrateResult? = null,
    val banner: String? = null,
) {
    val familyGone: Boolean get() = familyWrites == "gone" || loadIssue == FamilyMembersLoadIssue.GONE

    val syncMessage: String?
        get() = when (loadIssue) {
            FamilyMembersLoadIssue.RESTRICTED -> "Family/staff access is restricted for this account"
            FamilyMembersLoadIssue.OFFLINE -> "Offline mode active. Showing latest family/staff data"
            FamilyMembersLoadIssue.ERROR -> "Family/staff sync degraded. Retry is available"
            FamilyMembersLoadIssue.GONE -> "Family writes closed — use Team staff"
            null -> banner
        }
}

@HiltViewModel
class FamilyMembersViewModel @Inject constructor(
    private val api: PegasusApi,
) : ViewModel() {

    private val json = Json { ignoreUnknownKeys = true }

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
                val obj = element as? JsonObject
                val familyWrites = obj?.get("family_writes")?.jsonPrimitive?.contentOrNull ?: "open"
                val membersArray = when {
                    obj != null -> obj["members"]?.let { runCatching { it.jsonArray }.getOrNull() } ?: emptyList()
                    else -> runCatching { element.jsonArray }.getOrElse { emptyList() }
                }
                val membersList = membersArray.mapNotNull { item ->
                    val m = runCatching { item.jsonObject }.getOrNull() ?: return@mapNotNull null
                    FamilyMember(
                        id = m["member_id"]?.jsonPrimitive?.contentOrNull
                            ?: m["id"]?.jsonPrimitive?.contentOrNull
                            ?: "",
                        name = m["name"]?.jsonPrimitive?.contentOrNull
                            ?: m["nickname"]?.jsonPrimitive?.contentOrNull
                            ?: "Unknown",
                        phone = m["phone"]?.jsonPrimitive?.contentOrNull ?: "",
                    )
                }
                _uiState.update {
                    it.copy(
                        isLoading = false,
                        members = membersList,
                        error = null,
                        loadIssue = if (familyWrites == "gone") FamilyMembersLoadIssue.GONE else null,
                        familyWrites = familyWrites,
                    )
                }
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

    fun addMember(name: String, phone: String) {
        viewModelScope.launch {
            try {
                // Backend accepts name (preferred) or nickname.
                val map = mapOf("name" to name, "phone" to phone)
                api.createFamilyMember(map, idempotencyKey = "fam-add-${UUID.randomUUID()}")
                _uiState.update { it.copy(banner = null, error = null) }
                loadData()
            } catch (e: Exception) {
                if (e is HttpException && e.code() == 410) {
                    _uiState.update {
                        it.copy(
                            familyWrites = "gone",
                            loadIssue = FamilyMembersLoadIssue.GONE,
                            error = "Family writes closed. Migrate remaining members or use Team.",
                        )
                    }
                    return@launch
                }
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

    fun migrateToTeam() {
        viewModelScope.launch {
            _uiState.update { it.copy(migrating = true, error = null, banner = null, migrateResult = null) }
            try {
                val element = api.migrateFamilyToTeam(
                    body = mapOf("retailer_role" to "RECEIVER"),
                    idempotencyKey = "fam-migrate-${System.currentTimeMillis()}",
                )
                val result = json.decodeFromJsonElement(FamilyMigrateResult.serializer(), element)
                val migrated = result.migrated.size
                val skipped = result.skipped.size
                _uiState.update {
                    it.copy(
                        migrating = false,
                        migrateResult = result,
                        familyWrites = result.familyWrites.ifBlank { "gone" },
                        loadIssue = FamilyMembersLoadIssue.GONE,
                        banner = "Migrated $migrated · skipped $skipped. Copy temp passwords now.",
                    )
                }
                loadData()
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        migrating = false,
                        error = resolveErrorMessage(e, issue, "Migration failed"),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    fun clearBanner() {
        _uiState.update { it.copy(banner = null) }
    }

    private fun resolveLoadIssue(error: Exception): FamilyMembersLoadIssue {
        return when {
            error is HttpException && error.code() == 410 -> FamilyMembersLoadIssue.GONE
            error is HttpException && (error.code() == 401 || error.code() == 403) -> FamilyMembersLoadIssue.RESTRICTED
            error is IOException -> FamilyMembersLoadIssue.OFFLINE
            else -> FamilyMembersLoadIssue.ERROR
        }
    }

    private fun resolveErrorMessage(error: Exception, issue: FamilyMembersLoadIssue, fallback: String): String {
        return when (issue) {
            FamilyMembersLoadIssue.RESTRICTED -> "Family/staff access is restricted for this account"
            FamilyMembersLoadIssue.OFFLINE -> "Offline mode active. Reconnect and retry"
            FamilyMembersLoadIssue.GONE -> "Family writes closed. Use Team staff."
            FamilyMembersLoadIssue.ERROR -> error.message ?: fallback
        }
    }
}
