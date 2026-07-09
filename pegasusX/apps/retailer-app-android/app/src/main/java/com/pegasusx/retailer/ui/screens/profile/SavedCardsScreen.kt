package com.pegasusx.retailer.ui.screens.profile

import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items

import androidx.compose.foundation.lazy.grid.GridItemSpan

import androidx.compose.foundation.lazy.grid.LazyVerticalGrid

import androidx.compose.foundation.lazy.grid.GridCells

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.ArrowBack
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.CreditCard
import androidx.compose.material.icons.rounded.Delete
import androidx.compose.material.icons.rounded.Star
import androidx.compose.material.icons.rounded.StarBorder
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasusx.retailer.ui.components.PegasusEmptyState

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SavedCardsScreen(
    onNavigateBack: () -> Unit,
    returnTo: String? = null,
    onReturnToDeliveryPayment: (() -> Unit)? = null,
    viewModel: SavedCardsViewModel = hiltViewModel()
) {
    val uiState by viewModel.uiState.collectAsState()
    var showAddDialog by remember { mutableStateOf(false) }

    LaunchedEffect(uiState.cardJustAdded) {
        if (uiState.cardJustAdded && returnTo == "delivery_payment") {
            viewModel.clearCardJustAdded()
            onReturnToDeliveryPayment?.invoke()
        }
    }

    val handleBack: () -> Unit = {
        if (returnTo == "delivery_payment") {
            onReturnToDeliveryPayment?.invoke()
        } else {
            onNavigateBack()
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Saved Cards") },
                navigationIcon = {
                    IconButton(onClick = handleBack) {
                        Icon(imageVector = Icons.AutoMirrored.Rounded.ArrowBack, contentDescription = "Back")
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.surface,
                    titleContentColor = MaterialTheme.colorScheme.onSurface
                )
            )
        },
        floatingActionButton = {
            FloatingActionButton(onClick = { showAddDialog = true }) {
                Icon(Icons.Rounded.Add, contentDescription = "Add Card")
            }
        }
    ) { padding ->
        Column(modifier = Modifier.fillMaxSize().padding(padding)) {
            if (returnTo == "delivery_payment") {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 16.dp, vertical = 8.dp)
                        .clip(MaterialTheme.shapes.medium)
                        .background(MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.45f))
                        .padding(horizontal = 12.dp, vertical = 10.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.SpaceBetween,
                ) {
                    Text(
                        text = "Add a card, then return to complete delivery payment.",
                        modifier = Modifier.weight(1f),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onPrimaryContainer,
                    )
                    TextButton(onClick = handleBack) {
                        Text("Return", color = MaterialTheme.colorScheme.onPrimaryContainer)
                    }
                }
            }

            val syncMessage = when {
                uiState.loadIssue != null -> uiState.error ?: uiState.syncMessage.orEmpty()
                uiState.isLoading && uiState.cards.isNotEmpty() -> "Syncing saved cards..."
                else -> null
            }

            if (syncMessage != null) {
                val loadIssue = uiState.loadIssue
                val containerColor = when (loadIssue) {
                    SavedCardsLoadIssue.RESTRICTED -> MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.5f)
                    SavedCardsLoadIssue.OFFLINE -> MaterialTheme.colorScheme.tertiaryContainer.copy(alpha = 0.5f)
                    SavedCardsLoadIssue.ERROR -> MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.35f)
                    null -> MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.4f)
                }
                val contentColor = when (loadIssue) {
                    SavedCardsLoadIssue.RESTRICTED -> MaterialTheme.colorScheme.onErrorContainer
                    SavedCardsLoadIssue.OFFLINE -> MaterialTheme.colorScheme.onTertiaryContainer
                    SavedCardsLoadIssue.ERROR -> MaterialTheme.colorScheme.onErrorContainer
                    null -> MaterialTheme.colorScheme.onPrimaryContainer
                }

                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 16.dp, vertical = 8.dp)
                        .clip(MaterialTheme.shapes.medium)
                        .background(containerColor)
                        .padding(horizontal = 12.dp, vertical = 10.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        text = syncMessage,
                        modifier = Modifier.weight(1f),
                        style = MaterialTheme.typography.labelMedium,
                        color = contentColor,
                    )
                    if (loadIssue != null) {
                        TextButton(onClick = viewModel::loadCards) {
                            Text("Retry", color = contentColor)
                        }
                    }
                }
            }

            Box(modifier = Modifier.fillMaxSize()) {
                if (uiState.isLoading && uiState.cards.isEmpty()) {
                    CircularProgressIndicator(modifier = Modifier.align(Alignment.Center))
                } else if (uiState.cards.isEmpty()) {
                    PegasusEmptyState(
                        icon = Icons.Rounded.CreditCard,
                        title = when (uiState.loadIssue) {
                            SavedCardsLoadIssue.RESTRICTED -> "Saved Cards Access Restricted"
                            SavedCardsLoadIssue.OFFLINE -> "Saved Cards Offline"
                            SavedCardsLoadIssue.ERROR -> "Saved Cards Unavailable"
                            null -> "No Saved Cards"
                        },
                        message = uiState.error ?: "Add a payment method for faster checkout",
                        modifier = Modifier.align(Alignment.Center)
                    )
                } else {
                    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        
                        modifier = Modifier.fillMaxSize(),
                        contentPadding = PaddingValues(16.dp),
                        verticalArrangement = Arrangement.spacedBy(8.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp)
    ) {
                        items(uiState.cards) { card ->
                            Card(
                                modifier = Modifier.fillMaxWidth(),
                                colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainer)
                            ) {
                                Row(
                                    modifier = Modifier.fillMaxWidth().padding(16.dp),
                                    horizontalArrangement = Arrangement.SpaceBetween,
                                    verticalAlignment = Alignment.CenterVertically
                                ) {
                                    Column(modifier = Modifier.weight(1f)) {
                                        Text(card.pan, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                                        Text(card.type, style = MaterialTheme.typography.bodySmall)
                                    }
                                    Row(horizontalArrangement = Arrangement.End) {
                                        IconButton(onClick = { viewModel.setDefault(card.id) }) {
                                            Icon(
                                                imageVector = if (card.isDefault) Icons.Rounded.Star else Icons.Rounded.StarBorder,
                                                contentDescription = "Default",
                                                tint = if (card.isDefault) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant
                                            )
                                        }
                                        IconButton(onClick = { viewModel.deleteCard(card.id) }) {
                                            Icon(Icons.Rounded.Delete, contentDescription = "Delete", tint = MaterialTheme.colorScheme.error)
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    if (showAddDialog) {
        AlertDialog(
            onDismissRequest = { showAddDialog = false },
            title = { Text("Add Card") },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    if (uiState.addError != null) {
                        Text(uiState.addError!!, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
                    }
                    Text(
                        "This will initiate tokenization securely via GlobalPay.",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            },
            confirmButton = {
                TextButton(onClick = {
                    viewModel.initiateCard()
                    showAddDialog = false
                }) {
                    Text("Start Tokenization")
                }
            },
            dismissButton = {
                TextButton(onClick = { showAddDialog = false }) { Text("Cancel") }
            }
        )
    }

    if (uiState.initiateSession != null) {
        var otp by remember { mutableStateOf("") }
        AlertDialog(
            onDismissRequest = { viewModel.cancelAdd() },
            title = { Text("Confirm OTP") },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("Sent to ${uiState.otpPhone ?: "phone"}")
                    if (uiState.addError != null) {
                        Text(uiState.addError!!, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
                    }
                    OutlinedTextField(
                        value = otp,
                        onValueChange = { otp = it },
                        label = { Text("Code") },
                        modifier = Modifier.fillMaxWidth()
                    )
                }
            },
            confirmButton = {
                TextButton(onClick = {
                    if (otp.isNotBlank()) {
                        viewModel.confirmCard(otp)
                    }
                }) {
                    Text("Confirm")
                }
            },
            dismissButton = {
                TextButton(onClick = { viewModel.cancelAdd() }) { Text("Cancel") }
            }
        )
    }
}
