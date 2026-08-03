package com.pegasusx.driver.ui.screens.manifest.components

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.driver.ui.components.DriverStateKind
import com.pegasusx.driver.ui.components.DriverStatePane

@Composable
fun ManifestEmptyView() {
    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.surface)
            .padding(horizontal = 24.dp),
        contentAlignment = Alignment.Center
    ) {
        DriverStatePane(
            kind = DriverStateKind.Route,
            headline = "No upcoming rides",
            body = "Pull to refresh or check back when dispatch assigns your route.",
            usePegasusCard = true,
        )
    }
}
