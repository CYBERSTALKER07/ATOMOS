package com.pegasusx.retailer.ui.screens.settings

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.serialization.json.JsonObject
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.model.RetailerJoinRequestDecisionRequest
import com.pegasusx.retailer.data.model.RetailerJoinRequestDto
import com.pegasusx.retailer.util.RetailerIdempotencyKeys
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject
import com.pegasusx.retailer.R
import com.pegasusx.retailer.data.json.*

data class TeamMemberUi(
    val userId: String,
    val name: String,
    val phone: String,
    val role: String,
    val isOwner: Boolean,
    val isActive: Boolean,
)

@HiltViewModel
class TeamViewModel @Inject constructor(val api: PegasusApi) : ViewModel() {
    private val _joinRequests = MutableStateFlow<List<RetailerJoinRequestDto>>(emptyList())
    val joinRequests: StateFlow<List<RetailerJoinRequestDto>> = _joinRequests.asStateFlow()

    private val _decidingRequestId = MutableStateFlow<String?>(null)
    val decidingRequestId: StateFlow<String?> = _decidingRequestId.asStateFlow()

    private val _joinRequestBanner = MutableStateFlow<String?>(null)
    val joinRequestBanner: StateFlow<String?> = _joinRequestBanner.asStateFlow()

    suspend fun loadJoinRequests(): List<RetailerJoinRequestDto> {
        return try {
            val list = api.listRetailerJoinRequests(status = "PENDING").requests
            _joinRequests.value = list
            list
        } catch (_: Exception) {
            _joinRequests.value
        }
    }

    suspend fun approveJoinRequest(requestId: String, reason: String? = null): Boolean {
        _decidingRequestId.value = requestId
        return try {
            api.approveRetailerJoinRequest(
                requestId = requestId,
                body = RetailerJoinRequestDecisionRequest(reason = reason),
                idempotencyKey = RetailerIdempotencyKeys.joinRequestDecide(requestId, "approve"),
            )
            _joinRequests.value = _joinRequests.value.filterNot { it.requestId == requestId }
            _joinRequestBanner.value = "Join request approved"
            true
        } catch (e: Exception) {
            _joinRequestBanner.value = e.message ?: "Approve failed"
            false
        } finally {
            _decidingRequestId.value = null
        }
    }

    suspend fun rejectJoinRequest(requestId: String, reason: String? = null): Boolean {
        _decidingRequestId.value = requestId
        return try {
            api.rejectRetailerJoinRequest(
                requestId = requestId,
                body = RetailerJoinRequestDecisionRequest(reason = reason),
                idempotencyKey = RetailerIdempotencyKeys.joinRequestDecide(requestId, "reject"),
            )
            _joinRequests.value = _joinRequests.value.filterNot { it.requestId == requestId }
            _joinRequestBanner.value = "Join request rejected"
            true
        } catch (e: Exception) {
            _joinRequestBanner.value = e.message ?: "Reject failed"
            false
        } finally {
            _decidingRequestId.value = null
        }
    }

    fun refreshJoinRequestsInBackground() {
        viewModelScope.launch { loadJoinRequests() }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TeamScreen(
    onNavigateBack: () -> Unit,
    onNavigateToApprovals: () -> Unit = {},
    viewModel: TeamViewModel = hiltViewModel(),
    governanceViewModel: OwnerGovernanceViewModel = hiltViewModel(),
) {
    val scope = rememberCoroutineScope()
    val governanceState by governanceViewModel.uiState.collectAsStateWithLifecycle()
    val joinRequests by viewModel.joinRequests.collectAsStateWithLifecycle()
    val decidingRequestId by viewModel.decidingRequestId.collectAsStateWithLifecycle()
    val joinRequestBanner by viewModel.joinRequestBanner.collectAsStateWithLifecycle()
    var members by remember { mutableStateOf<List<TeamMemberUi>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var banner by remember { mutableStateOf<String?>(null) }
    var name by remember { mutableStateOf("") }
    var phone by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var role by remember { mutableStateOf("CASHIER") }
    var busy by remember { mutableStateOf(false) }

    fun reload() {
        scope.launch {
            loading = true
            error = null
            try {
                val el = viewModel.api.getOrgMembers()
                val arr = el.asJsonObject.getAsJsonArray("items").orEmpty()
                members = arr.map { item ->
                    val o = item.asJsonObject
                    TeamMemberUi(
                        userId = o.get("user_id")?.asString.orEmpty(),
                        name = o.get("name")?.asString.orEmpty(),
                        phone = o.get("phone")?.asString.orEmpty(),
                        role = o.get("retailer_role")?.asString.orEmpty(),
                        isOwner = o.get("is_owner")?.asBoolean == true,
                        isActive = o.get("is_active")?.asBoolean != false,
                    )
                }
            } catch (e: Exception) {
                error = e.message ?: "Load failed"
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) {
        reload()
        viewModel.loadJoinRequests()
        governanceViewModel.loadRoleAndApprovals()
    }

    governanceState.mandateForm?.let { form ->
        SellerMandateEditorDialog(
            form = form,
            isLoading = governanceState.mandateLoading,
            isSaving = governanceState.mandateSaving,
            error = governanceState.mandateError,
            onFormChange = governanceViewModel::updateMandateForm,
            onSave = governanceViewModel::saveMandate,
            onDismiss = governanceViewModel::closeMandateEditor,
        )
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Team") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
            )
        },
    ) { padding ->
        LazyColumn(
            Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(horizontal = 16.dp),
            contentPadding = PaddingValues(vertical = 12.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item {
                Text(
                    "Invite staff with roles. First invite enables the TEAM pack automatically.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            if (governanceState.isOwner) {
                item {
                    OutlinedButton(
                        onClick = onNavigateToApprovals,
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Text(stringResource(R.string.retailer_approval_inbox_title))
                    }
                }
            }
            governanceState.actionBanner?.let { item { Text(it, color = MaterialTheme.colorScheme.primary) } }
            joinRequestBanner?.let { item { Text(it, color = MaterialTheme.colorScheme.primary) } }
            banner?.let { item { Text(it, color = MaterialTheme.colorScheme.primary) } }
            if (joinRequests.isNotEmpty()) {
                item {
                    Text(
                        stringResource(R.string.retailer_desktop_tax_identity_join_requests_title),
                        style = MaterialTheme.typography.titleMedium,
                    )
                }
                items(joinRequests, key = { it.requestId }) { jr ->
                    Card {
                        Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                            Text(
                                jr.requesterName?.takeIf { it.isNotBlank() } ?: jr.requesterPhone,
                                style = MaterialTheme.typography.titleSmall,
                            )
                            Text(
                                "${jr.requesterPhone} • ${jr.roleSought}",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                Button(
                                    enabled = decidingRequestId != jr.requestId,
                                    onClick = {
                                        scope.launch {
                                            if (viewModel.approveJoinRequest(jr.requestId)) {
                                                reload()
                                            }
                                        }
                                    },
                                ) { Text(stringResource(R.string.retailer_desktop_tax_identity_approve_join)) }
                                OutlinedButton(
                                    enabled = decidingRequestId != jr.requestId,
                                    onClick = {
                                        scope.launch { viewModel.rejectJoinRequest(jr.requestId) }
                                    },
                                ) { Text(stringResource(R.string.retailer_desktop_tax_identity_reject_join)) }
                            }
                        }
                    }
                }
            }
            error?.let {
                item {
                    Text(it, color = MaterialTheme.colorScheme.error)
                    Button(onClick = { reload() }) { Text("Retry") }
                }
            }
            item {
                Card {
                    Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text("Invite", style = MaterialTheme.typography.titleMedium)
                        OutlinedTextField(value = name, onValueChange = { name = it }, label = { Text("Name") }, modifier = Modifier.fillMaxWidth())
                        OutlinedTextField(value = phone, onValueChange = { phone = it }, label = { Text("Phone") }, modifier = Modifier.fillMaxWidth())
                        OutlinedTextField(value = password, onValueChange = { password = it }, label = { Text("Password") }, modifier = Modifier.fillMaxWidth())
                        OutlinedTextField(value = role, onValueChange = { role = it.uppercase() }, label = { Text("Role") }, modifier = Modifier.fillMaxWidth())
                        Button(
                            enabled = !busy,
                            onClick = {
                                scope.launch {
                                    busy = true
                                    banner = null
                                    try {
                                        viewModel.api.createOrgMember(
                                            body = mapOf(
                                                "name" to name,
                                                "phone" to phone,
                                                "password" to password,
                                                "retailer_role" to role,
                                            ),
                                            idempotencyKey = "team-${System.currentTimeMillis()}",
                                        )
                                        banner = "Member created"
                                        name = ""; phone = ""; password = ""
                                        reload()
                                    } catch (e: Exception) {
                                        banner = e.message ?: "Create failed"
                                    } finally {
                                        busy = false
                                    }
                                }
                            },
                        ) { Text(if (busy) "…" else "Create member") }
                    }
                }
            }
            items(members, key = { it.userId }) { m ->
                Card {
                    Column(Modifier.padding(14.dp)) {
                        Text(m.name, style = MaterialTheme.typography.titleSmall)
                        Text(stringResource(R.string.mobile_retailer_ui_phone_role, m.phone, m.role), style = MaterialTheme.typography.bodySmall)
                        if (m.isOwner) Text("Owner", style = MaterialTheme.typography.labelSmall)
                        if (!m.isActive) Text("Inactive", color = MaterialTheme.colorScheme.error)
                        if (!m.isOwner && m.isActive) {
                            Spacer(Modifier.height(8.dp))
                            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                if (governanceState.isOwner) {
                                    Button(onClick = { governanceViewModel.openMandateEditor(m.userId, m.name) }) {
                                        Text(stringResource(R.string.retailer_mandate_title))
                                    }
                                }
                                OutlinedButton(onClick = {
                                    scope.launch {
                                        try {
                                            viewModel.api.deactivateOrgMember(
                                                userId = m.userId,
                                                idempotencyKey = "team-deact-${m.userId}-${System.currentTimeMillis()}",
                                            )
                                            banner = "Deactivated"
                                            reload()
                                        } catch (e: Exception) {
                                            banner = e.message
                                        }
                                    }
                                }) { Text("Deactivate") }
                            }
                        }
                    }
                }
            }
        }
    }
}
