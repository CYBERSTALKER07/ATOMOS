package com.pegasusx.warehouse

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.data.remote.WarehouseOperationsRepository
import com.pegasusx.warehouse.ui.navigation.WarehouseNavigation
import com.pegasusx.warehouse.ui.theme.PegasusWarehouseTheme
import dagger.hilt.android.AndroidEntryPoint
import javax.inject.Inject

@AndroidEntryPoint
class MainActivity : ComponentActivity() {

    @Inject lateinit var warehouseApi: WarehouseApi
    @Inject lateinit var warehouseOpsRepository: WarehouseOperationsRepository

    override fun onCreate(savedInstanceState: Bundle?) {
        installSplashScreen()
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            PegasusWarehouseTheme {
                WarehouseNavigation(
                    api = warehouseApi,
                    opsRepository = warehouseOpsRepository,
                )
            }
        }
    }
}
