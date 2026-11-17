package com.pegasusx.retailer.ui.screens.credit

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.outlined.Refresh
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.ViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.R
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.model.ArInvoiceDto
import com.pegasusx.retailer.data.model.ArPromiseDto
import com.pegasusx.retailer.data.model.ArStatementResponse
import com.pegasusx.retailer.data.model.CreateArPromiseRequest
import com.pegasusx.retailer.data.model.CreditRelationshipDto
import com.pegasusx.retailer.data.model.PayArFifoRequest
import com.pegasusx.retailer.data.model.PayArInvoiceRequest
import com.pegasusx.retailer.util.OrderLineContractHelper
import com.pegasusx.retailer.util.QuarterlyPromiseLimitError
import com.pegasusx.retailer.util.RetailerW2ContractHelper
import com.pegasusx.retailer.util.RetailerW3ContractHelper
import dagger.hilt.android.lifecycle.HiltViewModel
import javax.inject.Inject
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

data class CreditUiState(
    val isLoading: Boolean = false,
    val isRefreshingDetail: Boolean = false,
    val relationships: List<CreditRelationshipDto> = emptyList(),
    val selectedSupplierId: String = "",
    val statement: ArStatementResponse? = null,
    val invoices: List<ArInvoiceDto> = emptyList(),
    val promises: List<ArPromiseDto> = emptyList(),
    val currentQuarterKey: String = "",
    val selfServePromiseUsed: Boolean = false,
    val maxExtensionDays: Int = 14,
    val loadError: String? = null,
    val isSubmittingPayment: Boolean = false,
    val payError: String? = null,
    val paySuccessPaidMinor: Long? = null,
    val paySuccessAdvanceMinor: Long = 0L,
    val isCreatingPromise: Boolean = false,
    val promiseError: String? = null,
    val promiseCreated: Boolean = false,
    val quarterlyLimitError: QuarterlyPromiseLimitError? = null,
    val openingBalances: List<com.pegasusx.retailer.data.model.MigrationOpeningInvoiceDto> = emptyList(),
    val openingBusyInvoiceId: String? = null,
    val openingError: String? = null,
    val openingNotice: String? = null,
) {
    val activeRelationship: CreditRelationshipDto?
        get() = relationships.firstOrNull { it.supplierId == selectedSupplierId }
            ?: relationships.firstOrNull()

    val totalOpenMinor: Long
        get() = statement?.totalOpenMinor ?: invoices.sumOf { it.balanceMinor }

    val totalDisputedMinor: Long
        get() = statement?.totalDisputedMinor ?: invoices.sumOf { it.disputedMinor }

    val totalUndisputedMinor: Long
        get() = statement?.totalUndisputedMinor
            ?: invoices.sumOf { RetailerW3ContractHelper.undisputedMinor(it) }

    val advanceBalanceMinor: Long
        get() = statement?.advanceBalanceMinor ?: 0L
}

@HiltViewModel
class CreditViewModel @Inject constructor(val api: PegasusApi) : ViewModel() {
    private val _uiState = MutableStateFlow(CreditUiState())
    val uiState: StateFlow<CreditUiState> = _uiState.asStateFlow()

    fun loadAll() {
        viewModelScope.launch { refreshAll() }
    }

    suspend fun refreshAll() {
        _uiState.update { it.copy(isLoading = true, loadError = null, openingError = null) }
        val openingResult = runCatching { api.listNasiyaOpeningBalances().invoices }
        val openingInvoices = openingResult.getOrDefault(emptyList())
        val openingErr = openingResult.exceptionOrNull()
            ?.takeIf { it !is UnsupportedOperationException }
            ?.let { RetailerW3ContractHelper.parseArErrorMessage(it, "Failed to load opening balances") }
        _uiState.update {
            it.copy(
                openingBalances = openingInvoices,
                openingError = openingErr,
            )
        }
        val relsResult = runCatching {
            RetailerW3ContractHelper.parseCreditRelationshipsJson(api.getCreditRelationships())
        }
        relsResult.onFailure { err ->
            _uiState.update {
                it.copy(
                    isLoading = false,
                    loadError = RetailerW3ContractHelper.parseArErrorMessage(err, err.message ?: "error"),
                )
            }
            return
        }

        val rels = relsResult.getOrDefault(emptyList())
        val currentSelected = _uiState.value.selectedSupplierId
        val nextSelected = if (currentSelected.isNotBlank() && rels.any { it.supplierId == currentSelected }) {
            currentSelected
        } else {
            rels.firstOrNull()?.supplierId.orEmpty()
        }

        _uiState.update {
            it.copy(
                relationships = rels,
                selectedSupplierId = nextSelected,
            )
        }

        if (nextSelected.isNotBlank()) {
            refreshSupplierDetail(nextSelected)
        } else {
            val invFallback = runCatching {
                RetailerW3ContractHelper.parseArInvoicesJson(api.getArInvoices("OPEN,PARTIAL", 50))
            }.getOrDefault(emptyList())
            _uiState.update {
                it.copy(
                    isLoading = false,
                    statement = null,
                    invoices = RetailerW3ContractHelper.sortArInvoicesOldestFirst(invFallback),
                    promises = emptyList(),
                )
            }
            return
        }

        _uiState.update { it.copy(isLoading = false) }
    }

    suspend fun selectSupplier(supplierId: String) {
        _uiState.update {
            it.copy(
                selectedSupplierId = supplierId,
                payError = null,
                paySuccessPaidMinor = null,
                paySuccessAdvanceMinor = 0L,
                promiseError = null,
                promiseCreated = false,
                quarterlyLimitError = null,
            )
        }
        refreshSupplierDetail(supplierId)
    }

    suspend fun refreshSupplierDetail(supplierId: String) {
        if (supplierId.isBlank()) return
        _uiState.update { it.copy(isRefreshingDetail = true) }
        coroutineScope {
            val stmtDeferred = async { runCatching { api.getArStatement(supplierId, 50) } }
            val invDeferred = async {
                runCatching {
                    RetailerW3ContractHelper.parseArInvoicesJson(api.getArInvoices("OPEN,PARTIAL", 50))
                        .filter { it.supplierId.isBlank() || it.supplierId == supplierId }
                }
            }
            val promDeferred = async { runCatching { api.getArPromises(supplierId) } }

            val stmt = stmtDeferred.await().getOrNull()
            val invList = if (stmt != null && stmt.invoices.isNotEmpty()) {
                stmt.invoices
            } else {
                invDeferred.await().getOrDefault(emptyList())
            }
            val promResp = promDeferred.await().getOrNull()
            val promList = if (stmt != null && stmt.promises.isNotEmpty()) {
                stmt.promises
            } else {
                promResp?.promises.orEmpty()
            }

            _uiState.update {
                it.copy(
                    isRefreshingDetail = false,
                    statement = stmt,
                    invoices = RetailerW3ContractHelper.sortArInvoicesOldestFirst(invList),
                    promises = promList,
                    currentQuarterKey = stmt?.currentQuarterKey
                        ?: promResp?.currentQuarterKey.orEmpty(),
                    selfServePromiseUsed = stmt?.selfServePromiseUsed
                        ?: (promResp?.selfServePromiseUsed ?: false),
                    maxExtensionDays = stmt?.maxExtensionDays
                        ?: (promResp?.maxExtensionDays ?: 14),
                )
            }
        }
    }

    suspend fun submitPayFifo(
        supplierId: String,
        amountMinor: Long,
        paymentMethod: String,
        paymentReference: String? = null,
        earmarkInvoiceId: String? = null,
        idempotencyKey: String = RetailerW3ContractHelper.generateIdempotencyUuid(),
    ): Boolean {
        _uiState.update {
            it.copy(
                isSubmittingPayment = true,
                payError = null,
                paySuccessPaidMinor = null,
                paySuccessAdvanceMinor = 0L,
            )
        }
        val currency = _uiState.value.statement?.currency?.ifBlank { "UZS" } ?: "UZS"
        val res = runCatching {
            api.payArFifo(
                body = PayArFifoRequest(
                    supplierId = supplierId,
                    amountMinor = amountMinor,
                    currency = currency,
                    paymentMethod = paymentMethod,
                    paymentReference = paymentReference?.takeIf { it.isNotBlank() },
                    earmarkInvoiceId = earmarkInvoiceId?.takeIf { it.isNotBlank() },
                ),
                idempotencyKey = idempotencyKey,
            )
        }
        return res.fold(
            onSuccess = { body ->
                _uiState.update {
                    it.copy(
                        isSubmittingPayment = false,
                        paySuccessPaidMinor = body.paidMinor,
                        paySuccessAdvanceMinor = body.advanceAddedMinor,
                    )
                }
                refreshAll()
                true
            },
            onFailure = { err ->
                _uiState.update {
                    it.copy(
                        isSubmittingPayment = false,
                        payError = RetailerW3ContractHelper.parseArErrorMessage(err, err.message ?: "payment_failed"),
                    )
                }
                false
            },
        )
    }

    suspend fun submitPayInvoice(
        invoiceId: String,
        amountMinor: Long,
        paymentMethod: String,
        paymentReference: String? = null,
        idempotencyKey: String = RetailerW3ContractHelper.generateIdempotencyUuid(),
    ): Boolean {
        _uiState.update {
            it.copy(
                isSubmittingPayment = true,
                payError = null,
                paySuccessPaidMinor = null,
                paySuccessAdvanceMinor = 0L,
            )
        }
        val res = runCatching {
            api.payArInvoice(
                invoiceId = invoiceId,
                body = PayArInvoiceRequest(
                    amountMinor = amountMinor,
                    paymentMethod = paymentMethod,
                    paymentReference = paymentReference?.takeIf { it.isNotBlank() },
                ),
                idempotencyKey = idempotencyKey,
            )
        }
        return res.fold(
            onSuccess = {
                _uiState.update {
                    it.copy(
                        isSubmittingPayment = false,
                        paySuccessPaidMinor = amountMinor,
                        paySuccessAdvanceMinor = 0L,
                    )
                }
                refreshAll()
                true
            },
            onFailure = { err ->
                _uiState.update {
                    it.copy(
                        isSubmittingPayment = false,
                        payError = RetailerW3ContractHelper.parseArErrorMessage(err, err.message ?: "payment_failed"),
                    )
                }
                false
            },
        )
    }

    suspend fun submitPromiseExtension(
        supplierId: String,
        extensionDays: Int,
        invoiceId: String? = null,
        idempotencyKey: String = RetailerW3ContractHelper.generateIdempotencyUuid(),
    ): Boolean {
        _uiState.update {
            it.copy(
                isCreatingPromise = true,
                promiseError = null,
                promiseCreated = false,
                quarterlyLimitError = null,
            )
        }
        val res = runCatching {
            api.createArPromise(
                body = CreateArPromiseRequest(
                    supplierId = supplierId,
                    invoiceId = invoiceId?.takeIf { it.isNotBlank() },
                    extensionDays = extensionDays,
                ),
                idempotencyKey = idempotencyKey,
            )
        }
        return res.fold(
            onSuccess = {
                _uiState.update {
                    it.copy(
                        isCreatingPromise = false,
                        promiseCreated = true,
                    )
                }
                refreshSupplierDetail(supplierId)
                true
            },
            onFailure = { err ->
                val limitErr = RetailerW3ContractHelper.parseQuarterlyPromiseLimitError(err)
                _uiState.update {
                    it.copy(
                        isCreatingPromise = false,
                        quarterlyLimitError = limitErr,
                        promiseError = if (limitErr == null) {
                            RetailerW3ContractHelper.parseArErrorMessage(err, err.message ?: "promise_failed")
                        } else {
                            null
                        },
                    )
                }
                false
            },
        )
    }

    suspend fun acknowledgeOpeningBalance(invoiceId: String): Boolean {
        _uiState.update {
            it.copy(
                openingBusyInvoiceId = invoiceId,
                openingError = null,
                openingNotice = null,
            )
        }
        return try {
            val res = api.acknowledgeOpeningBalance(
                invoiceId = invoiceId,
                idempotencyKey = com.pegasusx.retailer.util.RetailerIdempotencyKeys.nasiyaAcknowledge(invoiceId),
            )
            val updated = res.invoice
            _uiState.update { state ->
                val next = if (state.openingBalances.any { it.invoiceId == updated.invoiceId }) {
                    state.openingBalances.map { if (it.invoiceId == updated.invoiceId) updated else it }
                } else {
                    listOf(updated) + state.openingBalances
                }
                state.copy(
                    openingBalances = next,
                    openingBusyInvoiceId = null,
                    openingNotice = "Confirmed",
                )
            }
            true
        } catch (t: Throwable) {
            _uiState.update {
                it.copy(
                    openingBusyInvoiceId = null,
                    openingError = RetailerW3ContractHelper.parseArErrorMessage(t, "Failed to confirm opening balance"),
                )
            }
            false
        }
    }

    suspend fun disputeOpeningBalance(invoiceId: String, reason: String): Boolean {
        val trimmed = reason.trim()
        if (trimmed.isEmpty()) {
            _uiState.update { it.copy(openingError = "Explain why this opening balance differs") }
            return false
        }
        _uiState.update {
            it.copy(
                openingBusyInvoiceId = invoiceId,
                openingError = null,
                openingNotice = null,
            )
        }
        return try {
            val res = api.disputeOpeningBalance(
                invoiceId = invoiceId,
                body = com.pegasusx.retailer.data.model.DisputeOpeningBalanceRequest(reason = trimmed),
                idempotencyKey = com.pegasusx.retailer.util.RetailerIdempotencyKeys.nasiyaDispute(invoiceId, trimmed),
            )
            val updated = res.invoice
            _uiState.update { state ->
                val next = if (state.openingBalances.any { it.invoiceId == updated.invoiceId }) {
                    state.openingBalances.map { if (it.invoiceId == updated.invoiceId) updated else it }
                } else {
                    listOf(updated) + state.openingBalances
                }
                state.copy(
                    openingBalances = next,
                    openingBusyInvoiceId = null,
                    openingNotice = "Disputed",
                )
            }
            true
        } catch (t: Throwable) {
            _uiState.update {
                it.copy(
                    openingBusyInvoiceId = null,
                    openingError = RetailerW3ContractHelper.parseArErrorMessage(t, "Failed to dispute opening balance"),
                )
            }
            false
        }
    }
}

private enum class PayDialogMode {
    FIFO,
    EARMARK_INVOICE,
}

private val PAYMENT_METHODS = listOf("WALLET", "CARD", "BANK_TRANSFER", "CASH")

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CreditScreen(
    onNavigateBack: () -> Unit,
    viewModel: CreditViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    val scope = rememberCoroutineScope()

    var showPayDialog by remember { mutableStateOf(false) }
    var payDialogMode by remember { mutableStateOf(PayDialogMode.FIFO) }
    var targetInvoiceId by remember { mutableStateOf<String?>(null) }
    var fifoEarmarkInvoiceId by remember { mutableStateOf<String?>(null) }
    var amountSomInput by remember { mutableStateOf("") }
    var selectedMethod by remember { mutableStateOf("BANK_TRANSFER") }
    var paymentRefInput by remember { mutableStateOf("") }
    var amountValidationError by remember { mutableStateOf<String?>(null) }

    var extensionDaysInput by remember { mutableStateOf("7") }
    var disputingOpeningInvoice by remember {
        mutableStateOf<com.pegasusx.retailer.data.model.MigrationOpeningInvoiceDto?>(null)
    }
    var openingDisputeReason by remember { mutableStateOf("") }

    LaunchedEffect(Unit) {
        viewModel.loadAll()
    }

    val invalidAmountText = stringResource(R.string.retailer_credit_invalid_amount)

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(stringResource(R.string.retailer_credit_title)) },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(
                            Icons.AutoMirrored.Filled.ArrowBack,
                            contentDescription = stringResource(R.string.retailer_credit_cancel),
                        )
                    }
                },
                actions = {
                    IconButton(
                        onClick = { viewModel.loadAll() },
                        enabled = !uiState.isLoading && !uiState.isRefreshingDetail,
                    ) {
                        Icon(
                            Icons.Outlined.Refresh,
                            contentDescription = stringResource(R.string.retailer_credit_refresh),
                        )
                    }
                },
            )
        },
    ) { innerPadding ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding),
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            uiState.loadError?.let { err ->
                item {
                    Card(
                        colors = CardDefaults.cardColors(
                            containerColor = MaterialTheme.colorScheme.errorContainer,
                        ),
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Text(
                            text = err,
                            modifier = Modifier.padding(12.dp),
                            color = MaterialTheme.colorScheme.onErrorContainer,
                            style = MaterialTheme.typography.bodySmall,
                        )
                    }
                }
            }

            item {
                Card(modifier = Modifier.fillMaxWidth()) {
                    Column(
                        modifier = Modifier.padding(14.dp),
                        verticalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        Text(
                            text = stringResource(R.string.mobile_retailer_nasiya_opening_title),
                            style = MaterialTheme.typography.titleSmall,
                            fontWeight = FontWeight.SemiBold,
                        )
                        uiState.openingNotice?.let { notice ->
                            Text(
                                text = notice,
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.primary,
                            )
                        }
                        uiState.openingError?.let { err ->
                            Text(
                                text = err,
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.error,
                            )
                        }

                        if (uiState.openingBalances.isEmpty()) {
                            Text(
                                text = stringResource(R.string.mobile_retailer_nasiya_opening_empty),
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        } else {
                            Text(
                                text = stringResource(R.string.mobile_retailer_nasiya_opening_frozen_notice),
                                style = MaterialTheme.typography.labelMedium,
                                fontWeight = FontWeight.SemiBold,
                                color = MaterialTheme.colorScheme.tertiary,
                            )
                            Text(
                                text = stringResource(R.string.retailer_desktop_nasiya_opening_dunning_frozen_banner),
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                            uiState.openingBalances.forEach { inv ->
                                HorizontalDivider()
                                Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                                    Row(
                                        modifier = Modifier.fillMaxWidth(),
                                        horizontalArrangement = Arrangement.SpaceBetween,
                                        verticalAlignment = Alignment.CenterVertically,
                                    ) {
                                        Row(
                                            horizontalArrangement = Arrangement.spacedBy(6.dp),
                                            verticalAlignment = Alignment.CenterVertically,
                                        ) {
                                            Text(
                                                text = inv.supplierName?.takeIf { it.isNotBlank() } ?: inv.supplierId,
                                                style = MaterialTheme.typography.bodyMedium,
                                                fontWeight = FontWeight.SemiBold,
                                            )
                                            Text(
                                                text = stringResource(R.string.mobile_retailer_nasiya_opening_migration_badge),
                                                style = MaterialTheme.typography.labelSmall,
                                                color = MaterialTheme.colorScheme.tertiary,
                                                fontWeight = FontWeight.Bold,
                                            )
                                        }
                                        val statusLabel = when {
                                            inv.isAcknowledged -> stringResource(R.string.retailer_desktop_nasiya_opening_acknowledged_status)
                                            inv.isDisputed -> stringResource(R.string.retailer_desktop_nasiya_opening_disputed_status)
                                            else -> stringResource(R.string.retailer_desktop_nasiya_opening_pending_status)
                                        }
                                        Text(
                                            text = statusLabel,
                                            style = MaterialTheme.typography.labelSmall,
                                            fontWeight = FontWeight.Bold,
                                            color = when {
                                                inv.isAcknowledged -> MaterialTheme.colorScheme.primary
                                                inv.isDisputed -> MaterialTheme.colorScheme.error
                                                else -> MaterialTheme.colorScheme.tertiary
                                            },
                                        )
                                    }

                                    Row(
                                        modifier = Modifier.fillMaxWidth(),
                                        horizontalArrangement = Arrangement.SpaceBetween,
                                    ) {
                                        Text(
                                            text = OrderLineContractHelper.formatWholeSom(inv.balanceMinor),
                                            style = MaterialTheme.typography.titleSmall,
                                            fontWeight = FontWeight.Bold,
                                        )
                                        inv.dueAt?.takeIf { it.isNotBlank() }?.let { due ->
                                            Text(
                                                text = due.take(10),
                                                style = MaterialTheme.typography.bodySmall,
                                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                            )
                                        }
                                    }

                                    (inv.externalId ?: inv.orderId)?.takeIf { it.isNotBlank() }?.let { ref ->
                                        Text(
                                            text = "Ref: $ref",
                                            style = MaterialTheme.typography.bodySmall,
                                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                                        )
                                    }

                                    inv.disputeReason?.takeIf { it.isNotBlank() }?.let { reason ->
                                        Text(
                                            text = reason,
                                            style = MaterialTheme.typography.bodySmall,
                                            color = MaterialTheme.colorScheme.error,
                                        )
                                    }

                                    if (inv.isPendingConfirmation) {
                                        Row(
                                            modifier = Modifier.fillMaxWidth().padding(top = 4.dp),
                                            horizontalArrangement = Arrangement.spacedBy(8.dp),
                                        ) {
                                            Button(
                                                enabled = uiState.openingBusyInvoiceId == null,
                                                onClick = {
                                                    scope.launch {
                                                        viewModel.acknowledgeOpeningBalance(inv.invoiceId)
                                                    }
                                                },
                                                modifier = Modifier.weight(1f),
                                            ) {
                                                Text(stringResource(R.string.mobile_retailer_nasiya_opening_confirm_balance))
                                            }
                                            OutlinedButton(
                                                enabled = uiState.openingBusyInvoiceId == null,
                                                onClick = {
                                                    openingDisputeReason = ""
                                                    disputingOpeningInvoice = inv
                                                },
                                                modifier = Modifier.weight(1f),
                                            ) {
                                                Text(stringResource(R.string.mobile_retailer_nasiya_opening_dispute_balance))
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }

            if (uiState.isLoading && uiState.relationships.isEmpty()) {
                item {
                    Row(
                        modifier = Modifier.fillMaxWidth().padding(16.dp),
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        CircularProgressIndicator()
                        Text(
                            text = stringResource(R.string.retailer_credit_loading),
                            style = MaterialTheme.typography.bodyMedium,
                        )
                    }
                }
            } else if (uiState.relationships.isEmpty()) {
                item {
                    Card(modifier = Modifier.fillMaxWidth()) {
                        Text(
                            text = stringResource(R.string.retailer_credit_empty_suppliers),
                            modifier = Modifier.padding(16.dp),
                            style = MaterialTheme.typography.bodyMedium,
                        )
                    }
                }
            } else {
                item {
                    Text(
                        text = stringResource(R.string.retailer_credit_suppliers_heading),
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.SemiBold,
                    )
                }

                items(uiState.relationships, key = { it.supplierId }) { rel ->
                    val isSelected = rel.supplierId == uiState.selectedSupplierId
                    val termsDays = RetailerW3ContractHelper.resolveTermsDays(rel)
                    val graceDays = RetailerW3ContractHelper.resolveGraceDays(rel)
                    Card(
                        onClick = { scope.launch { viewModel.selectSupplier(rel.supplierId) } },
                        colors = CardDefaults.cardColors(
                            containerColor = if (isSelected) {
                                MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.45f)
                            } else {
                                MaterialTheme.colorScheme.surfaceContainerLow
                            },
                        ),
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Column(
                            modifier = Modifier.padding(14.dp),
                            verticalArrangement = Arrangement.spacedBy(6.dp),
                        ) {
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                Text(
                                    text = rel.supplierName?.takeIf { it.isNotBlank() } ?: rel.supplierId,
                                    style = MaterialTheme.typography.titleSmall,
                                    fontWeight = FontWeight.SemiBold,
                                )
                                Text(
                                    text = when {
                                        rel.onHold -> stringResource(R.string.retailer_credit_on_hold)
                                        rel.prepayOnlyLocked -> stringResource(R.string.retailer_credit_prepay_only)
                                        else -> stringResource(R.string.retailer_credit_active)
                                    },
                                    style = MaterialTheme.typography.labelSmall,
                                    color = if (rel.onHold || rel.prepayOnlyLocked) {
                                        MaterialTheme.colorScheme.error
                                    } else {
                                        MaterialTheme.colorScheme.primary
                                    },
                                )
                            }

                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                            ) {
                                CreditMetricCol(
                                    label = stringResource(R.string.retailer_credit_available),
                                    value = OrderLineContractHelper.formatWholeSom(rel.availableCreditMinor),
                                )
                                CreditMetricCol(
                                    label = stringResource(R.string.retailer_credit_balance_due),
                                    value = OrderLineContractHelper.formatWholeSom(rel.currentBalanceMinor),
                                )
                                CreditMetricCol(
                                    label = stringResource(R.string.retailer_credit_limit),
                                    value = OrderLineContractHelper.formatWholeSom(rel.creditLimitMinor),
                                )
                            }

                            Text(
                                text = stringResource(
                                    R.string.retailer_credit_terms_fmt,
                                    termsDays,
                                    graceDays,
                                ),
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                    }
                }
            }

            if (uiState.selectedSupplierId.isNotBlank()) {
                item {
                    Card(modifier = Modifier.fillMaxWidth()) {
                        Column(
                            modifier = Modifier.padding(14.dp),
                            verticalArrangement = Arrangement.spacedBy(10.dp),
                        ) {
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                Text(
                                    text = stringResource(R.string.retailer_credit_statement_heading),
                                    style = MaterialTheme.typography.titleSmall,
                                    fontWeight = FontWeight.SemiBold,
                                )
                                if (uiState.currentQuarterKey.isNotBlank()) {
                                    Text(
                                        text = stringResource(
                                            R.string.retailer_credit_quarter_fmt,
                                            uiState.currentQuarterKey,
                                        ),
                                        style = MaterialTheme.typography.labelSmall,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                                    )
                                }
                            }

                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                            ) {
                                CreditMetricCol(
                                    label = stringResource(R.string.retailer_credit_total_open),
                                    value = OrderLineContractHelper.formatWholeSom(uiState.totalOpenMinor),
                                )
                                CreditMetricCol(
                                    label = stringResource(R.string.retailer_credit_undisputed),
                                    value = OrderLineContractHelper.formatWholeSom(uiState.totalUndisputedMinor),
                                )
                            }

                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                            ) {
                                CreditMetricCol(
                                    label = stringResource(R.string.retailer_credit_disputed),
                                    value = OrderLineContractHelper.formatWholeSom(uiState.totalDisputedMinor),
                                )
                                CreditMetricCol(
                                    label = stringResource(R.string.retailer_credit_advance_balance),
                                    value = OrderLineContractHelper.formatWholeSom(uiState.advanceBalanceMinor),
                                )
                            }

                            uiState.paySuccessPaidMinor?.let { paidMinor ->
                                Text(
                                    text = stringResource(
                                        R.string.retailer_credit_pay_success_fmt,
                                        OrderLineContractHelper.formatWholeSom(paidMinor),
                                        OrderLineContractHelper.formatWholeSom(uiState.paySuccessAdvanceMinor),
                                    ),
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.primary,
                                )
                            }

                            uiState.payError?.let { err ->
                                Text(
                                    text = err,
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.error,
                                )
                            }

                            Button(
                                onClick = {
                                    payDialogMode = PayDialogMode.FIFO
                                    targetInvoiceId = null
                                    fifoEarmarkInvoiceId = null
                                    amountSomInput = if (uiState.totalUndisputedMinor > 0L) {
                                        RetailerW2ContractHelper.minorToSomInput(uiState.totalUndisputedMinor)
                                    } else {
                                        ""
                                    }
                                    amountValidationError = null
                                    showPayDialog = true
                                },
                                modifier = Modifier.fillMaxWidth(),
                            ) {
                                Text(stringResource(R.string.retailer_credit_pay_fifo_cta))
                            }
                        }
                    }
                }

                item {
                    val isPrepayLocked = uiState.activeRelationship?.prepayOnlyLocked == true
                    Card(modifier = Modifier.fillMaxWidth()) {
                        Column(
                            modifier = Modifier.padding(14.dp),
                            verticalArrangement = Arrangement.spacedBy(8.dp),
                        ) {
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                Text(
                                    text = stringResource(R.string.retailer_credit_promises_heading),
                                    style = MaterialTheme.typography.titleSmall,
                                    fontWeight = FontWeight.SemiBold,
                                )
                                Text(
                                    text = if (uiState.selfServePromiseUsed) {
                                        stringResource(R.string.retailer_credit_promise_used)
                                    } else {
                                        stringResource(R.string.retailer_credit_promise_available)
                                    },
                                    style = MaterialTheme.typography.labelSmall,
                                    color = if (uiState.selfServePromiseUsed) {
                                        MaterialTheme.colorScheme.error
                                    } else {
                                        MaterialTheme.colorScheme.primary
                                    },
                                )
                            }

                            uiState.quarterlyLimitError?.let { limitErr ->
                                Card(
                                    colors = CardDefaults.cardColors(
                                        containerColor = MaterialTheme.colorScheme.errorContainer,
                                    ),
                                    modifier = Modifier.fillMaxWidth(),
                                ) {
                                    Column(modifier = Modifier.padding(10.dp)) {
                                        Text(
                                            text = stringResource(R.string.retailer_credit_quarterly_limit_title),
                                            style = MaterialTheme.typography.labelLarge,
                                            color = MaterialTheme.colorScheme.onErrorContainer,
                                        )
                                        Text(
                                            text = stringResource(
                                                R.string.retailer_credit_quarterly_limit_detail,
                                                limitErr.quarterKey ?: uiState.currentQuarterKey.ifBlank { "—" },
                                            ),
                                            style = MaterialTheme.typography.bodySmall,
                                            color = MaterialTheme.colorScheme.onErrorContainer,
                                        )
                                    }
                                }
                            }

                            if (isPrepayLocked) {
                                Text(
                                    text = stringResource(R.string.retailer_credit_promise_prepay_blocked),
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.error,
                                )
                            }

                            uiState.promiseError?.let { err ->
                                Text(
                                    text = err,
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.error,
                                )
                            }

                            if (uiState.promiseCreated) {
                                Text(
                                    text = stringResource(R.string.retailer_credit_promise_created),
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.primary,
                                )
                            }

                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.spacedBy(8.dp),
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                OutlinedTextField(
                                    value = extensionDaysInput,
                                    onValueChange = { extensionDaysInput = it },
                                    label = { Text(stringResource(R.string.retailer_credit_extension_days_label)) },
                                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                                    singleLine = true,
                                    modifier = Modifier.weight(1f),
                                )
                                OutlinedButton(
                                    onClick = {
                                        val days = extensionDaysInput.trim().toIntOrNull()?.coerceIn(
                                            1,
                                            uiState.maxExtensionDays.coerceAtLeast(1),
                                        ) ?: 7
                                        scope.launch {
                                            viewModel.submitPromiseExtension(
                                                supplierId = uiState.selectedSupplierId,
                                                extensionDays = days,
                                            )
                                        }
                                    },
                                    enabled = !uiState.isCreatingPromise && !isPrepayLocked,
                                ) {
                                    Text(stringResource(R.string.retailer_credit_request_extension))
                                }
                            }

                            if (uiState.promises.isEmpty()) {
                                Text(
                                    text = stringResource(R.string.retailer_credit_no_promises),
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            } else {
                                uiState.promises.forEach { prom ->
                                    Row(
                                        modifier = Modifier.fillMaxWidth(),
                                        horizontalArrangement = Arrangement.SpaceBetween,
                                    ) {
                                        Text(
                                            text = "${prom.quarterKey} · ${prom.dueAt}",
                                            style = MaterialTheme.typography.bodySmall,
                                        )
                                        Text(
                                            text = OrderLineContractHelper.formatWholeSom(prom.promisedMinor),
                                            style = MaterialTheme.typography.bodySmall,
                                            fontWeight = FontWeight.Medium,
                                        )
                                    }
                                }
                            }
                        }
                    }
                }
            }

            item {
                Text(
                    text = stringResource(R.string.retailer_credit_invoices_heading),
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.SemiBold,
                )
            }

            if (uiState.invoices.isEmpty()) {
                item {
                    Card(modifier = Modifier.fillMaxWidth()) {
                        Text(
                            text = stringResource(R.string.retailer_credit_no_invoices),
                            modifier = Modifier.padding(14.dp),
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                }
            } else {
                items(uiState.invoices, key = { it.invoiceId }) { inv ->
                    val undisputed = RetailerW3ContractHelper.undisputedMinor(inv)
                    Card(modifier = Modifier.fillMaxWidth()) {
                        Column(
                            modifier = Modifier.padding(14.dp),
                            verticalArrangement = Arrangement.spacedBy(6.dp),
                        ) {
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                Text(
                                    text = "${inv.orderId?.let { "#$it" } ?: inv.invoiceId} · ${inv.status}",
                                    style = MaterialTheme.typography.titleSmall,
                                    fontWeight = FontWeight.Medium,
                                )
                                Text(
                                    text = OrderLineContractHelper.formatWholeSom(inv.balanceMinor),
                                    style = MaterialTheme.typography.titleSmall,
                                    fontWeight = FontWeight.SemiBold,
                                )
                            }

                            Text(
                                text = stringResource(
                                    R.string.retailer_credit_due_fmt,
                                    inv.dueAt?.take(10) ?: "—",
                                ) + (inv.agingBucket?.let {
                                    " · " + stringResource(R.string.retailer_credit_aging_fmt, it)
                                } ?: ""),
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )

                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                Column {
                                    Text(
                                        text = stringResource(
                                            R.string.retailer_credit_undisputed_fmt,
                                            OrderLineContractHelper.formatWholeSom(undisputed),
                                        ),
                                        style = MaterialTheme.typography.labelSmall,
                                    )
                                    if (inv.disputedMinor > 0L) {
                                        Text(
                                            text = stringResource(
                                                R.string.retailer_credit_disputed_fmt,
                                                OrderLineContractHelper.formatWholeSom(inv.disputedMinor),
                                            ),
                                            style = MaterialTheme.typography.labelSmall,
                                            color = MaterialTheme.colorScheme.error,
                                        )
                                    }
                                }

                                OutlinedButton(
                                    onClick = {
                                        payDialogMode = PayDialogMode.EARMARK_INVOICE
                                        targetInvoiceId = inv.invoiceId
                                        val suggested = if (undisputed > 0L) undisputed else inv.balanceMinor
                                        amountSomInput = if (suggested > 0L) {
                                            RetailerW2ContractHelper.minorToSomInput(suggested)
                                        } else {
                                            ""
                                        }
                                        amountValidationError = null
                                        showPayDialog = true
                                    },
                                ) {
                                    Text(stringResource(R.string.retailer_credit_pay_invoice_cta))
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    if (showPayDialog) {
        AlertDialog(
            onDismissRequest = { if (!uiState.isSubmittingPayment) showPayDialog = false },
            title = {
                Text(
                    text = if (payDialogMode == PayDialogMode.FIFO) {
                        stringResource(R.string.retailer_credit_pay_dialog_fifo_title)
                    } else {
                        stringResource(
                            R.string.retailer_credit_pay_dialog_earmark_title,
                            targetInvoiceId.orEmpty(),
                        )
                    },
                )
            },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    OutlinedTextField(
                        value = amountSomInput,
                        onValueChange = {
                            amountSomInput = it
                            amountValidationError = null
                        },
                        label = { Text(stringResource(R.string.retailer_credit_amount_label)) },
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                        singleLine = true,
                        isError = amountValidationError != null,
                        modifier = Modifier.fillMaxWidth(),
                    )
                    amountValidationError?.let { err ->
                        Text(
                            text = err,
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.error,
                        )
                    }

                    Text(
                        text = stringResource(R.string.retailer_credit_method_label),
                        style = MaterialTheme.typography.labelMedium,
                    )
                    LazyRow(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                        items(PAYMENT_METHODS) { method ->
                            FilterChip(
                                selected = selectedMethod == method,
                                onClick = { selectedMethod = method },
                                label = {
                                    Text(
                                        when (method) {
                                            "WALLET" -> stringResource(R.string.retailer_credit_method_wallet)
                                            "CARD" -> stringResource(R.string.retailer_credit_method_card)
                                            "BANK_TRANSFER" -> stringResource(R.string.retailer_credit_method_bank)
                                            else -> stringResource(R.string.retailer_credit_method_cash)
                                        },
                                    )
                                },
                            )
                        }
                    }

                    if (payDialogMode == PayDialogMode.FIFO && uiState.invoices.isNotEmpty()) {
                        Text(
                            text = stringResource(R.string.retailer_credit_earmark_optional),
                            style = MaterialTheme.typography.labelMedium,
                        )
                        LazyRow(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                            item {
                                FilterChip(
                                    selected = fifoEarmarkInvoiceId == null,
                                    onClick = { fifoEarmarkInvoiceId = null },
                                    label = {
                                        Text(stringResource(R.string.retailer_credit_fifo_oldest_first))
                                    },
                                )
                            }
                            items(uiState.invoices) { inv ->
                                FilterChip(
                                    selected = fifoEarmarkInvoiceId == inv.invoiceId,
                                    onClick = { fifoEarmarkInvoiceId = inv.invoiceId },
                                    label = { Text(inv.orderId ?: inv.invoiceId) },
                                )
                            }
                        }
                    }

                    OutlinedTextField(
                        value = paymentRefInput,
                        onValueChange = { paymentRefInput = it },
                        label = { Text(stringResource(R.string.retailer_credit_reference_label)) },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
            },
            confirmButton = {
                Button(
                    enabled = !uiState.isSubmittingPayment,
                    onClick = {
                        val minor = RetailerW2ContractHelper.somInputToMinor(amountSomInput)
                        if (minor == null || minor <= 0L) {
                            amountValidationError = invalidAmountText
                            return@Button
                        }
                        val invId = targetInvoiceId
                        scope.launch {
                            val ok = if (payDialogMode == PayDialogMode.EARMARK_INVOICE && !invId.isNullOrBlank()) {
                                viewModel.submitPayInvoice(
                                    invoiceId = invId,
                                    amountMinor = minor,
                                    paymentMethod = selectedMethod,
                                    paymentReference = paymentRefInput,
                                )
                            } else {
                                viewModel.submitPayFifo(
                                    supplierId = uiState.selectedSupplierId,
                                    amountMinor = minor,
                                    paymentMethod = selectedMethod,
                                    paymentReference = paymentRefInput,
                                    earmarkInvoiceId = fifoEarmarkInvoiceId,
                                )
                            }
                            if (ok) {
                                showPayDialog = false
                                paymentRefInput = ""
                            }
                        }
                    },
                ) {
                    Text(stringResource(R.string.retailer_credit_submit_pay))
                }
            },
            dismissButton = {
                TextButton(
                    enabled = !uiState.isSubmittingPayment,
                    onClick = { showPayDialog = false },
                ) {
                    Text(stringResource(R.string.retailer_credit_cancel))
                }
            },
        )
    }

    disputingOpeningInvoice?.let { inv ->
        AlertDialog(
            onDismissRequest = {
                if (uiState.openingBusyInvoiceId == null) {
                    disputingOpeningInvoice = null
                }
            },
            title = { Text(stringResource(R.string.mobile_retailer_nasiya_opening_dispute_balance)) },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(
                        text = "${inv.supplierName ?: inv.supplierId} — ${OrderLineContractHelper.formatWholeSom(inv.balanceMinor)}",
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.SemiBold,
                    )
                    OutlinedTextField(
                        value = openingDisputeReason,
                        onValueChange = { openingDisputeReason = it },
                        label = { Text(stringResource(R.string.retailer_desktop_nasiya_opening_dispute_reason_label)) },
                        placeholder = { Text(stringResource(R.string.mobile_retailer_nasiya_opening_dispute_hint)) },
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
            },
            confirmButton = {
                Button(
                    enabled = uiState.openingBusyInvoiceId == null && openingDisputeReason.trim().isNotEmpty(),
                    onClick = {
                        scope.launch {
                            val ok = viewModel.disputeOpeningBalance(inv.invoiceId, openingDisputeReason)
                            if (ok) {
                                disputingOpeningInvoice = null
                                openingDisputeReason = ""
                            }
                        }
                    },
                ) {
                    Text(stringResource(R.string.retailer_desktop_nasiya_opening_submit_dispute))
                }
            },
            dismissButton = {
                TextButton(
                    enabled = uiState.openingBusyInvoiceId == null,
                    onClick = { disputingOpeningInvoice = null },
                ) {
                    Text(stringResource(R.string.retailer_credit_cancel))
                }
            },
        )
    }
}

@Composable
private fun CreditMetricCol(label: String, value: String) {
    Column {
        Text(
            text = label,
            style = MaterialTheme.typography.labelSmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Spacer(modifier = Modifier.height(2.dp))
        Text(
            text = value,
            style = MaterialTheme.typography.bodyMedium,
            fontWeight = FontWeight.SemiBold,
        )
    }
}
