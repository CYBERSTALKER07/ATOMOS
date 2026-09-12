import os
import json
import subprocess
from PIL import Image

def generate_ios_icons(source_image, output_dir):
    os.makedirs(output_dir, exist_ok=True)
    
    sizes = [
        ("20x20", [1, 2, 3]),
        ("29x29", [1, 2, 3]),
        ("40x40", [1, 2, 3]),
        ("60x60", [2, 3]),
        ("76x76", [1, 2]),
        ("83.5x83.5", [2]),
        ("1024x1024", [1])
    ]
    
    contents = {"images": [], "info": {"author": "xcode", "version": 1}}
    
    img = Image.open(source_image).convert("RGBA")
    
    for size_str, scales in sizes:
        base_size = float(size_str.split('x')[0])
        for scale in scales:
            actual_size = int(base_size * scale)
            filename = f"Icon-{size_str}@{scale}x.png"
            if scale == 1:
                filename = f"Icon-{size_str}.png"
            
            resized = img.resize((actual_size, actual_size), Image.Resampling.LANCZOS)
            resized.save(os.path.join(output_dir, filename))
            
            contents["images"].append({
                "size": size_str,
                "idiom": "universal",
                "filename": filename,
                "scale": f"{scale}x"
            })
            
    with open(os.path.join(output_dir, "Contents.json"), "w") as f:
        json.dump(contents, f, indent=2)


def generate_android_icons(source_image, res_dir):
    os.makedirs(res_dir, exist_ok=True)
    
    mipmaps = {
        "mipmap-mdpi": 48,
        "mipmap-hdpi": 72,
        "mipmap-xhdpi": 96,
        "mipmap-xxhdpi": 144,
        "mipmap-xxxhdpi": 192
    }
    
    img = Image.open(source_image).convert("RGBA")
    
    for folder, size in mipmaps.items():
        folder_path = os.path.join(res_dir, folder)
        os.makedirs(folder_path, exist_ok=True)
        
        resized = img.resize((size, size), Image.Resampling.LANCZOS)
        resized.save(os.path.join(folder_path, "ic_launcher.png"))
        
        # for round
        resized.save(os.path.join(folder_path, "ic_launcher_round.png"))


def generate_tauri_icons(source_image, tauri_dir):
    os.makedirs(os.path.join(tauri_dir, "icons"), exist_ok=True)
    icon_path = os.path.join(tauri_dir, "app-icon.png")
    img = Image.open(source_image).convert("RGBA")
    img.resize((1024, 1024), Image.Resampling.LANCZOS).save(icon_path)
    
    print(f"Running tauri icon in {tauri_dir}")
    try:
        subprocess.run(["npx", "@tauri-apps/cli", "icon", "app-icon.png"], cwd=tauri_dir, check=True)
    except Exception as e:
        print("Tauri icon generation failed:", e)


if __name__ == "__main__":
    import sys
    if len(sys.argv) < 2:
        print("Usage: python generate_icons.py <source_image>")
        sys.exit(1)
        
    source = sys.argv[1]
    
    ios_apps = [
        "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/factory-app-ios/FactoryApp/Assets.xcassets/AppIcon.appiconset",
        "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/driver-app-ios/driverappios/driverappios/Assets.xcassets/AppIcon.appiconset",
        "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/warehouse-app-ios/WarehouseApp/Assets.xcassets/AppIcon.appiconset",
        "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/payload-app-ios/payload-app-ios/Assets.xcassets/AppIcon.appiconset",
        "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/retailer-app-ios/retailerapp/retailerapp/Assets.xcassets/AppIcon.appiconset",
        "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/supplier-app-ios/SupplierApp/Assets.xcassets/AppIcon.appiconset"
    ]
    
    android_apps = [
        "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/factory-app-android/app/src/main/res",
        "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/driver-app-android/app/src/main/res",
        "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/warehouse-app-android/app/src/main/res",
        "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/retailer-app-android/app/src/main/res",
        "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/supplier-app-android/app/src/main/res",
        "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/payload-app-android/app/src/main/res"
    ]
    
    tauri_apps = [
        "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/retailer-app-desktop/src-tauri",
        "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/supplier-app-desktop/src-tauri"
    ]
    
    for ios_app in ios_apps:
        print(f"Generating iOS icons for {ios_app}")
        generate_ios_icons(source, ios_app)
        
    for android_app in android_apps:
        print(f"Generating Android icons for {android_app}")
        generate_android_icons(source, android_app)
        
    for tauri_app in tauri_apps:
        print(f"Generating Tauri icons for {tauri_app}")
        if os.path.exists(tauri_app):
            generate_tauri_icons(source, tauri_app)
        else:
            print(f"Tauri app not found: {tauri_app}")
    
    print("Done!")
