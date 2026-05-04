package com.pegasus.driver.services

import android.Manifest
import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.Service
import android.content.Intent
import android.content.IntentFilter
import android.content.pm.PackageManager
import android.content.pm.ServiceInfo
import android.os.BatteryManager
import android.os.Build
import android.os.IBinder
import android.os.PowerManager
import android.util.Log
import androidx.core.app.NotificationCompat
import androidx.core.content.ContextCompat
import com.google.android.gms.location.FusedLocationProviderClient
import com.google.android.gms.location.LocationCallback
import com.google.android.gms.location.LocationRequest
import com.google.android.gms.location.LocationResult
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import com.pegasus.driver.BuildConfig
import com.pegasus.driver.R
import com.pegasus.driver.data.local.OrderDao
import com.pegasus.driver.data.model.OrderState
import com.pegasus.driver.data.model.TelemetryPayload
import com.pegasus.driver.data.remote.DriverApi
import com.pegasus.driver.data.remote.TelemetrySocket
import com.pegasus.driver.data.remote.TokenHolder
import com.pegasus.driver.util.Haversine
import dagger.hilt.android.AndroidEntryPoint
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import javax.inject.Inject

@AndroidEntryPoint
class TelemetryService : Service() {

    companion object {
        const val CHANNEL_ID = "telemetry_channel"
        const val NOTIFICATION_ID = 1001
        const val ACTION_START = "com.pegasus.driver.START_TELEMETRY"
        const val ACTION_STOP = "com.pegasus.driver.STOP_TELEMETRY"
        private const val TAG = "TelemetryService"
        private const val WAKELOCK_TAG = "LabDriver::TelemetryWakelock"
    }

    @Inject lateinit var telemetrySocket: TelemetrySocket
    @Inject lateinit var api: DriverApi
    @Inject lateinit var orderDao: OrderDao

    private lateinit var fusedClient: FusedLocationProviderClient
    private var wakeLock: PowerManager.WakeLock? = null
    private var locationCallback: LocationCallback? = null

    // V.O.I.D. Adaptive Transmission Protocol
    private var lastSentLocation: android.location.Location? = null
    private var lastSentTimeMs: Long = 0

    private val serviceScope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private val arrivedIds = mutableSetOf<String>()

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onCreate() {
        super.onCreate()
        fusedClient = LocationServices.getFusedLocationProviderClient(this)
        createNotificationChannel()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_STOP -> {
                stopTracking()
                stopForeground(STOP_FOREGROUND_REMOVE)
                stopSelf()
            }
            else -> {
                val notification = buildNotification()
                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.UPSIDE_DOWN_CAKE) {
                    startForeground(NOTIFICATION_ID, notification, ServiceInfo.FOREGROUND_SERVICE_TYPE_LOCATION)
                } else {
                    startForeground(NOTIFICATION_ID, notification)
                }
                startTracking()
            }
        }
        return START_STICKY
    }

    override fun onDestroy() {
        stopTracking()
        super.onDestroy()
    }

    private fun startTracking() {
        // Acquire partial wakelock
        val pm = getSystemService(POWER_SERVICE) as PowerManager
        wakeLock = pm.newWakeLock(PowerManager.PARTIAL_WAKE_LOCK, WAKELOCK_TAG).apply {
            acquire() // Released explicitly in stopTracking()
        }

        // Connect WebSocket
        val token = TokenHolder.token
        if (token != null) {
            telemetrySocket.connect(BuildConfig.API_BASE_URL, token)
        } else {
            Log.w(TAG, "No auth token — WebSocket skipped")
        }

        // Start location updates — check foreground + background permissions
        if (ContextCompat.checkSelfPermission(this, Manifest.permission.ACCESS_FINE_LOCATION)
            != PackageManager.PERMISSION_GRANTED
        ) {
            Log.e(TAG, "Missing ACCESS_FINE_LOCATION permission")
            stopSelf()
            return
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q &&
            ContextCompat.checkSelfPermission(this, Manifest.permission.ACCESS_BACKGROUND_LOCATION)
            != PackageManager.PERMISSION_GRANTED
        ) {
            Log.w(TAG, "Missing ACCESS_BACKGROUND_LOCATION — tracking may be suspended in background")
        }

        // Adaptive GPS: use balanced accuracy when battery is low
        val priority = if (getBatteryPercent() < 20) {
            Log.d(TAG, "Battery low — using BALANCED_POWER_ACCURACY")
            Priority.PRIORITY_BALANCED_POWER_ACCURACY
        } else {
            Priority.PRIORITY_HIGH_ACCURACY
        }

        val locationRequest = LocationRequest.Builder(priority, 10_000L)
            .setMinUpdateIntervalMillis(5_000L)
            .setWaitForAccurateLocation(false)
            .build()

        locationCallback = object : LocationCallback() {
            override fun onLocationResult(result: LocationResult) {
                val location = result.lastLocation ?: return
                val driverId = TokenHolder.userId ?: return

                // V.O.I.D. Adaptive Transmission Protocol (Client-Side Filter)
                val timeSinceLastMs = System.currentTimeMillis() - lastSentTimeMs
                var shouldTransmit = false

                if (timeSinceLastMs > 15000) {
                    // Heartbeat: Always send if it's been more than 15 seconds
                    shouldTransmit = true
                } else if (lastSentLocation != null) {
                    // Dead Reckoning: Check deviation
                    val distanceDeviation = location.distanceTo(lastSentLocation!!)
                    val bearingDeviation = Math.abs(location.bearing - lastSentLocation!!.bearing)

                    // Thresholds: Deviated by > 20 meters OR turned > 15 degrees
                    if (distanceDeviation > 20f || bearingDeviation > 15f) {
                        shouldTransmit = true
                    }
                } else {
                    // Send first point immediately
                    shouldTransmit = true
                }

                if (shouldTransmit) {
                    val payload = TelemetryPayload(
                        driverId = driverId,
                        latitude = location.latitude,
                        longitude = location.longitude,
                        timestamp = System.currentTimeMillis(),
                        speed = location.speed,
                        bearing = location.bearing
                    )

                    val sent = telemetrySocket.send(payload)
                    if (!sent) {
                        Log.w(TAG, "Telemetry send failed — socket may be disconnected")
                    } else {
                        lastSentLocation = location
                        lastSentTimeMs = System.currentTimeMillis()
                    }
                }

                // Auto-ARRIVED: check proximity to IN_TRANSIT order destinations
                checkAutoArrive(location.latitude, location.longitude)
            }
        }

        fusedClient.requestLocationUpdates(locationRequest, locationCallback!!, mainLooper)
        Log.d(TAG, "Telemetry tracking started")
    }

    private fun checkAutoArrive(lat: Double, lng: Double) {
        serviceScope.launch {
            try {
                val orders = orderDao.getByState(OrderState.IN_TRANSIT.name)
                for (order in orders) {
                    if (order.id in arrivedIds) continue
                    val destLat = order.latitude ?: continue
                    val destLng = order.longitude ?: continue
                    val dist = Haversine.distanceMeters(lat, lng, destLat, destLng)
                    if (dist <= 100.0) {
                        arrivedIds.add(order.id)
                        try {
                            api.markArrived(
                                body = mapOf("order_id" to order.id),
                                idempotencyKey = "driver-mark-arrived-${order.id}"
                            )
                            orderDao.updateState(order.id, OrderState.ARRIVED.name, System.currentTimeMillis().toString())
                            Log.d(TAG, "Auto-ARRIVED: ${order.id} (${dist.toInt()}m)")
                        } catch (e: Exception) {
                            Log.w(TAG, "Auto-ARRIVED failed for ${order.id}", e)
                            arrivedIds.remove(order.id)
                        }
                    }
                }
            } catch (e: Exception) {
                Log.w(TAG, "checkAutoArrive failed", e)
            }
        }
    }

    private fun stopTracking() {
        serviceScope.cancel()
        locationCallback?.let { fusedClient.removeLocationUpdates(it) }
        locationCallback = null
        telemetrySocket.disconnect()
        wakeLock?.let {
            if (it.isHeld) it.release()
        }
        wakeLock = null
        Log.d(TAG, "Telemetry tracking stopped")
    }

    private fun getBatteryPercent(): Int {
        val batteryStatus = registerReceiver(null, IntentFilter(Intent.ACTION_BATTERY_CHANGED))
        val level = batteryStatus?.getIntExtra(BatteryManager.EXTRA_LEVEL, -1) ?: -1
        val scale = batteryStatus?.getIntExtra(BatteryManager.EXTRA_SCALE, -1) ?: -1
        return if (level >= 0 && scale > 0) (level * 100) / scale else 100
    }

    private fun createNotificationChannel() {
        val channel = NotificationChannel(
            CHANNEL_ID,
            getString(R.string.notification_channel_telemetry),
            NotificationManager.IMPORTANCE_LOW
        ).apply {
            description = "Active transit telemetry"
            setShowBadge(false)
        }
        val manager = getSystemService(NotificationManager::class.java)
        manager.createNotificationChannel(channel)
    }

    private fun buildNotification(): Notification =
        NotificationCompat.Builder(this, CHANNEL_ID)
            .setContentTitle(getString(R.string.notification_telemetry_title))
            .setContentText(getString(R.string.notification_telemetry_text))
            .setSmallIcon(android.R.drawable.ic_menu_mylocation)
            .setOngoing(true)
            .setCategory(NotificationCompat.CATEGORY_SERVICE)
            .setForegroundServiceBehavior(NotificationCompat.FOREGROUND_SERVICE_IMMEDIATE)
            .build()
}
