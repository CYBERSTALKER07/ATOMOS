package com.thelab.driver

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import com.thelab.driver.data.remote.DriverApi
import com.thelab.driver.ui.navigation.DriverNavigation
import com.thelab.driver.ui.theme.LabDriverTheme
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
