package com.thelab.factory

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import com.thelab.factory.data.remote.FactoryApi
import com.thelab.factory.ui.navigation.FactoryNavigation
import com.thelab.factory.ui.theme.LabFactoryTheme
import dagger.hilt.android.AndroidEntryPoint
import javax.inject.Inject

@AndroidEntryPoint
class MainActivity : ComponentActivity() {

    @Inject lateinit var factoryApi: FactoryApi

    override fun onCreate(savedInstanceState: Bundle?) {
        installSplashScreen()
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            LabFactoryTheme {
                FactoryNavigation(api = factoryApi)
            }
        }
    }
}
