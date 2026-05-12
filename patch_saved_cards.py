import os

pkg_dir = "pegasus/apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/profile"

vm_code = """package com.pegasus.retailer.ui.screens.profile

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
"""

screen_code = """package com.pegasus.retailer.ui.screens.profile

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasus.retailer.ui.components.PegasusEmptyState

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SavedCardsScreen(
    onNavigateBack: () -> Unit,
    viewModel: SavedCardsViewModel = hiltViewModel()
) {
    val uiState by viewModel.uiState.collectAsState()
    var showAddDialog by remember { mutableStateOf(false) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Saved Cards") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
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
        Box(modifier = Modifier.fillMaxSize().padding(padding)) {
            if (uiState.isLoading) {
                CircularProgressIndicator(modifier = Modifier.align(Alignment.Center))
            } else if (uiState.error \!= null) {
                Text(
                    text = uiState.error\!\!,
                    color = MaterialTheme.colorScheme.error,
                    modifier = Modifier.align(Alignment.Center).padding(16.dp)
                )
            } else if (uiState.cards.isEmpty()) {
                PegasusEmptyState(
                    icon = Icons.Rounded.CreditCard,
                    title = "No Saved Cards",
                    description = "Add a payment method for faster checkout",
                    modifier = Modifier.align(Alignment.Center)
                )
            } else {
                LazyColumn(
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(16.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp)
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

    if (showAddDialog) {
        var cardNumber by remember { mutableStateOf("") }
        var expire by remember { mutableStateOf("") }
        
        AlertDialog(
            onDismissRequest = { showAddDialog = false },
            title = { Text("Add Card") },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    if (uiState.addError \!= null) {
                        Text(uiState.addError\!\!, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
                    }
                    OutlinedTextField(
                        value = cardNumber,
                        onValueChange = { cardNumber = it },
                        label = { Text("Card Number") },
                        modifier = Modifier.fillMaxWidth()
                    )
                    OutlinedTextField(
                        value = expire,
                        onValueChange = { expire = it },
                        label = { Text("MMYY") },
                        modifier = Modifier.fillMaxWidth()
                    )
                }
            },
            confirmButton = {
                TextButton(onClick = {
                    if (cardNumber.isNotBlank() && expire.isNotBlank()) {
                        viewModel.initiateCard(cardNumber, expire)
                        showAddDialog = false
                    }
                }) {
                    Text("Next")
                }
            },
            dismissButton = {
                TextButton(onClick = { showAddDialog = false }) { Text("Cancel") }
            }
        )
    }

    if (uiState.initiateSession \!= null) {
        var otp by remember { mutableStateOf("") }
        AlertDialog(
            onDismissRequest = { viewModel.cancelAdd() },
            title = { Text("Confirm OTP") },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("Sent to ${uiState.otpPhone ?: "phone"}")
                    if (uiState.addError \!= null) {
                        Text(uiState.addError\!\!, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
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
"""

with open(f"{pkg_dir}/SavedCardsViewModel.kt", "w") as f:
    f.write(vm_code)

with open(f"{pkg_dir}/SavedCardsScreen.kt", "w") as f:
    f.write(screen_code)

print("SavedCardsViewModel and Screen created.")
