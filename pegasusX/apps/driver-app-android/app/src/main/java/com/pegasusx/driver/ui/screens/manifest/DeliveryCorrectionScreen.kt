package com.pegasusx.driver.ui.screens.manifest

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.Spring
import androidx.compose.animation.core.spring
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.scaleIn
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.Inventory2
import androidx.compose.material.icons.filled.Remove
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilledIconButton
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FilterChipDefaults
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.IconButtonDefaults
import androidx.compose.material3.LargeTopAppBar
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.nestedscroll.nestedScroll
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasusx.driver.data.model.RejectionReason
import com.pegasusx.driver.ui.components.DriverLoadingState
import com.pegasusx.driver.ui.components.DriverSectionTitle
import com.pegasusx.driver.ui.components.DriverStateKind
import com.pegasusx.driver.ui.components.DriverStatePane
import com.pegasusx.driver.ui.theme.StatusGreen
import com.pegasusx.driver.ui.theme.StatusRed
import kotlinx.coroutines.launch
import java.text.NumberFormat
import java.util.Locale
import com.pegasusx.driver.ui.screens.manifest.components.CorrectionFooter
import com.pegasusx.driver.ui.screens.manifest.components.LineItemCard
import com.pegasusx.driver.ui.screens.manifest.components.ModificationSheetContent
import com.pegasusx.driver.ui.screens.manifest.components.formatAmount

// ─────────────────────────────────────────────────────────────────────────────
//  DeliveryCorrectionScreen — Line-Item Reconciliation UI
// ─────────────────────────────────────────────────────────────────────────────

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DeliveryCorrectionScreen(
    onClose: () -> Unit,
    onComplete: () -> Unit,
    viewModel: CorrectionViewModel = hiltViewModel()
) {
    val state by viewModel.state.collectAsState()
    val editingIndex by viewModel.editingIndex.collectAsState()
    val scrollBehavior = TopAppBarDefaults.exitUntilCollapsedScrollBehavior()
    val scope = rememberCoroutineScope()
    var showConfirmDialog by remember { mutableStateOf(false) }
    var showStartTransitDialog by remember { mutableStateOf(false) }

    // Navigate out on success
    if (state.submitSuccess) {
        onComplete()
        return
    }

    Scaffold(
        modifier = Modifier.nestedScroll(scrollBehavior.nestedScrollConnection),
        topBar = {
            LargeTopAppBar(
                title = {
                    Column {
                        Text("Verify Cargo")
                        if (state.retailerName.isNotBlank()) {
                            Text(
                                text = state.retailerName,
                                style = MaterialTheme.typography.bodyMedium,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )
                        }
                    }
                },
                navigationIcon = {
                    IconButton(onClick = onClose) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Filled.ArrowBack,
                            contentDescription = "Back"
                        )
                    }
                },
                actions = {
                    // Modification count badge
                    if (state.hasModifications) {
                        Surface(
                            shape = RoundedCornerShape(8.dp),
                            color = MaterialTheme.colorScheme.errorContainer
                        ) {
                            Text(
                                text = "${state.modifiedCount} MODIFIED",
                                style = MaterialTheme.typography.labelSmall,
                                fontWeight = FontWeight.Bold,
                                color = MaterialTheme.colorScheme.onErrorContainer,
                                modifier = Modifier.padding(horizontal = 10.dp, vertical = 4.dp)
                            )
                        }
                        Spacer(modifier = Modifier.width(12.dp))
                    }
                },
                scrollBehavior = scrollBehavior,
                colors = TopAppBarDefaults.largeTopAppBarColors(
                    containerColor = MaterialTheme.colorScheme.surface,
                    scrolledContainerColor = MaterialTheme.colorScheme.surface
                )
            )
        },
        bottomBar = {
            if (!state.isLoading && state.audits.isNotEmpty()) {
                CorrectionFooter(
                    state = state,
                    onConfirm = {
                        if (state.hasModifications) {
                            showConfirmDialog = true
                        } else {
                            viewModel.submitWithoutModification()
                        }
                    }
                )
            }
        }
    ) { padding ->
        when {
            state.isLoading -> LoadingState(Modifier.padding(padding))
            state.error != null && state.audits.isEmpty() -> ErrorState(
                message = state.error!!,
                modifier = Modifier.padding(padding)
            )
            else -> {
                LazyColumn(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(padding),
                    contentPadding = PaddingValues(start = 16.dp, end = 16.dp, top = 8.dp, bottom = 200.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp)
                ) {
                    // Section header
                    if (state.isPartial) {
                        item {
                            Card(
                                modifier = Modifier.fillMaxWidth().padding(bottom = 8.dp),
                                colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.primaryContainer)
                            ) {
                                Column(modifier = Modifier.padding(16.dp)) {
                                    Text(
                                        text = "Partial Order Split",
                                        style = MaterialTheme.typography.titleMedium,
                                        fontWeight = FontWeight.Bold,
                                        color = MaterialTheme.colorScheme.onPrimaryContainer
                                    )
                                    Spacer(modifier = Modifier.height(4.dp))
                                    Text(
                                        text = "This order is split across multiple trucks. Press Start Transit when you are heading to this route to notify the other driver.",
                                        style = MaterialTheme.typography.bodyMedium,
                                        color = MaterialTheme.colorScheme.onPrimaryContainer.copy(alpha = 0.8f)
                                    )
                                    Spacer(modifier = Modifier.height(12.dp))
                                    Button(
                                        onClick = { showStartTransitDialog = true },
                                        modifier = Modifier.fillMaxWidth()
                                    ) {
                                        Text("Start Transit")
                                    }
                                }
                            }
                        }
                    }

                    item {
                        DriverSectionTitle(title = "MANIFEST ITEMS · ${state.audits.size}")
                    }

                    // Order ID mono badge
                    item {
                        Text(
                            text = state.orderId,
                            style = MaterialTheme.typography.bodySmall.copy(
                                fontFamily = FontFamily.Monospace,
                                letterSpacing = 0.5.sp
                            ),
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }

                    // Line item cards
                    itemsIndexed(state.audits, key = { _, a -> a.item.productId }) { index, audit ->
                        LineItemCard(
                            audit = audit,
                            onModifyClick = { viewModel.openEditor(index) }
                        )
                    }
                }
            }
        }
    }

    // ── Modification Bottom Sheet ────────────────────────────────────────────
    if (editingIndex >= 0 && editingIndex < state.audits.size) {
        val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
        val audit = state.audits[editingIndex]

        ModalBottomSheet(
            onDismissRequest = { viewModel.closeEditor() },
            sheetState = sheetState,
            shape = RoundedCornerShape(topStart = 32.dp, topEnd = 32.dp),
            containerColor = MaterialTheme.colorScheme.surface,
            dragHandle = {
                Surface(
                    modifier = Modifier
                        .padding(vertical = 12.dp)
                        .size(width = 32.dp, height = 4.dp),
                    shape = CircleShape,
                    color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.4f)
                ) {}
            }
        ) {
            ModificationSheetContent(
                audit = audit,
                onAcceptedChange = { viewModel.updateAcceptedQty(editingIndex, it) },
                onReasonChange = { viewModel.updateReason(editingIndex, it) },
                onDone = {
                    scope.launch {
                        sheetState.hide()
                        viewModel.closeEditor()
                    }
                }
            )
        }
    }

    // ── Confirmation Dialog ──────────────────────────────────────────────────
    if (showStartTransitDialog) {
        AlertDialog(
            onDismissRequest = { showStartTransitDialog = false },
            title = { Text("Start Transit", fontWeight = FontWeight.Bold) },
            text = { Text("Are you sure you want to start transit? This will notify the other driver that you are on the way to this shared route.") },
            confirmButton = {
                Button(
                    onClick = {
                        showStartTransitDialog = false
                        viewModel.startTransitForPartialOrder()
                    }
                ) {
                    Text("Confirm")
                }
            },
            dismissButton = {
                TextButton(onClick = { showStartTransitDialog = false }) {
                    Text("Cancel")
                }
            }
        )
    }

    if (showConfirmDialog) {
        AlertDialog(
            onDismissRequest = { showConfirmDialog = false },
            icon = {
                Icon(
                    imageVector = Icons.Default.Warning,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.error,
                    modifier = Modifier.size(32.dp)
                )
            },
            title = {
                Text(
                    text = "Confirm Amendment",
                    style = MaterialTheme.typography.headlineSmall
                )
            },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(
                        text = "You are about to submit an amended delivery manifest with ${state.modifiedCount} modifications.",
                        style = MaterialTheme.typography.bodyMedium
                    )
                    Surface(
                        color = MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.5f),
                        shape = RoundedCornerShape(8.dp),
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Column(modifier = Modifier.padding(12.dp)) {
                            Text(
                                text = "Refund Amount",
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.onErrorContainer
                            )
                            Text(
                                text = state.refundDelta.formatAmount(),
                                style = MaterialTheme.typography.titleLarge,
                                fontWeight = FontWeight.ExtraBold,
                                color = MaterialTheme.colorScheme.error
                            )
                        }
                    }
                }
            },
            confirmButton = {
                Button(
                    onClick = {
                        showConfirmDialog = false
                        viewModel.submitAmendment()
                    },
                    colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.error)
                ) {
                    Text("Confirm Amendment")
                }
            },
            dismissButton = {
                TextButton(onClick = { showConfirmDialog = false }) {
                    Text("Cancel")
                }
            }
        )
    }
}



// ─────────────────────────────────────────────────────────────────────────────
//  Loading & Error States
// ─────────────────────────────────────────────────────────────────────────────

@Composable
private fun LoadingState(modifier: Modifier = Modifier) {
    Box(
        modifier = modifier.fillMaxSize(),
        contentAlignment = Alignment.Center,
    ) {
        DriverLoadingState(
            title = "Loading manifest",
            body = "Fetching line items and audit state for this delivery.",
            compact = true,
        )
    }
}

@Composable
private fun ErrorState(message: String, modifier: Modifier = Modifier) {
    Box(
        modifier = modifier
            .fillMaxSize()
            .padding(24.dp),
        contentAlignment = Alignment.Center,
    ) {
        DriverStatePane(
            kind = DriverStateKind.Error,
            headline = "Failed to load order",
            body = message,
            compact = true,
        )
    }
}
