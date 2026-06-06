package com.pegasusx.supplier

import android.app.Application
import com.pegasusx.supplier.data.remote.TokenHolder
import dagger.hilt.android.HiltAndroidApp

@HiltAndroidApp
class PegasusXSupplierApp : Application() {
    override fun onCreate() {
        super.onCreate()
        TokenHolder.init(this)
    }
}
