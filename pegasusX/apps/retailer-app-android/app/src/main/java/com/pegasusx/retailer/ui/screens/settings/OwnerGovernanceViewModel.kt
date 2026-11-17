package com.pegasusx.retailer.ui.screens.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.model.ApprovalDecisionRequest
import com.pegasusx.retailer.data.model.RetailerApprovalItem
import com.pegasusx.retailer.data.model.SellerDailyUsageDTO
import com.pegasusx.retailer.data.model.SellerMandateUpdateRequest
import com.pegasusx.retailer.util.RetailerW2ContractHelper
import dagger.hilt.android.lifecycle.HiltViewModel
import javax.inject.Inject
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

data class SellerMandateFormState(
    val userId: String = "",
    val memberName: String = "",
    val memberRole: String = "",
    val canReceiveGoods: Boolean = true,
    val canPayCash: Boolean = true,
    val canUseCredit: Boolean = true,
    val dailyCreditCapSom: String = "0",
    val dailyCashCapSom: String = "0",
    val maxSingleOrderSom: String = "0",
    val dailyUsage: SellerDailyUsageDTO = SellerDailyUsageDTO(),
)

data class OwnerGovernanceUiState(
    val isLoadingRole: Boolean = true,
    val isOwner: Boolean = false,
    val retailerRole: String = "",
    val statusFilter: String = "PENDING_OWNER",
    val approvals: List<RetailerApprovalItem> = emptyList(),
    val approvalsLoading: Boolean = false,
    val approvalsError: String? = null,
    val decidingRequestId: String? = null,
    val actionBanner: String? = null,
    val mandateForm: SellerMandateFormState? = null,
    val mandateLoading: Boolean = false,
    val mandateSaving: Boolean = false,
    val mandateError: String? = null,
)

@HiltViewModel
class OwnerGovernanceViewModel @Inject constructor(
    private val api: PegasusApi,
) : ViewModel() {

    private val _uiState = MutableStateFlow(OwnerGovernanceUiState())
    val uiState: StateFlow<OwnerGovernanceUiState> = _uiState.asStateFlow()

    fun loadRoleAndApprovals() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoadingRole = true, approvalsError = null) }
            try {
                val me = api.getRetailerMe().jsonObject
                val role = me["retailer_role"]?.jsonPrimitive?.contentOrNull?.trim()?.uppercase().orEmpty()
                val ownerFlag = me["is_owner"]?.jsonPrimitive?.booleanOrNull == true || role == "OWNER"
                _uiState.update {
                    it.copy(
                        isLoadingRole = false,
                        isOwner = ownerFlag,
                        retailerRole = role,
                    )
                }
                if (ownerFlag) {
                    refreshApprovalsInternal(_uiState.value.statusFilter)
                }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(
                        isLoadingRole = false,
                        isOwner = false,
                        approvalsError = e.message ?: "Failed to verify role",
                    )
                }
            }
        }
    }

    fun selectStatusFilter(status: String) {
        _uiState.update { it.copy(statusFilter = status, actionBanner = null) }
        if (_uiState.value.isOwner) {
            viewModelScope.launch { refreshApprovalsInternal(status) }
        }
    }

    fun refreshApprovals() {
        if (!_uiState.value.isOwner) {
            loadRoleAndApprovals()
            return
        }
        viewModelScope.launch {
            refreshApprovalsInternal(_uiState.value.statusFilter)
        }
    }

    private suspend fun refreshApprovalsInternal(status: String) {
        _uiState.update { it.copy(approvalsLoading = true, approvalsError = null) }
        try {
            val response = api.getRetailerApprovals(status = status, limit = 20, cursor = null)
            _uiState.update {
                it.copy(
                    approvalsLoading = false,
                    approvals = response.items,
                )
            }
        } catch (e: CancellationException) {
            throw e
        } catch (e: Exception) {
            _uiState.update {
                it.copy(
                    approvalsLoading = false,
                    approvalsError = e.message ?: "Failed to load approval requests",
                )
            }
        }
    }

    fun approveRequest(requestId: String, reason: String? = null) {
        decideRequest(requestId = requestId, approve = true, reason = reason)
    }

    fun rejectRequest(requestId: String, reason: String? = null) {
        decideRequest(requestId = requestId, approve = false, reason = reason)
    }

    private fun decideRequest(requestId: String, approve: Boolean, reason: String?) {
        if (_uiState.value.decidingRequestId != null || !_uiState.value.isOwner) return
        viewModelScope.launch {
            _uiState.update { it.copy(decidingRequestId = requestId, approvalsError = null, actionBanner = null) }
            val body = ApprovalDecisionRequest(reason = reason?.trim()?.ifBlank { null })
            try {
                if (approve) {
                    api.approveRetailerRequest(requestId, body)
                } else {
                    api.rejectRetailerRequest(requestId, body)
                }
                val message = if (approve) "Order approved and dispatched." else "Approval request rejected."
                _uiState.update {
                    it.copy(
                        decidingRequestId = null,
                        actionBanner = message,
                    )
                }
                refreshApprovalsInternal(_uiState.value.statusFilter)
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(
                        decidingRequestId = null,
                        approvalsError = e.message ?: "Failed to submit decision",
                    )
                }
            }
        }
    }

    fun openMandateEditor(userId: String, memberName: String) {
        if (!_uiState.value.isOwner) return
        viewModelScope.launch {
            _uiState.update {
                it.copy(
                    mandateLoading = true,
                    mandateError = null,
                    actionBanner = null,
                    mandateForm = SellerMandateFormState(userId = userId, memberName = memberName),
                )
            }
            try {
                val res = api.getSellerMandate(userId)
                _uiState.update {
                    it.copy(
                        mandateLoading = false,
                        mandateForm = SellerMandateFormState(
                            userId = userId,
                            memberName = memberName,
                            memberRole = res.retailerRole,
                            canReceiveGoods = res.mandate.canReceiveGoods,
                            canPayCash = res.mandate.canPayCash,
                            canUseCredit = res.mandate.canUseCredit,
                            dailyCreditCapSom = RetailerW2ContractHelper.minorToSomInput(res.mandate.dailyCreditCapMinor),
                            dailyCashCapSom = RetailerW2ContractHelper.minorToSomInput(res.mandate.dailyCashCapMinor),
                            maxSingleOrderSom = RetailerW2ContractHelper.minorToSomInput(res.mandate.maxSingleOrderMinor),
                            dailyUsage = res.dailyUsage,
                        ),
                    )
                }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(
                        mandateLoading = false,
                        mandateError = e.message ?: "Failed to load seller mandate",
                    )
                }
            }
        }
    }

    fun updateMandateForm(transform: (SellerMandateFormState) -> SellerMandateFormState) {
        _uiState.update { state ->
            state.mandateForm?.let { form ->
                state.copy(mandateForm = transform(form), mandateError = null)
            } ?: state
        }
    }

    fun closeMandateEditor() {
        _uiState.update { it.copy(mandateForm = null, mandateError = null, mandateLoading = false, mandateSaving = false) }
    }

    fun saveMandate() {
        val form = _uiState.value.mandateForm ?: return
        if (!_uiState.value.isOwner || _uiState.value.mandateSaving) return

        val creditCapMinor = RetailerW2ContractHelper.somInputToMinor(form.dailyCreditCapSom)
        val cashCapMinor = RetailerW2ContractHelper.somInputToMinor(form.dailyCashCapSom)
        val singleCapMinor = RetailerW2ContractHelper.somInputToMinor(form.maxSingleOrderSom)
        if (creditCapMinor == null || cashCapMinor == null || singleCapMinor == null) {
            _uiState.update { it.copy(mandateError = "Enter valid whole-number so'm amounts (0 or greater).") }
            return
        }

        viewModelScope.launch {
            _uiState.update { it.copy(mandateSaving = true, mandateError = null) }
            try {
                api.updateSellerMandate(
                    userId = form.userId,
                    body = SellerMandateUpdateRequest(
                        canReceiveGoods = form.canReceiveGoods,
                        canPayCash = form.canPayCash,
                        canUseCredit = form.canUseCredit,
                        dailyCreditCapMinor = creditCapMinor,
                        dailyCashCapMinor = cashCapMinor,
                        maxSingleOrderMinor = singleCapMinor,
                    ),
                )
                _uiState.update {
                    it.copy(
                        mandateSaving = false,
                        mandateForm = null,
                        actionBanner = "Saved seller mandate for ${form.memberName}",
                    )
                }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(
                        mandateSaving = false,
                        mandateError = e.message ?: "Failed to save mandate",
                    )
                }
            }
        }
    }
}
