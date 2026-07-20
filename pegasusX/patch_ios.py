import os
import glob

# Find all AutoUpdater.swift files
swift_files = glob.glob('apps/*-ios/**/AutoUpdater.swift', recursive=True)

for filepath in swift_files:
    with open(filepath, 'r') as f:
        content = f.read()

    old_code = """        let encoded = manifestURL.absoluteString.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed)
            ?? manifestURL.absoluteString
        guard let itms = URL(string: "itms-services://?action=download-manifest&url=\(encoded)") else {"""
        
    new_code = """        var components = URLComponents(string: "itms-services://")
        components?.queryItems = [
            URLQueryItem(name: "action", value: "download-manifest"),
            URLQueryItem(name: "url", value: manifestURL.absoluteString)
        ]
        guard let itms = components?.url else {"""
        
    if old_code in content:
        content = content.replace(old_code, new_code)
        with open(filepath, 'w') as f:
            f.write(content)
        print(f"Patched {filepath}")
    else:
        print(f"Could not find exact block in {filepath}")

