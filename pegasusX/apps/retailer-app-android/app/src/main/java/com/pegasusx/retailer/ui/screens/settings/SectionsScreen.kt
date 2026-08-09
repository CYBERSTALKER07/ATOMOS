package com.pegasusx.retailer.ui.screens.settings

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
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
import com.pegasusx.retailer.R
import com.pegasusx.retailer.data.json.*

data class SectionRow(val id: String, val name: String, val aisle: String?)

@HiltViewModel
class SectionsViewModel @Inject constructor(val api: PegasusApi) : ViewModel()

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SectionsScreen(
    onNavigateBack: () -> Unit,
    viewModel: SectionsViewModel = hiltViewModel(),
) {
    val scope = rememberCoroutineScope()
    var name by remember { mutableStateOf("Dairy") }
    var aisle by remember { mutableStateOf("") }
    var skuText by remember { mutableStateOf("") }
    var selectedId by remember { mutableStateOf<String?>(null) }
    var banner by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    val rows = remember { mutableStateListOf<SectionRow>() }

    fun refresh() {
        scope.launch {
            try {
                val items = viewModel.api.getSections().asJsonObject.getAsJsonArray("items")
                rows.clear()
                if (items != null) {
                    for (el in items) {
                        val o = el.asJsonObject
                        rows.add(
                            SectionRow(
                                id = o.get("section_id")?.asString ?: continue,
                                name = o.get("name")?.asString ?: "",
                                aisle = o.get("aisle_tag")?.takeIf { !it.isJsonNull }?.asString,
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
                title = { Text("Sections") },
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
            item {
                Text(
                    "Create departments, map SKUs, assign staff. Auto-enables SECTIONS + STORE_STOCK.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            banner?.let { item { Text(it, color = MaterialTheme.colorScheme.primary) } }
            item {
                Card {
                    Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        OutlinedTextField(value = name, onValueChange = { name = it }, label = { Text("Name") }, modifier = Modifier.fillMaxWidth())
                        OutlinedTextField(value = aisle, onValueChange = { aisle = it }, label = { Text("Aisle tag") }, modifier = Modifier.fillMaxWidth())
                        Button(enabled = !busy, onClick = {
                            scope.launch {
                                busy = true
                                try {
                                    val body = mutableMapOf<String, Any>("name" to name)
                                    if (aisle.isNotBlank()) body["aisle_tag"] = aisle
                                    viewModel.api.createSection(body = body, idempotencyKey = "sec-${System.currentTimeMillis()}")
                                    banner = "Section created"
                                    refresh()
                                } catch (e: Exception) {
                                    banner = e.message
                                } finally {
                                    busy = false
                                }
                            }
                        }) { Text("Create section") }
                    }
                }
            }
            items(rows) { row ->
                Card(onClick = { selectedId = row.id }) {
                    Column(Modifier.padding(14.dp)) {
                        Text(row.name, style = MaterialTheme.typography.titleMedium)
                        row.aisle?.let { Text(stringResource(R.string.mobile_retailer_ui_aisle_it, it), style = MaterialTheme.typography.bodySmall) }
                    }
                }
            }
            selectedId?.let { id ->
                item {
                    Card {
                        Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                            Text("Map SKUs to selected section", style = MaterialTheme.typography.titleSmall)
                            OutlinedTextField(
                                value = skuText,
                                onValueChange = { skuText = it },
                                label = { Text("SKUs (comma-separated)") },
                                modifier = Modifier.fillMaxWidth(),
                            )
                            Button(enabled = !busy, onClick = {
                                scope.launch {
                                    busy = true
                                    try {
                                        val skus = skuText.split(",", " ", "\n").map { it.trim() }.filter { it.isNotEmpty() }
                                        viewModel.api.putSectionSkus(sectionId = id, body = mapOf("skus" to skus))
                                        banner = "SKUs saved"
                                    } catch (e: Exception) {
                                        banner = e.message
                                    } finally {
                                        busy = false
                                    }
                                }
                            }) { Text("Save SKUs") }
                        }
                    }
                }
            }
        }
    }
}
