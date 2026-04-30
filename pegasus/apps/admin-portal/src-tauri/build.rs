use std::path::Path;

fn main() {
    // ── C++ FFmpeg Evidence Compressor ──────────────────────────────────
    let cpp_source = Path::new("src-cpp/evidence_compressor.cpp");
    if cpp_source.exists() {
        cc::Build::new()
            .cpp(true)
            .file(cpp_source)
            .flag_if_supported("-std=c++17")
            .flag_if_supported("-O2")
            // Link FFmpeg libraries (must be installed on the build host)
            .flag_if_supported("-lavcodec")
            .flag_if_supported("-lavformat")
            .flag_if_supported("-lavutil")
            .flag_if_supported("-lswscale")
            .compile("libcompressor.a");

        // Tell cargo to re-run if the C++ source changes
        println!("cargo:rerun-if-changed=src-cpp/evidence_compressor.cpp");

        // Link FFmpeg system libraries
        #[cfg(target_os = "macos")]
        {
            println!("cargo:rustc-link-lib=framework=VideoToolbox");
            println!("cargo:rustc-link-lib=framework=CoreMedia");
            println!("cargo:rustc-link-lib=framework=CoreVideo");
        }
    }

    tauri_build::build();
}
