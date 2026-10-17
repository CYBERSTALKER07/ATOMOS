.PHONY: generate build run snapshot

generate:
	xcodegen generate

build: generate
	xcodebuild -project PegasusCartograph.xcodeproj -scheme PegasusCartograph \
	  -configuration Debug -derivedDataPath build CODE_SIGNING_ALLOWED=NO \
	  -destination 'platform=macOS' build

run: build
	open "build/Build/Products/Debug/Pegasus Cartograph.app"

snapshot: build
	CARTOGRAPH_SNAPSHOT="$(CURDIR)/snapshot.png" CARTOGRAPH_SNAPSHOT_EXIT=1 \
	  "build/Build/Products/Debug/Pegasus Cartograph.app/Contents/MacOS/Pegasus Cartograph"
