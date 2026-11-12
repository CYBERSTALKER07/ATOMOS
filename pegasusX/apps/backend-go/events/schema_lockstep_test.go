package events_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var strictSchemaFlag = flag.Bool("strict", false, "enforce strict event-to-payload schema lockstep and byte-for-byte gen-contracts parity")

var syncDirectiveRegex = regexp.MustCompile(`@Sync(?:\(([^)\n]+)\))?`)

type schemaDocument struct {
	Defs       map[string]map[string]any `json:"$defs"`
	Properties struct {
		Type struct {
			Enum []string `json:"enum"`
		} `json:"type"`
	} `json:"properties"`
	EventPayloadMap map[string]string `json:"x-event-payload-map"`
}

func TestEventsSchemaLockstep(t *testing.T) {
	schemaPath := filepath.Clean(filepath.Join("..", "..", "..", "contracts", "events.schema.json"))
	rawSchema, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read %s: %v", schemaPath, err)
	}

	var doc schemaDocument
	if err := json.Unmarshal(rawSchema, &doc); err != nil {
		t.Fatalf("unmarshal %s: %v", schemaPath, err)
	}
	if len(doc.Defs) == 0 {
		t.Fatalf("%s has empty $defs", schemaPath)
	}

	// Invariant 7: every payload object definition in $defs must enforce "additionalProperties": false.
	for defName, defObj := range doc.Defs {
		assertStrictAdditionalProperties(t, defName, defObj)
	}

	enumSet := make(map[string]struct{}, len(doc.Properties.Type.Enum))
	for _, ev := range doc.Properties.Type.Enum {
		enumSet[ev] = struct{}{}
	}

	syncedEvents := extractSyncedEventsFromSource(t, "events.go")
	if len(syncedEvents) == 0 {
		t.Fatal("expected @Sync event constants in events.go, found 0")
	}

	for name, val := range syncedEvents {
		if _, ok := enumSet[val]; !ok {
			t.Errorf("synced event %s (%q) missing from contracts/events.schema.json properties.type.enum", name, val)
		}
		payloadName, mapped := doc.EventPayloadMap[val]
		if !mapped || strings.TrimSpace(payloadName) == "" {
			if *strictSchemaFlag {
				t.Errorf("strict mode: synced event %s (%q) missing payload mapping in x-event-payload-map", name, val)
			}
			continue
		}
		if _, exists := doc.Defs[payloadName]; !exists {
			t.Errorf("event %s (%q) maps to payload %q which does not exist in $defs", name, val, payloadName)
		}
	}

	if *strictSchemaFlag {
		for ev, payloadName := range doc.EventPayloadMap {
			if strings.TrimSpace(payloadName) == "" {
				t.Errorf("strict mode: event %q has empty payload in x-event-payload-map", ev)
				continue
			}
			if _, exists := doc.Defs[payloadName]; !exists {
				t.Errorf("strict mode: event %q maps to missing $defs[%q]", ev, payloadName)
			}
		}

		tmpOut := filepath.Join(t.TempDir(), "events.schema.json")
		cmd := exec.Command(
			"go", "run", "../cmd/gen-contracts",
			"-source", ".",
			"-mode", "json-schema",
			"-schema-out", tmpOut,
			"-pretty=true",
			"-strict=true",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("gen-contracts -strict=true failed: %v\n%s", err, string(out))
		}
		generated, err := os.ReadFile(tmpOut)
		if err != nil {
			t.Fatalf("read generated schema: %v", err)
		}
		if !bytes.Equal(bytes.TrimSpace(rawSchema), bytes.TrimSpace(generated)) {
			t.Fatalf("contracts/events.schema.json drifted from events package; run make gen-contracts-gate")
		}
	}
}

func assertStrictAdditionalProperties(t *testing.T, path string, node map[string]any) {
	t.Helper()
	if typ, _ := node["type"].(string); typ == "object" {
		ap, exists := node["additionalProperties"]
		apBool, isBool := ap.(bool)
		if !exists || !isBool || apBool {
			t.Errorf("$defs.%s has type=object without additionalProperties=false (got %v)", path, ap)
		}
	}
	if props, ok := node["properties"].(map[string]any); ok {
		for propName, rawProp := range props {
			if propObj, ok := rawProp.(map[string]any); ok {
				assertStrictAdditionalProperties(t, path+"."+propName, propObj)
			}
		}
	}
	if items, ok := node["items"].(map[string]any); ok {
		assertStrictAdditionalProperties(t, path+"[]", items)
	}
}

func extractSyncedEventsFromSource(t *testing.T, path string) map[string]string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	result := make(map[string]string)
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}
		declSynced := hasSyncComment(genDecl.Doc)
		stickySynced := false
		for _, spec := range genDecl.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			if hasSyncComment(vs.Doc) || hasSyncComment(vs.Comment) {
				stickySynced = true
			}
			if !declSynced && !stickySynced {
				continue
			}
			for idx, ident := range vs.Names {
				if idx >= len(vs.Values) {
					continue
				}
				lit, ok := vs.Values[idx].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				unquoted, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatalf("unquote %s: %v", ident.Name, err)
				}
				result[ident.Name] = unquoted
			}
		}
	}
	return result
}

func hasSyncComment(groups ...*ast.CommentGroup) bool {
	for _, cg := range groups {
		if cg == nil {
			continue
		}
		for _, c := range cg.List {
			if syncDirectiveRegex.MatchString(c.Text) {
				return true
			}
		}
	}
	return false
}
