package com.pegasusx.driver

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import com.pegasusx.driver.data.remote.DriverApi
import com.pegasusx.driver.data.remote.DriverWebSocket
import com.pegasusx.driver.ui.navigation.DriverNavigation
import com.pegasusx.driver.ui.theme.PegasusDriverTheme
import dagger.hilt.android.AndroidEntryPoint
import javax.inject.Inject

@AndroidEntryPoint
class MainActivity : ComponentActivity() {

    @Inject lateinit var driverApi: DriverApi
    @Inject lateinit var driverWebSocket: DriverWebSocket

    override fun onCreate(savedInstanceState: Bundle?) {
        installSplashScreen()
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            PegasusDriverTheme {
                DriverNavigation(api = driverApi, driverWebSocket = driverWebSocket)
            }
        }
    }
}
