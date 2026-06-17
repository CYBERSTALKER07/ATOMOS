package com.pegasusx.warehouse

import android.app.Application
import com.pegasusx.warehouse.data.remote.FirebaseAuthHelper
import com.pegasusx.warehouse.data.remote.TokenHolder
import dagger.hilt.android.HiltAndroidApp

@HiltAndroidApp
class PegasusWarehouseApp : Application() {
    override fun onCreate() {
        super.onCreate()
        TokenHolder.init(this)
        FirebaseAuthHelper.init(this)
    }
}
