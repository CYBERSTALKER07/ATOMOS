import os

apps = {
    'retailer': {
        'desktop': 'retailer-app-desktop/app/(dashboard)',
        'android': 'retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens',
        'ios': 'retailer-app-ios/retailerapp/reatilerapp/Screens'
    },
    'warehouse': {
        'desktop': 'warehouse-portal/app',
        'android': 'warehouse-app-android/app/src/main/java/com/pegasus/warehouse/ui/screens',
        'ios': 'warehouse-app-ios/WarehouseApp/Views'
    },
    'factory': {
        'desktop': 'factory-portal/app',
        'android': 'factory-app-android/app/src/main/java/com/pegasus/factory/ui/screens',
        'ios': 'factory-app-ios/FactoryApp/Views'
    },
    'driver': {
        'android': 'driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens',
        'ios': 'driverappios/driverappios/Views'
    },
    'payloader': {
        'android': 'payload-app-android/app/src/main/java/com/pegasus/payloader/ui/screens',
        'ios': 'payload-app-ios/payload-app-ios/Views'
    }
}

for role, platforms in apps.items():
    print(f"Role: {role}")
    for platform, path in platforms.items():
        screens = []
        if os.path.exists(path):
            screens = [d for d in os.listdir(path) if os.path.isdir(os.path.join(path, d)) and not d.startswith('.')]
        print(f"  {platform}: {sorted(screens)}")
