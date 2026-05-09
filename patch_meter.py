import re

with open('pegasus/apps/backend-go/workers/utility_meter.go', 'r') as f:
    text = f.read()

# Replace multiple import blocks with one clean block
text = re.sub(r'import \([\s\S]*?\)', r'''import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"cloud.google.com/go/spanner"
	
	"backend-go/outbox"
	"backend-go/telemetry"
)''', text, count=1)

with open('pegasus/apps/backend-go/workers/utility_meter.go', 'w') as f:
    f.write(text)
