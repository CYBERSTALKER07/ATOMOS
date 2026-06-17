package com.pegasusx.warehouse

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.material3.windowsizeclass.ExperimentalMaterial3WindowSizeClassApi
import androidx.compose.material3.windowsizeclass.calculateWindowSizeClass
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import com.pegasusx.warehouse.data.remote.TokenHolder
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.data.remote.WarehouseOperationsRepository
import com.pegasusx.warehouse.data.remote.WarehouseRealtimeClient
import com.pegasusx.warehouse.data.remote.WarehouseRealtimeSignals
import com.pegasusx.warehouse.data.remote.reconcileWarehouseSession
import com.pegasusx.warehouse.ui.navigation.WarehouseNavigation
import com.pegasusx.warehouse.ui.theme.PegasusWarehouseTheme
import dagger.hilt.android.AndroidEntryPoint
import javax.inject.Inject
import kotlinx.coroutines.launch
import androidx.lifecycle.lifecycleScope
import org.maplibre.android.MapLibre

@AndroidEntryPoint
class MainActivity : ComponentActivity() {

    @Inject lateinit var warehouseApi: WarehouseApi
    @Inject lateinit var warehouseOpsRepository: WarehouseOperationsRepository
    @Inject lateinit var warehouseRealtimeClient: WarehouseRealtimeClient
    @Inject lateinit var realtimeSignals: WarehouseRealtimeSignals

    @OptIn(ExperimentalMaterial3WindowSizeClassApi::class)
    override fun onCreate(savedInstanceState: Bundle?) {
        installSplashScreen()
        super.onCreate(savedInstanceState)
        MapLibre.getInstance(this)
        enableEdgeToEdge()
        connectRealtimeIfAuthenticated()
        setContent {
            val windowSizeClass = calculateWindowSizeClass(this)
            PegasusWarehouseTheme {
                WarehouseNavigation(
                    api = warehouseApi,
                    opsRepository = warehouseOpsRepository,
                    realtimeSignals = realtimeSignals,
                    windowSizeClass = windowSizeClass,
                    onAuthenticated = { connectRealtimeIfAuthenticated() },
                )
            }
        }
    }

    override fun onDestroy() {
        warehouseRealtimeClient.dispose()
        super.onDestroy()
    }

    private fun connectRealtimeIfAuthenticated() {
        if (!TokenHolder.isLoggedIn) return
        warehouseRealtimeClient.connect(
            onStateChange = {},
            onEvent = { event ->
                if (!event.type.startsWith("SYSTEM")) {
                    realtimeSignals.bump()
                }
            },
            onReconnect = {
                lifecycleScope.launch {
                    reconcileWarehouseSession(warehouseApi, this@MainActivity)
                    realtimeSignals.bumpReconnect()
                }
            },
        )
    }
}
