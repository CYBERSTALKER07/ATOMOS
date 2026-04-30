package com.pegasus.driver

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import com.pegasus.driver.data.remote.DriverApi
import com.pegasus.driver.ui.navigation.DriverNavigation
import com.pegasus.driver.ui.theme.LabDriverTheme
import dagger.hilt.android.AndroidEntryPoint
import javax.inject.Inject

@AndroidEntryPoint
class MainActivity : ComponentActivity() {

    @Inject lateinit var driverApi: DriverApi

    override fun onCreate(savedInstanceState: Bundle?) {
        installSplashScreen()
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            LabDriverTheme {
                DriverNavigation(api = driverApi)
            }
        }
    }
}
