package com.thelab.retailer.ui.screens.auth

import android.Manifest
import android.annotation.SuppressLint
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import com.thelab.retailer.ui.theme.SquircleShape
import com.thelab.retailer.ui.theme.PillShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Map
import androidx.compose.material.icons.filled.MyLocation
import androidx.compose.material.icons.filled.Storefront
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableDoubleStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import com.google.accompanist.permissions.ExperimentalPermissionsApi
import com.google.accompanist.permissions.isGranted
import com.google.accompanist.permissions.rememberPermissionState
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import com.google.android.gms.tasks.CancellationTokenSource
import com.thelab.retailer.ui.theme.StatusRed
import kotlinx.coroutines.launch
import kotlinx.coroutines.tasks.await

@OptIn(ExperimentalPermissionsApi::class)
@SuppressLint("MissingPermission")
@Composable
fun AuthScreen(
    viewModel: AuthViewModel = hiltViewModel(),
    onAuthenticated: () -> Unit,
) {
    val state by viewModel.uiState.collectAsState()
    var isLoginMode by rememberSaveable { mutableStateOf(true) }
    var showMapPicker by rememberSaveable { mutableStateOf(false) }

    // Registration form fields
    var phone by rememberSaveable { mutableStateOf("+998") }
    var password by rememberSaveable { mutableStateOf("") }
    var storeName by rememberSaveable { mutableStateOf("") }
    var ownerName by rememberSaveable { mutableStateOf("") }
    var addressText by rememberSaveable { mutableStateOf("") }
    var taxId by rememberSaveable { mutableStateOf("") }
    var receivingWindowOpen by rememberSaveable { mutableStateOf("") }
    var receivingWindowClose by rememberSaveable { mutableStateOf("") }
    var selectedAccessType by rememberSaveable { mutableStateOf("") }
    var ceilingHeightText by rememberSaveable { mutableStateOf("") }

    // Location state
    var latitude by rememberSaveable { mutableDoubleStateOf(0.0) }
    var longitude by rememberSaveable { mutableDoubleStateOf(0.0) }
    var locationLabel by rememberSaveable { mutableStateOf("") }

    // GPS share state
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val locationPermission = rememberPermissionState(Manifest.permission.ACCESS_FINE_LOCATION)
    val fusedClient = remember { LocationServices.getFusedLocationProviderClient(context) }
    var locating by remember { mutableStateOf(false) }

    LaunchedEffect(state.isAuthenticated) {
        if (state.isAuthenticated) onAuthenticated()
    }

    val textFieldColors = OutlinedTextFieldDefaults.colors(
        focusedBorderColor = Color.Black,
        unfocusedBorderColor = Color.Black.copy(alpha = 0.2f),
        cursorColor = Color.Black,
        focusedLabelColor = Color.Black,
    )

    if (showMapPicker) {
        LocationPickerScreen(
            initialLat = if (latitude != 0.0) latitude else 41.2995,
            initialLng = if (longitude != 0.0) longitude else 69.2401,
            onConfirm = { picked ->
                latitude = picked.latitude
                longitude = picked.longitude
                locationLabel = picked.displayText
                showMapPicker = false
            },
            onBack = { showMapPicker = false },
        )
        return
    }

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(Color.White)
            .imePadding(),
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 24.dp, vertical = 48.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) {
            Spacer(Modifier.height(40.dp))

            // Logo
            Box(
                modifier = Modifier
                    .size(72.dp)
                    .background(Color.Black, CircleShape),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    imageVector = Icons.Default.Storefront,
                    contentDescription = null,
                    tint = Color.White,
                    modifier = Modifier.size(32.dp),
                )
            }

            Spacer(Modifier.height(16.dp))

            Text(
                text = "The Lab",
                style = MaterialTheme.typography.headlineLarge.copy(fontWeight = FontWeight.Bold),
                color = Color.Black,
            )
            Text(
                text = "Retailer Portal",
                style = MaterialTheme.typography.bodyMedium,
                color = Color.Black.copy(alpha = 0.5f),
            )

            Spacer(Modifier.height(40.dp))

            // Phone field (always visible)
            OutlinedTextField(
                value = phone,
                onValueChange = { if (it.startsWith("+998") || it == "+99" || it == "+9" || it == "+") phone = it else if (it.startsWith("+998")) phone = it },
                label = { Text("Phone Number") },
                placeholder = { Text("+998 XX XXX XX XX") },
                singleLine = true,
                keyboardOptions = KeyboardOptions(
                    keyboardType = KeyboardType.Phone,
                    imeAction = ImeAction.Next,
                ),
                colors = textFieldColors,
                modifier = Modifier.fillMaxWidth(),
                shape = SquircleShape,
            )

            Spacer(Modifier.height(12.dp))

            // Password
            OutlinedTextField(
                value = password,
                onValueChange = { password = it },
                label = { Text("Password") },
                singleLine = true,
                visualTransformation = PasswordVisualTransformation(),
                keyboardOptions = KeyboardOptions(
                    keyboardType = KeyboardType.Password,
                    imeAction = if (isLoginMode) ImeAction.Done else ImeAction.Next,
                ),
                colors = textFieldColors,
                modifier = Modifier.fillMaxWidth(),
                shape = SquircleShape,
            )

            // ── Registration-only fields ──
            AnimatedVisibility(
                visible = !isLoginMode,
                enter = fadeIn() + slideInVertically(),
                exit = fadeOut() + slideOutVertically(),
            ) {
                Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                    Spacer(Modifier.height(12.dp))

                    OutlinedTextField(
                        value = storeName,
                        onValueChange = { storeName = it },
                        label = { Text("Store Name") },
                        singleLine = true,
                        colors = textFieldColors,
                        modifier = Modifier.fillMaxWidth(),
                        shape = SquircleShape,
                    )

                    OutlinedTextField(
                        value = ownerName,
                        onValueChange = { ownerName = it },
                        label = { Text("Owner Name") },
                        singleLine = true,
                        colors = textFieldColors,
                        modifier = Modifier.fillMaxWidth(),
                        shape = SquircleShape,
                    )

                    OutlinedTextField(
                        value = addressText,
                        onValueChange = { addressText = it },
                        label = { Text("Store Address") },
                        singleLine = true,
                        colors = textFieldColors,
                        modifier = Modifier.fillMaxWidth(),
                        shape = SquircleShape,
                    )

                    // ── Location picker buttons ──
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        OutlinedButton(
                            onClick = { showMapPicker = true },
                            modifier = Modifier.weight(1f).height(44.dp),
                            shape = SquircleShape,
                            colors = ButtonDefaults.outlinedButtonColors(contentColor = Color.Black),
                        ) {
                            Icon(Icons.Default.Map, contentDescription = null, modifier = Modifier.size(16.dp))
                            Spacer(Modifier.width(6.dp))
                            Text("Open Map", fontSize = 13.sp)
                        }
                        OutlinedButton(
                            onClick = {
                                if (!locationPermission.status.isGranted) {
                                    locationPermission.launchPermissionRequest()
                                    return@OutlinedButton
                                }
                                locating = true
                                scope.launch {
                                    try {
                                        val loc = fusedClient.getCurrentLocation(
                                            Priority.PRIORITY_HIGH_ACCURACY,
                                            CancellationTokenSource().token,
                                        ).await()
                                        if (loc != null) {
                                            latitude = loc.latitude
                                            longitude = loc.longitude
                                            locationLabel = "%.5f, %.5f".format(loc.latitude, loc.longitude)
                                        }
                                    } catch (_: Exception) {
                                        // Ignore
                                    } finally {
                                        locating = false
                                    }
                                }
                            },
                            modifier = Modifier.weight(1f).height(44.dp),
                            shape = SquircleShape,
                            colors = ButtonDefaults.outlinedButtonColors(contentColor = Color.Black),
                        ) {
                            if (locating) {
                                CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp, color = Color.Black)
                            } else {
                                Icon(Icons.Default.MyLocation, contentDescription = null, modifier = Modifier.size(16.dp))
                                Spacer(Modifier.width(6.dp))
                                Text("Use GPS", fontSize = 13.sp)
                            }
                        }
                    }

                    if (locationLabel.isNotEmpty()) {
                        Text(
                            text = "Selected: $locationLabel",
                            style = MaterialTheme.typography.bodySmall,
                            color = Color.Black.copy(alpha = 0.6f),
                            modifier = Modifier.padding(horizontal = 4.dp),
                        )
                    }

                    OutlinedTextField(
                        value = taxId,
                        onValueChange = { taxId = it },
                        label = { Text("Tax ID (INN)") },
                        singleLine = true,
                        colors = textFieldColors,
                        modifier = Modifier.fillMaxWidth(),
                        shape = SquircleShape,
                    )

                    // ── Logistics Details ──
                    Text(
                        text = "Receiving Window (optional)",
                        style = MaterialTheme.typography.labelSmall,
                        color = Color.Black.copy(alpha = 0.5f),
                        modifier = Modifier.padding(top = 4.dp),
                    )
                    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        OutlinedTextField(
                            value = receivingWindowOpen,
                            onValueChange = { receivingWindowOpen = it },
                            label = { Text("Opens HH:MM") },
                            placeholder = { Text("09:00") },
                            singleLine = true,
                            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Next),
                            colors = textFieldColors,
                            modifier = Modifier.weight(1f),
                            shape = SquircleShape,
                        )
                        OutlinedTextField(
                            value = receivingWindowClose,
                            onValueChange = { receivingWindowClose = it },
                            label = { Text("Closes HH:MM") },
                            placeholder = { Text("18:00") },
                            singleLine = true,
                            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Next),
                            colors = textFieldColors,
                            modifier = Modifier.weight(1f),
                            shape = SquircleShape,
                        )
                    }

                    Text(
                        text = "Loading Access Type (optional)",
                        style = MaterialTheme.typography.labelSmall,
                        color = Color.Black.copy(alpha = 0.5f),
                        modifier = Modifier.padding(top = 4.dp),
                    )
                    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        listOf(
                            "STREET_PARKING" to "Street",
                            "ALLEYWAY" to "Alley",
                            "LOADING_DOCK" to "Dock",
                        ).forEach { (value, label) ->
                            val isSelected = selectedAccessType == value
                            OutlinedButton(
                                onClick = { selectedAccessType = if (isSelected) "" else value },
                                modifier = Modifier.weight(1f).height(40.dp),
                                shape = SquircleShape,
                                colors = if (isSelected)
                                    ButtonDefaults.outlinedButtonColors(containerColor = Color.Black, contentColor = Color.White)
                                else
                                    ButtonDefaults.outlinedButtonColors(contentColor = Color.Black),
                                border = if (isSelected) null else ButtonDefaults.outlinedButtonBorder,
                            ) { Text(label, fontSize = 12.sp, fontWeight = FontWeight.SemiBold) }
                        }
                    }

                    OutlinedTextField(
                        value = ceilingHeightText,
                        onValueChange = { ceilingHeightText = it },
                        label = { Text("Storage Ceiling Height cm (optional)") },
                        placeholder = { Text("e.g. 300") },
                        singleLine = true,
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal, imeAction = ImeAction.Done),
                        colors = textFieldColors,
                        modifier = Modifier.fillMaxWidth(),
                        shape = SquircleShape,
                    )
                }
            }

            Spacer(Modifier.height(24.dp))

            // Main Action Button
            Button(
                onClick = {
                    if (isLoginMode) {
                        viewModel.login(phone, password)
                    } else {
                        viewModel.register(
                            phone = phone,
                            password = password,
                            storeName = storeName,
                            ownerName = ownerName,
                            addressText = addressText,
                            taxId = taxId,
                            latitude = latitude,
                            longitude = longitude,
                            receivingWindowOpen = receivingWindowOpen.takeIf { it.isNotBlank() },
                            receivingWindowClose = receivingWindowClose.takeIf { it.isNotBlank() },
                            accessType = selectedAccessType.takeIf { it.isNotBlank() },
                            storageCeilingHeightCM = ceilingHeightText.toDoubleOrNull(),
                        )
                    }
                },
                modifier = Modifier
                    .fillMaxWidth()
                    .height(52.dp),
                shape = PillShape,
                colors = ButtonDefaults.buttonColors(
                    containerColor = Color.Black,
                    contentColor = Color.White,
                ),
                enabled = !state.isLoading,
            ) {
                if (state.isLoading) {
                    CircularProgressIndicator(modifier = Modifier.size(24.dp), color = Color.White)
                } else {
                    Text(
                        text = if (isLoginMode) "Sign In" else "Create Account",
                        fontSize = 16.sp,
                        fontWeight = FontWeight.Bold,
                    )
                }
            }

            Spacer(Modifier.height(16.dp))

            // Error Message
            if (state.error != null) {
                Text(
                    text = state.error ?: "",
                    color = StatusRed,
                    style = MaterialTheme.typography.bodySmall,
                    textAlign = TextAlign.Center,
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(Modifier.height(16.dp))
            }

            // Toggle Mode
            TextButton(
                onClick = { isLoginMode = !isLoginMode },
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(
                    text = if (isLoginMode) "Don't have an account? Sign Up" else "Already have an account? Sign In",
                    color = Color.Black,
                    fontSize = 14.sp,
                )
            }
        }
    }
}
