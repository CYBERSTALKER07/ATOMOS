package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var syncTagPattern = regexp.MustCompile(`@Sync(?:\(([^)\n]+)\))?`)

type Registry struct {
	Source      string    `json:"source"`
	GeneratedAt time.Time `json:"generated_at"`
	Events      []Event   `json:"events"`
	Payloads    []Payload `json:"payloads"`
	Warnings    []string  `json:"warnings,omitempty"`
}

type Event struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	Payload string `json:"payload,omitempty"`
}

type Payload struct {
	Name   string         `json:"name"`
	Fields []PayloadField `json:"fields"`
}

type PayloadField struct {
	Name          string `json:"name"`
	JSONName      string `json:"json_name,omitempty"`
	JSONTag       string `json:"json_tag,omitempty"`
	JSONOmitEmpty bool   `json:"json_omitempty,omitempty"`
	GoType        string `json:"go_type"`
	Optional      bool   `json:"optional"`
	Nullable      bool   `json:"nullable"`
}

type syncDirective struct {
	Enabled bool
	Payload string
}

func main() {
	var (
		sourcePath = flag.String("source", "", "Path to source-of-truth Go events file")
		outPath    = flag.String("out", "", "Output file path for extracted registry JSON (stdout when empty)")
		tsOutPath  = flag.String("ts-out", "", "Output file path for generated TypeScript definitions")
		schemaOut  = flag.String("schema-out", "", "Output file path for generated JSON-Schema definitions")
		mode       = flag.String("mode", "registry", "Generation mode: registry|json-schema|all")
		pretty     = flag.Bool("pretty", true, "Pretty-print JSON output")
		strict     = flag.Bool("strict", false, "Fail when a synced event cannot map to a synced payload")
	)
	flag.Parse()

	modeValue := strings.ToLower(strings.TrimSpace(*mode))
	switch modeValue {
	case "registry", "json-schema", "all":
	default:
		log.Fatalf("invalid -mode %q (expected registry|json-schema|all)", *mode)
	}

	emitRegistry := modeValue == "registry" || modeValue == "all"
	emitSchema := modeValue == "json-schema" || modeValue == "all"

	resolvedSource, err := resolveSourcePath(*sourcePath)
	if err != nil {
		log.Fatalf("resolve source: %v", err)
	}

	registry, err := extractRegistry(resolvedSource)
	if err != nil {
		log.Fatalf("extract registry: %v", err)
	}

	unmapped := 0
	for _, evt := range registry.Events {
		if evt.Payload == "" {
			unmapped++
		}
	}
	if *strict && unmapped > 0 {
		log.Fatalf("strict mode: %d synced event(s) missing payload mapping", unmapped)
	}

	emitRegistryJSON := false
	writeRegistryToStdout := false
	if emitRegistry {
		if *outPath != "" {
			emitRegistryJSON = true
		} else if *tsOutPath == "" && !emitSchema {
			emitRegistryJSON = true
			writeRegistryToStdout = true
		}
	}

	if emitRegistryJSON {
		jsonBytes, err := marshalRegistry(registry, *pretty)
		if err != nil {
			log.Fatalf("marshal registry: %v", err)
		}

		if !writeRegistryToStdout {
			if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
				log.Fatalf("mkdir output dir: %v", err)
			}
			if err := os.WriteFile(*outPath, jsonBytes, 0o644); err != nil {
				log.Fatalf("write output: %v", err)
			}
		} else {
			if _, err := os.Stdout.Write(jsonBytes); err != nil {
				log.Fatalf("write stdout: %v", err)
			}
			if len(jsonBytes) == 0 || jsonBytes[len(jsonBytes)-1] != '\n' {
				fmt.Fprintln(os.Stdout)
			}
		}
	}

	schemaOutPath := strings.TrimSpace(*schemaOut)
	if schemaOutPath == "" && modeValue == "json-schema" && *outPath != "" {
		schemaOutPath = *outPath
	}
	if emitSchema && schemaOutPath == "" && (emitRegistryJSON || *tsOutPath != "") {
		log.Fatalf("schema output path is required when combining schema with other outputs; pass -schema-out")
	}

	if emitSchema {
		schemaBytes, err := emitJSONSchema(registry, *pretty)
		if err != nil {
			log.Fatalf("emit schema: %v", err)
		}

		if schemaOutPath != "" {
			if err := os.MkdirAll(filepath.Dir(schemaOutPath), 0o755); err != nil {
				log.Fatalf("mkdir schema output dir: %v", err)
			}
			if err := os.WriteFile(schemaOutPath, schemaBytes, 0o644); err != nil {
				log.Fatalf("write schema output: %v", err)
			}
		} else {
			if _, err := os.Stdout.Write(schemaBytes); err != nil {
				log.Fatalf("write schema stdout: %v", err)
			}
			if len(schemaBytes) == 0 || schemaBytes[len(schemaBytes)-1] != '\n' {
				fmt.Fprintln(os.Stdout)
			}
		}
	}

	if *tsOutPath != "" {
		tsContent, err := emitTypeScript(registry)
		if err != nil {
			log.Fatalf("emit ts: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(*tsOutPath), 0o755); err != nil {
			log.Fatalf("mkdir ts output dir: %v", err)
		}
		if err := os.WriteFile(*tsOutPath, []byte(tsContent), 0o644); err != nil {
			log.Fatalf("write ts output: %v", err)
		}
	}

	fmt.Fprintf(os.Stderr, "gen-contracts: extracted %d events and %d payloads from %s\n", len(registry.Events), len(registry.Payloads), resolvedSource)
	if *outPath != "" {
		fmt.Fprintf(os.Stderr, "gen-contracts: wrote registry JSON to %s\n", *outPath)
	}
	if emitSchema {
		if schemaOutPath != "" {
			fmt.Fprintf(os.Stderr, "gen-contracts: wrote JSON schema to %s\n", schemaOutPath)
		} else {
			fmt.Fprintf(os.Stderr, "gen-contracts: wrote JSON schema to stdout\n")
		}
	}
	if *tsOutPath != "" {
		fmt.Fprintf(os.Stderr, "gen-contracts: wrote TypeScript contracts to %s\n", *tsOutPath)
	}
	if unmapped > 0 {
		fmt.Fprintf(os.Stderr, "gen-contracts: warning: %d event(s) are synced but have no payload mapping\n", unmapped)
	}
}

func resolveSourcePath(explicit string) (string, error) {
	if explicit != "" {
		info, err := os.Stat(explicit)
		if err == nil {
			if info.IsDir() {
				return explicit, nil
			}
			return explicit, nil
		}
		return "", fmt.Errorf("source not found: %s", explicit)
	}

	candidates := []string{
		"events",
		"internal/events",
		"kafka",
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", errors.New("no events source file found; pass -source")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func extractRegistry(source string) (Registry, error) {
	fset := token.NewFileSet()
	
	var files []*ast.File
	info, err := os.Stat(source)
	if err != nil {
		return Registry{}, err
	}
	
	if info.IsDir() {
		pkgs, err := parser.ParseDir(fset, source, nil, parser.ParseComments)
		if err != nil {
			return Registry{}, err
		}
		for _, pkg := range pkgs {
			for _, file := range pkg.Files {
				files = append(files, file)
			}
		}
	} else {
		file, err := parser.ParseFile(fset, source, nil, parser.ParseComments)
		if err != nil {
			return Registry{}, err
		}
		files = append(files, file)
	}

	syncMode := false
	for _, file := range files {
		if fileHasSyncDirective(file) {
			syncMode = true
			break
		}
	}

	registry := Registry{
		Source:      source,
		GeneratedAt: time.Now().UTC(),
		Events:      make([]Event, 0),
		Payloads:    make([]Payload, 0),
		Warnings:    make([]string, 0),
	}

	for _, file := range files {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}

			switch genDecl.Tok {
			case token.CONST:
				extractConstants(genDecl, &registry, syncMode)
			case token.TYPE:
				extractStructs(fset, genDecl, &registry, syncMode)
			}
		}
	}

	if !syncMode {
		if err := mergeLegacyWSEvents(&registry); err != nil {
			registry.Warnings = append(registry.Warnings, fmt.Sprintf("supplemental ws merge failed: %v", err))
		}
	}

	payloadByName := make(map[string]struct{}, len(registry.Payloads))
	payloadNames := make([]string, 0, len(registry.Payloads))
	for _, payload := range registry.Payloads {
		payloadByName[payload.Name] = struct{}{}
		payloadNames = append(payloadNames, payload.Name)
	}

	for idx := range registry.Events {
		event := &registry.Events[idx]
		if event.Payload != "" {
			if _, ok := payloadByName[event.Payload]; !ok {
				registry.Warnings = append(registry.Warnings, fmt.Sprintf("event %s references missing payload %s", event.Name, event.Payload))
				event.Payload = ""
			}
			continue
		}

		matched, ambiguous := inferPayloadName(*event, payloadNames)
		if matched == "" {
			registry.Warnings = append(registry.Warnings, fmt.Sprintf("event %s has no inferred payload", event.Name))
			continue
		}
		event.Payload = matched
		if ambiguous {
			registry.Warnings = append(registry.Warnings, fmt.Sprintf("event %s has multiple payload candidates; chose %s", event.Name, matched))
		}
	}

	registry.Events = dedupeEventsByValue(registry.Events)

	sort.Slice(registry.Events, func(i, j int) bool {
		if registry.Events[i].Value == registry.Events[j].Value {
			return registry.Events[i].Name < registry.Events[j].Name
		}
		return registry.Events[i].Value < registry.Events[j].Value
	})
	sort.Slice(registry.Payloads, func(i, j int) bool {
		return registry.Payloads[i].Name < registry.Payloads[j].Name
	})

	if len(registry.Warnings) == 0 {
		registry.Warnings = nil
	}

	return registry, nil
}

func extractConstants(genDecl *ast.GenDecl, registry *Registry, syncMode bool) {
	declDir := parseSyncDirective(genDecl.Doc)
	var lastSpecDir syncDirective

	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		specDir := parseSyncDirective(valueSpec.Doc, valueSpec.Comment)
		if specDir.Enabled {
			lastSpecDir = specDir
		} else {
			specDir = lastSpecDir
		}
		
		active := declDir.Enabled || specDir.Enabled
		if syncMode && !active {
			continue
		}

		payloadName := ""
		if syncMode {
			payloadName = firstNonEmpty(specDir.Payload, declDir.Payload)
		}

		for idx, ident := range valueSpec.Names {
			if !syncMode && !strings.HasPrefix(ident.Name, "Event") {
				continue
			}

			value, ok := extractConstString(valueSpec.Values, idx)
			if !ok {
				if syncMode {
					registry.Warnings = append(registry.Warnings, fmt.Sprintf("const %s is @Sync but has no string literal value", ident.Name))
				}
				continue
			}
			if !syncMode && !isLikelyEventValue(value) {
				continue
			}

			registry.Events = append(registry.Events, Event{
				Name:    ident.Name,
				Value:   value,
				Payload: payloadName,
			})
		}
	}
}

func extractStructs(fset *token.FileSet, genDecl *ast.GenDecl, registry *Registry, syncMode bool) {
	declDir := parseSyncDirective(genDecl.Doc)

	for _, spec := range genDecl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}

		specDir := parseSyncDirective(typeSpec.Doc, typeSpec.Comment)
		active := declDir.Enabled || specDir.Enabled
		if syncMode && !active && !isLikelyPayloadStruct(typeSpec.Name.Name) {
			continue
		}
		if !syncMode && !isLikelyPayloadStruct(typeSpec.Name.Name) {
			continue
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			if syncMode {
				registry.Warnings = append(registry.Warnings, fmt.Sprintf("type %s is @Sync but is not a struct", typeSpec.Name.Name))
			}
			continue
		}

		registry.Payloads = append(registry.Payloads, Payload{
			Name:   typeSpec.Name.Name,
			Fields: extractFields(fset, structType),
		})
	}
}

func extractFields(fset *token.FileSet, st *ast.StructType) []PayloadField {
	if st.Fields == nil {
		return nil
	}

	fields := make([]PayloadField, 0)
	for _, field := range st.Fields.List {
		goType := renderExpr(fset, field.Type)
		jsonName, jsonTag, omitempty := parseJSONTag(field.Tag)
		nullable := isNullableType(field.Type)

		if len(field.Names) == 0 {
			embeddedName := embeddedFieldName(field.Type)
			fields = append(fields, PayloadField{
				Name:          embeddedName,
				JSONName:      jsonName,
				JSONTag:       jsonTag,
				JSONOmitEmpty: omitempty,
				GoType:        goType,
				Optional:      omitempty || nullable,
				Nullable:      nullable,
			})
			continue
		}

		for _, name := range field.Names {
			fields = append(fields, PayloadField{
				Name:          name.Name,
				JSONName:      jsonName,
				JSONTag:       jsonTag,
				JSONOmitEmpty: omitempty,
				GoType:        goType,
				Optional:      omitempty || nullable,
				Nullable:      nullable,
			})
		}
	}

	return fields
}

func embeddedFieldName(expr ast.Expr) string {
	name := compactIdentifier(expr)
	if name == "" {
		return "embedded"
	}
	return name
}

func compactIdentifier(expr ast.Expr) string {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return x.Sel.Name
	case *ast.StarExpr:
		return compactIdentifier(x.X)
	default:
		return ""
	}
}

func parseJSONTag(tag *ast.BasicLit) (jsonName string, rawJSONTag string, omitempty bool) {
	if tag == nil {
		return "", "", false
	}

	rawTag := strings.Trim(tag.Value, "`")
	jsonTag := reflect.StructTag(rawTag).Get("json")
	if jsonTag == "" {
		return "", "", false
	}

	parts := strings.Split(jsonTag, ",")
	if len(parts) > 0 {
		jsonName = strings.TrimSpace(parts[0])
	}
	for _, opt := range parts[1:] {
		if strings.TrimSpace(opt) == "omitempty" {
			omitempty = true
			break
		}
	}

	return jsonName, jsonTag, omitempty
}

func isNullableType(expr ast.Expr) bool {
	switch x := expr.(type) {
	case *ast.StarExpr:
		return true
	case *ast.ArrayType:
		return x.Len == nil
	case *ast.MapType:
		return true
	case *ast.InterfaceType:
		return true
	case *ast.ChanType:
		return true
	case *ast.SelectorExpr:
		return strings.HasPrefix(x.Sel.Name, "Null")
	case *ast.Ident:
		return strings.HasPrefix(x.Name, "Null")
	default:
		return false
	}
}

func renderExpr(fset *token.FileSet, expr ast.Expr) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, expr); err != nil {
		return ""
	}
	return buf.String()
}

func parseSyncDirective(groups ...*ast.CommentGroup) syncDirective {
	var directive syncDirective
	for _, group := range groups {
		if group == nil {
			continue
		}
		matches := syncTagPattern.FindAllStringSubmatch(group.Text(), -1)
		for _, match := range matches {
			directive.Enabled = true
			payload := ""
			if len(match) > 1 {
				payload = strings.TrimSpace(match[1])
			}
			if payload != "" {
				directive.Payload = payload
			}
		}
	}
	return directive
}

func fileHasSyncDirective(file *ast.File) bool {
	for _, cg := range file.Comments {
		if cg == nil {
			continue
		}
		if syncTagPattern.MatchString(cg.Text()) {
			return true
		}
	}
	return false
}

func isLikelyPayloadStruct(typeName string) bool {
	return strings.HasSuffix(typeName, "Event") || strings.HasSuffix(typeName, "Payload")
}

func isLikelyEventValue(value string) bool {
	if value == "" {
		return false
	}
	if strings.ContainsAny(value, " \\t\\n") {
		return false
	}
	for _, r := range value {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return false
	}
	return true
}

func dedupeEventsByValue(events []Event) []Event {
	if len(events) == 0 {
		return events
	}
	byValue := make(map[string]Event, len(events))
	for _, evt := range events {
		current, exists := byValue[evt.Value]
		if !exists {
			byValue[evt.Value] = evt
			continue
		}
		if current.Payload == "" && evt.Payload != "" {
			byValue[evt.Value] = evt
		}
	}
	result := make([]Event, 0, len(byValue))
	for _, evt := range byValue {
		result = append(result, evt)
	}
	return result
}

func mergeLegacyWSEvents(registry *Registry) error {
	wsSource, ok := resolveSupplementalWSEventsPath(registry.Source)
	if !ok {
		return nil
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, wsSource, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for idx, ident := range valueSpec.Names {
				if !strings.HasPrefix(ident.Name, "Event") {
					continue
				}
				value, ok := extractConstString(valueSpec.Values, idx)
				if !ok || !isLikelyEventValue(value) {
					continue
				}
				registry.Events = append(registry.Events, Event{Name: ident.Name, Value: value})
			}
		}
	}

	return nil
}

func resolveSupplementalWSEventsPath(primarySource string) (string, bool) {
	candidates := make([]string, 0, 4)
	if strings.Contains(primarySource, string(filepath.Separator)+"kafka"+string(filepath.Separator)+"events.go") {
		candidates = append(candidates, strings.Replace(primarySource, string(filepath.Separator)+"kafka"+string(filepath.Separator)+"events.go", string(filepath.Separator)+"ws"+string(filepath.Separator)+"events.go", 1))
	}
	candidates = append(candidates,
		"ws/events.go",
		"pegasus/apps/backend-go/ws/events.go",
	)
	for _, candidate := range candidates {
		if fileExists(candidate) {
			return candidate, true
		}
	}
	return "", false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func extractConstString(values []ast.Expr, idx int) (string, bool) {
	if len(values) == 0 {
		return "", false
	}

	valueIdx := idx
	if idx >= len(values) {
		if len(values) == 1 {
			valueIdx = 0
		} else {
			return "", false
		}
	}

	basic, ok := values[valueIdx].(*ast.BasicLit)
	if !ok || basic.Kind != token.STRING {
		return "", false
	}

	unquoted, err := strconv.Unquote(basic.Value)
	if err != nil {
		return "", false
	}
	return unquoted, true
}

func inferPayloadName(event Event, payloadNames []string) (name string, ambiguous bool) {
	if len(payloadNames) == 0 {
		return "", false
	}

	payloadSet := make(map[string]struct{}, len(payloadNames))
	for _, candidate := range payloadNames {
		payloadSet[candidate] = struct{}{}
	}

	base := eventBaseName(event)
	directCandidates := []string{
		base + "Event",
		base + "Payload",
		base,
		event.Name + "Event",
		event.Name + "Payload",
	}
	for _, candidate := range directCandidates {
		if _, ok := payloadSet[candidate]; ok {
			return candidate, false
		}
	}

	eventNorm := normalizeSymbol(base)
	if eventNorm == "" {
		eventNorm = normalizeSymbol(event.Name)
	}
	if eventNorm == "" {
		return "", false
	}

	matches := make([]string, 0)
	for _, payloadName := range payloadNames {
		if normalizeSymbol(payloadName) == eventNorm {
			matches = append(matches, payloadName)
		}
	}
	if len(matches) == 0 {
		return "", false
	}

	sort.Strings(matches)
	return matches[0], len(matches) > 1
}

func eventBaseName(event Event) string {
	base := snakeToCamel(event.Value)
	if base != "" {
		return base
	}
	name := strings.TrimPrefix(event.Name, "Event")
	if name != "" {
		return name
	}
	return event.Name
}

func snakeToCamel(value string) string {
	if value == "" {
		return ""
	}

	parts := strings.FieldsFunc(value, func(r rune) bool {
		return !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})
	if len(parts) == 0 {
		return ""
	}

	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		lower := strings.ToLower(part)
		b.WriteString(strings.ToUpper(lower[:1]))
		if len(lower) > 1 {
			b.WriteString(lower[1:])
		}
	}
	return b.String()
}

func normalizeSymbol(value string) string {
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	lower = strings.TrimPrefix(lower, "event")
	lower = strings.TrimSuffix(lower, "event")
	lower = strings.TrimSuffix(lower, "payload")

	var b strings.Builder
	for _, r := range lower {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}

	clean := b.String()
	clean = strings.TrimPrefix(clean, "event")
	clean = strings.TrimSuffix(clean, "event")
	clean = strings.TrimSuffix(clean, "payload")
	return clean
}

func marshalRegistry(reg Registry, pretty bool) ([]byte, error) {
	if pretty {
		return json.MarshalIndent(reg, "", "  ")
	}
	return json.Marshal(reg)
}

func emitJSONSchema(reg Registry, pretty bool) ([]byte, error) {
	payloadByName := make(map[string]Payload, len(reg.Payloads))
	for _, payload := range reg.Payloads {
		payloadByName[payload.Name] = payload
	}

	type eventSchemaMapping struct {
		EventValue   string
		PayloadDefID string
	}

	mappings := make([]eventSchemaMapping, 0, len(reg.Events))
	defs := make(map[string]map[string]any)
	unknownNeeded := false

	for _, evt := range reg.Events {
		if payload, ok := payloadByName[evt.Payload]; ok && evt.Payload != "" {
			defName := tsPayloadName(payload.Name)
			if _, exists := defs[defName]; !exists {
				defs[defName] = payloadJSONSchema(payload)
			}
			mappings = append(mappings, eventSchemaMapping{EventValue: evt.Value, PayloadDefID: defName})
			continue
		}
		unknownNeeded = true
		mappings = append(mappings, eventSchemaMapping{EventValue: evt.Value, PayloadDefID: "UnknownEventPayload"})
	}

	sort.Slice(mappings, func(i, j int) bool { return mappings[i].EventValue < mappings[j].EventValue })

	if unknownNeeded {
		defs["UnknownEventPayload"] = map[string]any{
			"title":                "UnknownEventPayload",
			"type":                 "object",
			"additionalProperties": true,
		}
	}

	eventEnum := make([]string, 0, len(mappings))
	eventPayloadMap := make(map[string]string, len(mappings))
	oneOf := make([]map[string]any, 0, len(mappings))
	for _, item := range mappings {
		eventEnum = append(eventEnum, item.EventValue)
		eventPayloadMap[item.EventValue] = item.PayloadDefID
		oneOf = append(oneOf, map[string]any{
			"title": fmt.Sprintf("%sEnvelope", toPascalFromEvent(item.EventValue)),
			"allOf": []any{
				map[string]any{
					"$ref": fmt.Sprintf("#/$defs/%s", item.PayloadDefID),
				},
				map[string]any{
					"type":     "object",
					"required": []string{"type"},
					"properties": map[string]any{
						"type": map[string]any{
							"const": item.EventValue,
						},
					},
				},
			},
		})
	}

	sort.Strings(eventEnum)

	defKeys := make([]string, 0, len(defs))
	for name := range defs {
		defKeys = append(defKeys, name)
	}
	sort.Strings(defKeys)
	orderedDefs := make(map[string]any, len(defs))
	for _, key := range defKeys {
		orderedDefs[key] = defs[key]
	}

	doc := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"title":   "PegasusWSEventEnvelope",
		"type":    "object",
		"required": []string{
			"type",
		},
		"properties": map[string]any{
			"type": map[string]any{
				"type": "string",
				"enum": eventEnum,
			},
		},
		"oneOf":               oneOf,
		"$defs":               orderedDefs,
		"x-event-payload-map": eventPayloadMap,
	}

	if pretty {
		return json.MarshalIndent(doc, "", "  ")
	}
	return json.Marshal(doc)
}

func payloadJSONSchema(payload Payload) map[string]any {
	properties := make(map[string]any)
	required := make([]string, 0, len(payload.Fields))

	for _, field := range payload.Fields {
		jsonName := field.JSONName
		if jsonName == "" {
			jsonName = toSnakeCase(field.Name)
		}
		if jsonName == "-" || jsonName == "" {
			continue
		}

		properties[jsonName] = goTypeToJSONSchema(field.GoType, field.Nullable)
		if !field.Optional {
			required = append(required, jsonName)
		}
	}

	sort.Strings(required)

	schema := map[string]any{
		"title":                tsPayloadName(payload.Name),
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": true,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func goTypeToJSONSchema(goType string, nullable bool) map[string]any {
	t := strings.TrimSpace(goType)
	if t == "" {
		return nullableSchema(map[string]any{}, nullable)
	}

	for strings.HasPrefix(t, "*") {
		t = strings.TrimSpace(strings.TrimPrefix(t, "*"))
		nullable = true
	}

	if strings.HasPrefix(t, "[]") {
		items := goTypeToJSONSchema(strings.TrimPrefix(t, "[]"), false)
		return nullableSchema(map[string]any{
			"type":  "array",
			"items": items,
		}, nullable)
	}

	if strings.HasPrefix(t, "map[") {
		if idx := strings.Index(t, "]"); idx != -1 {
			valueSchema := goTypeToJSONSchema(strings.TrimSpace(t[idx+1:]), false)
			return nullableSchema(map[string]any{
				"type":                 "object",
				"additionalProperties": valueSchema,
			}, nullable)
		}
		return nullableSchema(map[string]any{
			"type":                 "object",
			"additionalProperties": true,
		}, nullable)
	}

	switch t {
	case "string":
		return nullableSchema(map[string]any{"type": "string"}, nullable)
	case "bool":
		return nullableSchema(map[string]any{"type": "boolean"}, nullable)
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "byte", "rune":
		return nullableSchema(map[string]any{"type": "integer"}, nullable)
	case "float32", "float64":
		return nullableSchema(map[string]any{"type": "number"}, nullable)
	case "time.Time":
		return nullableSchema(map[string]any{"type": "string", "format": "date-time"}, nullable)
	case "interface{}", "any", "json.RawMessage":
		return nullableSchema(map[string]any{}, nullable)
	}

	if strings.HasPrefix(t, "spanner.Null") {
		suffix := strings.TrimPrefix(t, "spanner.Null")
		nullable = true
		switch suffix {
		case "String":
			return nullableSchema(map[string]any{"type": "string"}, nullable)
		case "Bool":
			return nullableSchema(map[string]any{"type": "boolean"}, nullable)
		case "Int64":
			return nullableSchema(map[string]any{"type": "integer"}, nullable)
		case "Float64":
			return nullableSchema(map[string]any{"type": "number"}, nullable)
		case "Time":
			return nullableSchema(map[string]any{"type": "string", "format": "date-time"}, nullable)
		default:
			return nullableSchema(map[string]any{}, nullable)
		}
	}

	if strings.Contains(t, ".") {
		return nullableSchema(map[string]any{
			"type":                 "object",
			"additionalProperties": true,
		}, nullable)
	}

	return nullableSchema(map[string]any{}, nullable)
}

func nullableSchema(base map[string]any, nullable bool) map[string]any {
	if !nullable {
		return base
	}
	return map[string]any{
		"anyOf": []any{
			base,
			map[string]any{"type": "null"},
		},
	}
}

func emitTypeScript(reg Registry) (string, error) {
	payloadByName := make(map[string]Payload, len(reg.Payloads))
	for _, payload := range reg.Payloads {
		payloadByName[payload.Name] = payload
	}

	type mappedEvent struct {
		Event       Event
		PayloadType string
	}
	mapped := make([]mappedEvent, 0, len(reg.Events))
	usedPayloads := make(map[string]Payload)
	unknownUsed := false

	for _, evt := range reg.Events {
		if payload, ok := payloadByName[evt.Payload]; ok && evt.Payload != "" {
			tsName := tsPayloadName(payload.Name)
			usedPayloads[tsName] = payload
			mapped = append(mapped, mappedEvent{Event: evt, PayloadType: tsName})
			continue
		}
		unknownUsed = true
		mapped = append(mapped, mappedEvent{Event: evt, PayloadType: "UnknownEventPayload"})
	}

	sort.Slice(mapped, func(i, j int) bool { return mapped[i].Event.Value < mapped[j].Event.Value })

	usedPayloadNames := make([]string, 0, len(usedPayloads))
	for name := range usedPayloads {
		usedPayloadNames = append(usedPayloadNames, name)
	}
	sort.Strings(usedPayloadNames)

	var b strings.Builder
	b.WriteString("/**\n")
	b.WriteString(" * GENERATED FILE - DO NOT EDIT MANUALLY.\n")
	b.WriteString(fmt.Sprintf(" * Source: %s\n", reg.Source))
	b.WriteString(fmt.Sprintf(" * Generated at: %s\n", reg.GeneratedAt.Format(time.RFC3339)))
	b.WriteString(" */\n\n")

	b.WriteString("export enum WSEventType {\n")
	for _, item := range mapped {
		b.WriteString(fmt.Sprintf("  %s = '%s',\n", tsEnumKey(item.Event.Value), item.Event.Value))
	}
	b.WriteString("}\n\n")

	if unknownUsed {
		b.WriteString("export interface UnknownEventPayload {\n")
		b.WriteString("  [key: string]: any;\n")
		b.WriteString("}\n\n")
	}

	for _, tsName := range usedPayloadNames {
		payload := usedPayloads[tsName]
		b.WriteString(fmt.Sprintf("export interface %s {\n", tsName))
		if len(payload.Fields) == 0 {
			b.WriteString("  // Empty payload\n")
		}
		for _, field := range payload.Fields {
			jsonName := field.JSONName
			if jsonName == "" {
				jsonName = toSnakeCase(field.Name)
			}
			if jsonName == "-" {
				continue
			}

			prop := jsonName
			if !isValidTSIdentifier(prop) {
				prop = fmt.Sprintf("'%s'", strings.ReplaceAll(prop, "'", "\\\\'"))
			}

			optionalMark := ""
			if field.Optional {
				optionalMark = "?"
			}

			typeName := mapGoTypeToTSType(field.GoType)
			b.WriteString(fmt.Sprintf("  %s%s: %s;\n", prop, optionalMark, typeName))
		}
		b.WriteString("}\n\n")
	}

	b.WriteString("/** Strong-typed map from event type to payload shape. */\n")
	b.WriteString("export interface WSEventPayloadMap {\n")
	for _, item := range mapped {
		b.WriteString(fmt.Sprintf("  '%s': %s;\n", item.Event.Value, item.PayloadType))
	}
	b.WriteString("}\n\n")

	b.WriteString("export type WSEventTypeValue = keyof WSEventPayloadMap;\n")
	b.WriteString("export type WSEventMessage<T extends WSEventTypeValue = WSEventTypeValue> = { type: T } & WSEventPayloadMap[T];\n")
	b.WriteString("export type WSEvent = { [K in WSEventTypeValue]: WSEventMessage<K> }[WSEventTypeValue];\n\n")

	for _, item := range mapped {
		aliasName := fmt.Sprintf("%sWSEvent", toPascalFromEvent(item.Event.Value))
		b.WriteString(fmt.Sprintf("export type %s = WSEventMessage<'%s'>;\n", aliasName, item.Event.Value))
	}

	return b.String(), nil
}

func tsPayloadName(goName string) string {
	if strings.HasSuffix(goName, "Payload") {
		return goName
	}
	if strings.HasSuffix(goName, "Event") {
		return strings.TrimSuffix(goName, "Event") + "Payload"
	}
	return goName + "Payload"
}

func tsEnumKey(eventValue string) string {
	if eventValue == "" {
		return "UNKNOWN"
	}
	var b strings.Builder
	for i, r := range eventValue {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			if i == 0 && r >= '0' && r <= '9' {
				b.WriteString("E_")
			}
			b.WriteRune(r)
			continue
		}
		b.WriteRune('_')
	}
	result := strings.Trim(resultUnderscores(b.String()), "_")
	if result == "" {
		return "UNKNOWN"
	}
	return result
}

func resultUnderscores(value string) string {
	for strings.Contains(value, "__") {
		value = strings.ReplaceAll(value, "__", "_")
	}
	return value
}

func toPascalFromEvent(value string) string {
	parts := strings.Split(value, "_")
	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		lower := strings.ToLower(part)
		b.WriteString(strings.ToUpper(lower[:1]))
		if len(lower) > 1 {
			b.WriteString(lower[1:])
		}
	}
	if b.Len() == 0 {
		return "Unknown"
	}
	return b.String()
}

func toSnakeCase(value string) string {
	if value == "" {
		return ""
	}
	var out []rune
	var prev rune
	for i, r := range value {
		if unicode.IsUpper(r) {
			if i > 0 && (unicode.IsLower(prev) || unicode.IsDigit(prev)) {
				out = append(out, '_')
			}
			out = append(out, unicode.ToLower(r))
		} else {
			out = append(out, unicode.ToLower(r))
		}
		prev = r
	}
	return string(out)
}

func isValidTSIdentifier(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if r == '_' || r == '$' || unicode.IsLetter(r) {
				continue
			}
			return false
		}
		if r == '_' || r == '$' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		return false
	}
	return true
}

func mapGoTypeToTSType(goType string) string {
	t := strings.TrimSpace(goType)
	if t == "" {
		return "any"
	}

	if strings.HasPrefix(t, "*") {
		return mapGoTypeToTSType(strings.TrimSpace(strings.TrimPrefix(t, "*")))
	}
	if strings.HasPrefix(t, "[]") {
		inner := mapGoTypeToTSType(strings.TrimPrefix(t, "[]"))
		return inner + "[]"
	}
	if strings.HasPrefix(t, "map[") {
		if idx := strings.Index(t, "]"); idx != -1 {
			valueType := mapGoTypeToTSType(strings.TrimSpace(t[idx+1:]))
			return fmt.Sprintf("Record<string, %s>", valueType)
		}
		return "Record<string, any>"
	}

	switch t {
	case "string":
		return "string"
	case "bool":
		return "boolean"
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "float32", "float64", "byte", "rune":
		return "number"
	case "time.Time":
		return "string"
	case "interface{}", "any", "json.RawMessage":
		return "any"
	}

	if strings.HasPrefix(t, "spanner.Null") {
		suffix := strings.TrimPrefix(t, "spanner.Null")
		switch suffix {
		case "String":
			return "string | null"
		case "Bool":
			return "boolean | null"
		case "Int64", "Float64":
			return "number | null"
		case "Time":
			return "string | null"
		default:
			return "any"
		}
	}

	if strings.Contains(t, ".") {
		// Cross-package and selector types are emitted as any unless modeled explicitly.
		return "any"
	}

	return "any"
}
