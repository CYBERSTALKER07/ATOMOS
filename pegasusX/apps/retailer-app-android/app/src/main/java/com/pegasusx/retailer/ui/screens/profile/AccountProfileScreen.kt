package com.pegasusx.retailer.ui.screens.profile

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material.icons.outlined.Schedule
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasusx.retailer.ui.screens.profile.components.CreditProfileCard
import com.pegasusx.retailer.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AccountProfileScreen(
    onBack: () -> Unit,
    viewModel: AccountProfileViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Account") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Outlined.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
            )
        },
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .verticalScroll(rememberScrollState())
                .padding(PaddingValues(horizontal = 16.dp, vertical = 12.dp)),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            CreditProfileCard()

            Text(
                text = stringResource(R.string.mobile_retailer_ui_business_details_and_receiving_hours_used_for_dispatch_sla_sched),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )

            if (uiState.error != null) {
                Text(
                    text = uiState.error.orEmpty(),
                    color = MaterialTheme.colorScheme.error,
                    style = MaterialTheme.typography.bodySmall,
                )
            }
            if (uiState.saveMessage != null) {
                Text(
                    text = uiState.saveMessage.orEmpty(),
                    color = MaterialTheme.colorScheme.primary,
                    style = MaterialTheme.typography.bodySmall,
                )
            }

            OutlinedTextField(
                value = uiState.name,
                onValueChange = viewModel::onNameChanged,
                label = { Text("Entity name") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
                enabled = !uiState.isLoading && !uiState.isSaving,
            )
            OutlinedTextField(
                value = uiState.company,
                onValueChange = viewModel::onCompanyChanged,
                label = { Text("Company") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
                enabled = !uiState.isLoading && !uiState.isSaving,
            )
            OutlinedTextField(
                value = uiState.phone,
                onValueChange = {},
                label = { Text("Phone") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
                readOnly = true,
                enabled = false,
            )
            OutlinedTextField(
                value = uiState.regionId,
                onValueChange = viewModel::onRegionIdChanged,
                label = { Text("Region ID") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
                enabled = !uiState.isLoading && !uiState.isSaving,
            )

            Text(
                text = stringResource(R.string.mobile_retailer_ui_receiving_window),
                style = MaterialTheme.typography.titleSmall,
            )
            OutlinedTextField(
                value = uiState.receivingWindowOpen,
                onValueChange = viewModel::onReceivingWindowOpenChanged,
                label = { Text("Opens (HH:MM)") },
                placeholder = { Text("09:00") },
                leadingIcon = {
                    Icon(Icons.Outlined.Schedule, contentDescription = null)
                },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
                isError = uiState.openWindowError != null,
                supportingText = uiState.openWindowError?.let { { Text(it) } },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                enabled = !uiState.isLoading && !uiState.isSaving,
            )
            OutlinedTextField(
                value = uiState.receivingWindowClose,
                onValueChange = viewModel::onReceivingWindowCloseChanged,
                label = { Text("Closes (HH:MM)") },
                placeholder = { Text("18:00") },
                leadingIcon = {
                    Icon(Icons.Outlined.Schedule, contentDescription = null)
                },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
                isError = uiState.closeWindowError != null,
                supportingText = uiState.closeWindowError?.let { { Text(it) } },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                enabled = !uiState.isLoading && !uiState.isSaving,
            )

            Button(
                onClick = viewModel::save,
                modifier = Modifier.fillMaxWidth(),
                enabled = !uiState.isLoading && !uiState.isSaving,
            ) {
                if (uiState.isSaving) {
                    CircularProgressIndicator(
                        modifier = Modifier.padding(end = 8.dp),
                        strokeWidth = 2.dp,
                    )
                }
                Text("Save profile")
            }
        }
    }
}
