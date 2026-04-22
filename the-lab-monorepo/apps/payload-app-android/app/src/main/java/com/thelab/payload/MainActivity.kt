package com.thelab.payload

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.result.contract.ActivityResultContracts
import androidx.core.content.ContextCompat
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import com.thelab.payload.services.NotificationBus
import com.thelab.payload.ui.PayloadRoot
import com.thelab.payload.ui.theme.LabPayloadTheme
import dagger.hilt.android.AndroidEntryPoint
import javax.inject.Inject

/**
 * Single-Activity host. Composition starts at [PayloadRoot] which gates
 * authenticated vs unauthenticated routes off [com.thelab.payload.data.repository.AuthRepository].
 */
@AndroidEntryPoint
class MainActivity : ComponentActivity() {

    @Inject lateinit var notificationBus: NotificationBus

    private val requestNotificationPermission =
        registerForActivityResult(ActivityResultContracts.RequestPermission()) { /* no-op */ }

    override fun onCreate(savedInstanceState: Bundle?) {
        installSplashScreen()
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        maybeRequestNotificationPermission()
        handleIntentExtras(intent)
        setContent {
            LabPayloadTheme {
                PayloadRoot()
            }
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        handleIntentExtras(intent)
    }

    private fun handleIntentExtras(intent: Intent?) {
        val open = intent?.getBooleanExtra(NotificationBus.EXTRA_OPEN_NOTIFICATIONS, false) == true
        if (open) notificationBus.requestOpenPanel()
    }

    private fun maybeRequestNotificationPermission() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU) return
        val perm = Manifest.permission.POST_NOTIFICATIONS
        if (ContextCompat.checkSelfPermission(this, perm) == PackageManager.PERMISSION_GRANTED) return
        requestNotificationPermission.launch(perm)
    }
}

