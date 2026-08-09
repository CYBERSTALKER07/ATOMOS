package com.pegasusx.supplier.ui.screens.auth

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Phone
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.pegasus.design.PegasusRuntimeBanner
import com.pegasus.design.PegasusRuntimeTone
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.ui.viewmodel.OnboardingViewModel
import com.pegasusx.supplier.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RegisterScreen(
    onBack: () -> Unit,
    onRegistered: () -> Unit,
    viewModel: OnboardingViewModel = hiltViewModel(),
) {
    val state by viewModel.registerState.collectAsStateWithLifecycle()
    val stepLabels = listOf("Phone", "Verification", "Profile")

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Register supplier") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .padding(padding)
                .verticalScroll(rememberScrollState())
                .padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Text(
                stringResource(R.string.mobile_supplier_ui_step_step_1_of_3_step, state.step + 1, stepLabels[state.step]),
                style = MaterialTheme.typography.titleSmall,
                color = MaterialTheme.colorScheme.primary,
            )
            when (state.step) {
                0 -> {
                    OutlinedTextField(
                        value = state.phone,
                        onValueChange = { viewModel.updateRegister { copy(phone = it) } },
                        label = { Text("Phone") },
                        leadingIcon = { Icon(Icons.Default.Phone, contentDescription = null) },
                        singleLine = true,
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Phone),
                        modifier = Modifier.fillMaxWidth(),
                    )
                    OutlinedTextField(
                        value = state.countryCode,
                        onValueChange = { viewModel.updateRegister { copy(countryCode = it.uppercase()) } },
                        label = { Text("Country code") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
                1 -> {
                    Text(
                        "Enter the verification code sent to your phone (OTP placeholder for dev).",
                        style = MaterialTheme.typography.bodyMedium,
                    )
                    OutlinedTextField(
                        value = state.otpCode,
                        onValueChange = { viewModel.updateRegister { copy(otpCode = it) } },
                        label = { Text("OTP code") },
                        singleLine = true,
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
                else -> {
                    OutlinedTextField(
                        value = state.legalName,
                        onValueChange = { viewModel.updateRegister { copy(legalName = it) } },
                        label = { Text("Legal name") },
                        modifier = Modifier.fillMaxWidth(),
                    )
                    OutlinedTextField(
                        value = state.contactName,
                        onValueChange = { viewModel.updateRegister { copy(contactName = it) } },
                        label = { Text("Contact name") },
                        modifier = Modifier.fillMaxWidth(),
                    )
                    OutlinedTextField(
                        value = state.email,
                        onValueChange = { viewModel.updateRegister { copy(email = it) } },
                        label = { Text("Email") },
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Email),
                        modifier = Modifier.fillMaxWidth(),
                    )
                    OutlinedTextField(
                        value = state.password,
                        onValueChange = { viewModel.updateRegister { copy(password = it) } },
                        label = { Text("Password") },
                        visualTransformation = PasswordVisualTransformation(),
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
            }
            state.error?.let {
                PegasusRuntimeBanner(
                    tone = PegasusRuntimeTone.Warning,
                    message = it,
                    modifier = Modifier.fillMaxWidth(),
                )
            }
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                TextButton(
                    onClick = { if (state.step == 0) onBack() else viewModel.prevRegisterStep() },
                    enabled = !state.loading,
                ) { Text(if (state.step == 0) "Cancel" else "Back") }
                Button(
                    onClick = {
                        if (state.step < 2) viewModel.nextRegisterStep()
                        else viewModel.submitRegister(onRegistered)
                    },
                    enabled = !state.loading,
                ) {
                    if (state.loading) {
                        CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
                    } else {
                        Text(if (state.step < 2) "Continue" else "Create supplier")
                    }
                }
            }
        }
    }
}
