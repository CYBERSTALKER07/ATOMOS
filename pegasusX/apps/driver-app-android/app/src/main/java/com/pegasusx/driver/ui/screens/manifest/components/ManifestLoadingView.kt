package com.pegasusx.driver.ui.screens.manifest.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.driver.ui.components.DriverLoadingState
import com.pegasusx.driver.R

@Composable
fun ManifestLoadingView() {
    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.surface)
            .padding(horizontal = 24.dp),
        contentAlignment = Alignment.Center
    ) {
        DriverLoadingState(
            title = stringResource(R.string.mobile_driver_ui_loading_routes),
            body = "Checking manifest state, sequence, and delivery assignments.",
            shimmerLines = true,
        )
    }
}
