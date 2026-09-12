import re

with open("apps/backend-go/payload/ship_units.go", "r") as f:
    content = f.read()

pattern = re.compile(r'import \(\n\t"context"\n\t"encoding/binary"\n\t"encoding/json"\n\t"fmt"\n\t"hash/fnv"\n\t"os"\n\t"strings"\n\t"time"\n\n\t"cloud\.google\.com/go/spanner"\n\t"github\.com/google/uuid"\n\t"github\.com/pegasusx/pegasusx/apps/backend-go/gs1"\n\t"google\.golang\.org/api/iterator"\n\)')

replacement = r"""import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/gs1"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)"""
content = pattern.sub(replacement, content)

with open("apps/backend-go/payload/ship_units.go", "w") as f:
    f.write(content)
