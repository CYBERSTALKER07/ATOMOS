# PegasusX Cartograph

Native macOS isometric map of **`pegasusX/` on `origin/main` (`fbfd134e`)**.

Not a product role. Not status from docs. Buildings and traces cite `file:line` opened from that commit.

```
cd pegasusX/artifacts/cartograph
xcodegen generate
xcodebuild -project PegasusCartograph.xcodeproj -scheme PegasusCartograph \
  -configuration Debug -derivedDataPath build CODE_SIGNING_ALLOWED=NO
open build/Build/Products/Debug/Pegasus\ Cartograph.app
```

Click a building or press `1`–`9` to isolate a trace. `esc` clears the trace filter.
