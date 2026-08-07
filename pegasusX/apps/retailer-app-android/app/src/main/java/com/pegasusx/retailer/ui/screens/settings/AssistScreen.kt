package com.pegasusx.retailer.ui.screens.settings

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
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
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.ViewModel
import com.pegasusx.retailer.data.api.PegasusApi
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.launch
import javax.inject.Inject

data class AssistTicketRow(val id: String, val note: String, val status: String)
data class SectionPick(val id: String, val name: String)

@HiltViewModel
class AssistViewModel @Inject constructor(val api: PegasusApi) : ViewModel()

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AssistScreen(
    onNavigateBack: () -> Unit,
    viewModel: AssistViewModel = hiltViewModel(),
) {
    val scope = rememberCoroutineScope()
    var note by remember { mutableStateOf("") }
    var sectionId by remember { mutableStateOf<String?>(null) }
    var sectionLabel by remember { mutableStateOf("—") }
    var banner by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    val tickets = remember { mutableStateListOf<AssistTicketRow>() }
    val sections = remember { mutableStateListOf<SectionPick>() }

    fun refresh() {
        scope.launch {
            try {
                val secs = viewModel.api.getSections().asJsonObject.getAsJsonArray("items")
                sections.clear()
                if (secs != null) {
                    for (el in secs) {
                        val o = el.asJsonObject
                        val id = o.get("section_id")?.asString ?: continue
                        val name = o.get("name")?.asString ?: id
                        sections.add(SectionPick(id, name))
                    }
                    if (sectionId == null && sections.isNotEmpty()) {
                        sectionId = sections[0].id
                        sectionLabel = sections[0].name
                    }
                }
                val items = viewModel.api.getAssistTickets().asJsonObject.getAsJsonArray("items")
                tickets.clear()
                if (items != null) {
                    for (el in items) {
                        val o = el.asJsonObject
                        tickets.add(
                            AssistTicketRow(
                                id = o.get("ticket_id")?.asString ?: continue,
                                note = o.get("note")?.asString ?: "",
                                status = o.get("status")?.asString ?: "",
                            ),
                        )
                    }
                }
            } catch (e: Exception) {
                banner = e.message
            }
        }
    }

    LaunchedEffect(Unit) { refresh() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Floor assist") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
            )
        },
    ) { padding ->
        LazyColumn(
            Modifier.fillMaxSize().padding(padding).padding(horizontal = 16.dp),
            contentPadding = PaddingValues(vertical = 12.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            banner?.let { item { Text(it, color = MaterialTheme.colorScheme.primary) } }
            item {
                Card {
                    Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text(stringResource(R.string.mobile_retailer_ui_section_sectionlabel, sectionLabel), style = MaterialTheme.typography.titleSmall)
                        if (sections.size > 1) {
                            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                sections.take(4).forEach { s ->
                                    OutlinedButton(onClick = {
                                        sectionId = s.id
                                        sectionLabel = s.name
                                    }) { Text(s.name) }
                                }
                            }
                        }
                        OutlinedTextField(
                            value = note,
                            onValueChange = { note = it },
                            label = { Text("Help note") },
                            modifier = Modifier.fillMaxWidth(),
                        )
                        Button(enabled = !busy && sectionId != null && note.isNotBlank(), onClick = {
                            scope.launch {
                                busy = true
                                try {
                                    viewModel.api.createAssistTicket(
                                        body = mapOf(
                                            "section_id" to (sectionId ?: ""),
                                            "note" to note.trim(),
                                        ),
                                        idempotencyKey = "assist-${System.currentTimeMillis()}",
                                    )
                                    note = ""
                                    banner = "Ticket opened"
                                    refresh()
                                } catch (e: Exception) {
                                    banner = e.message
                                } finally {
                                    busy = false
                                }
                            }
                        }) { Text("Open ticket") }
                    }
                }
            }
            items(tickets) { t ->
                Card {
                    Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                        Text(stringResource(R.string.mobile_retailer_ui_status_note, t.status, t.note))
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            if (t.status == "OPEN") {
                                OutlinedButton(onClick = {
                                    scope.launch {
                                        try {
                                            viewModel.api.claimAssistTicket(t.id)
                                            refresh()
                                        } catch (e: Exception) {
                                            banner = e.message
                                        }
                                    }
                                }) { Text("Claim") }
                            }
                            if (t.status == "OPEN" || t.status == "CLAIMED") {
                                OutlinedButton(onClick = {
                                    scope.launch {
                                        try {
                                            viewModel.api.completeAssistTicket(t.id)
                                            refresh()
                                        } catch (e: Exception) {
                                            banner = e.message
                                        }
                                    }
                                }) { Text("Complete") }
                            }
                        }
                    }
                }
            }
        }
    }
}
