package com.pegasusx.supplier

import android.app.Application
import com.pegasusx.supplier.data.push.DeviceTokenRegistrar
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.data.remote.TokenHolder
import com.pegasusx.supplier.network.NetworkLifecycleEngine
import dagger.hilt.android.HiltAndroidApp
import javax.inject.Inject

@HiltAndroidApp
class PegasusXSupplierApp : Application() {
    @Inject lateinit var api: SupplierApi

    override fun onCreate() {
        super.onCreate()
        TokenHolder.init(this)
        NetworkLifecycleEngine.shared.initialize(this)
        DeviceTokenRegistrar.uploadBestEffort(api)
    }
}
