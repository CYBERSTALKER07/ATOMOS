package com.pegasusx.factory.ui.screens.staff

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Button
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import com.pegasusx.factory.data.model.StaffMember
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.factory.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import com.pegasusx.factory.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun StaffDetailScreen(
    api: FactoryApi,
    staffId: String,
    onBack: () -> Unit,
) {
    var staff by remember { mutableStateOf<StaffMember?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var pin by remember { mutableStateOf("") }
    var setMsg by remember { mutableStateOf<String?>(null) }
    var setting by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    fun load() {
        loading = true
        error = null
        scope.launch {
            try {
                val resp = api.getStaffDetail(staffId)
                if (resp.isSuccessful && resp.body() != null) {
                    staff = resp.body()!!
                } else {
                    error = "Failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(staffId) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Staff detail") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back")
                    }
                },
            )
        },
    ) { innerPadding ->
        when {
            loading -> PegasusLoadingState(
                title = stringResource(R.string.mobile_factory_ui_loading_staff),
                body = "Fetching operator profile.",
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            )
            error != null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Unable to load staff",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            )
            staff != null -> Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding)
                    .padding(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                Text(staff!!.name, style = MaterialTheme.typography.headlineSmall)
                Text(staff!!.role, style = MaterialTheme.typography.bodyLarge)
                DetailRow(label = stringResource(R.string.factory_portal_staff_id_text_staff_id), value = staff!!.id)
                DetailRow(label = stringResource(R.string.common_field_phone), value = staff!!.phone.ifBlank { "—" })
                DetailRow(label = stringResource(R.string.factory_portal_fleet_text_status), value = staff!!.status.ifBlank { "ACTIVE" })
                DetailRow(label = stringResource(R.string.factory_portal_staff_id_text_joined), value = staff!!.joinedAt.ifBlank { "—" })
                OutlinedTextField(
                    value = pin,
                    onValueChange = { pin = it },
                    label = { Text("New PIN") },
                    modifier = Modifier.fillMaxWidth(),
                )
                Button(
                    onClick = {
                        setting = true
                        setMsg = null
                        scope.launch {
                            try {
                                val resp = api.setStaffPassword(
                                    staffId,
                                    mapOf("pin" to pin),
                                    "factory-staff-set-password:$staffId",
                                )
                                setMsg = if (resp.isSuccessful) "Password set" else "Failed (${resp.code()})"
                                if (resp.isSuccessful) pin = ""
                            } catch (e: Exception) {
                                setMsg = e.message ?: "set_password_failed"
                            } finally {
                                setting = false
                            }
                        }
                    },
                    enabled = !setting && pin.trim().length >= 4,
                    modifier = Modifier.fillMaxWidth(),
                ) { Text(if (setting) "Saving…" else "Set login PIN") }
                if (setMsg != null) {
                    Text(setMsg!!, style = MaterialTheme.typography.bodySmall)
                }
            }
        }
    }
}

@Composable
private fun DetailRow(label: String, value: String) {
    Column(modifier = Modifier.fillMaxWidth()) {
        Text(label, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
        Text(value, style = MaterialTheme.typography.bodyMedium, fontFamily = FontFamily.Monospace)
    }
}
