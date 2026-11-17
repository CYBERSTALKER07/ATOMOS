.PHONY: all generate build test clean

all: generate build test

generate:
	xcodegen generate

build: generate
	xcodebuild -project PegasusControlTower.xcodeproj \
		-target PegasusControlTower \
		build CODE_SIGNING_ALLOWED=NO

test: generate
	xcodebuild -project PegasusControlTower.xcodeproj \
		-target PegasusControlTowerTests \
		build CODE_SIGNING_ALLOWED=NO
	xcrun xctest build/Debug/PegasusControlTowerTests.xctest

clean:
	rm -rf PegasusControlTower.xcodeproj build
