package com.pegasusx.warehouse

import android.app.Application
import com.pegasusx.warehouse.data.push.DeviceTokenRegistrar
import com.pegasusx.warehouse.data.remote.FirebaseAuthHelper
import com.pegasusx.warehouse.data.remote.TokenHolder
import com.pegasusx.warehouse.data.remote.WarehouseApi
import dagger.hilt.android.HiltAndroidApp
import javax.inject.Inject

@HiltAndroidApp
class PegasusWarehouseApp : Application() {
    @Inject lateinit var api: WarehouseApi

    override fun onCreate() {
        super.onCreate()
        TokenHolder.init(this)
        FirebaseAuthHelper.init(this)
        DeviceTokenRegistrar.uploadBestEffort(api)
    }
}
