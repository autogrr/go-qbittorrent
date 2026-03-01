// cmd/gen generates merge_gen.go and deepcopy_gen.go for the qbittorrent package.
//
// It parses types.go via go/ast, classifies every struct field, then emits:
//
//	merge_gen.go   – mergePartial<Type> functions (nil-check for pointer fields;
//	                 skip-if-zero for value types; always-overwrite for plain bools)
//	deepcopy_gen.go – deepCopy<Type> functions (deep-copies pointer fields so
//	                 callers cannot mutate internal SyncState through them)
//
// Run via:   go generate ./...
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"
	"text/template"
)

// mergeTargets lists the structs for which a mergePartial function is generated.
var mergeTargets = []string{"Torrent", "ServerState"}

// internFields maps struct name → set of field names whose *string values should
// be interned during deep-copy.  Only low-cardinality fields that repeat across
// many Torrent entries benefit; unique-per-torrent fields (Hash, Name, …) must
// NOT be listed here because interning them would bloat the interner for no gain.
var internFields = map[string]map[string]bool{
	"Torrent": {
		"Category":     true, // typically a small set shared across all torrents
		"SavePath":     true, // most torrents share one of a few download roots
		"DownloadPath": true, // same as SavePath
		"ContentPath":  true, // shared across cross-seed copies of the same content
		"State":        true, // only ~21 possible values (TorrentState constants)
		"Tags":         true, // comma-separated tag strings, often repeated
		"Tracker":      true, // tracker URL repeated across all torrents of a tracker
		"Name":         true, // cross-seed duplicates share identical names
	},
	"ServerState": {
		"ConnectionStatus": true, // only "connected", "firewalled", "disconnected"
	},
}

// ---- field descriptor -------------------------------------------------------

type fieldDesc struct {
	Name         string
	AlwaysWrite  bool   // bool, *T — overwritten even when zero
	Zero         string // zero literal for skip-if-zero fields: `""` or `0`
	IsPointer    bool   // *T field — needs deep-copy
	IsSlice      bool   // []T field — copy only when len > 0 in merge
	PointeeType  string // T in *T
	JSONTag      string // json tag name, e.g. "added_on"
	Underlying   string // resolved underlying type, e.g. "string", "int64"
	IsBool       bool   // underlying type is bool
	InternString bool   // intern the string value (low-cardinality fields only)
}

// ---- AST helpers ------------------------------------------------------------

// resolveUnderlying follows chains of named-type aliases back to a builtin name.
// e.g., TorrentState → string.
func resolveUnderlying(name string, aliases map[string]string) string {
	seen := map[string]bool{}
	for {
		if seen[name] {
			return name
		}
		seen[name] = true
		if under, ok := aliases[name]; ok {
			name = under
		} else {
			return name
		}
	}
}

func classifyField(name string, expr ast.Expr, aliases map[string]string) fieldDesc {
	switch t := expr.(type) {
	case *ast.StarExpr:
		// *T — always overwrite; pointee must be deep-copied by callers.
		pt := ""
		var under string
		var isBool bool
		if id, ok := t.X.(*ast.Ident); ok {
			pt = id.Name
			under = resolveUnderlying(id.Name, aliases)
			isBool = under == "bool"
		}
		return fieldDesc{Name: name, AlwaysWrite: true, IsPointer: true, PointeeType: pt, Underlying: under, IsBool: isBool}

	case *ast.Ident:
		under := resolveUnderlying(t.Name, aliases)
		switch under {
		case "bool":
			return fieldDesc{Name: name, AlwaysWrite: true, Underlying: under, IsBool: true}
		case "string":
			return fieldDesc{Name: name, Zero: `""`, Underlying: under}
		case "int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64",
			"float32", "float64":
			return fieldDesc{Name: name, Zero: "0", Underlying: under}
		default:
			// Opaque/unknown type; always overwrite to be safe.
			return fieldDesc{Name: name, AlwaysWrite: true, Underlying: under}
		}

	case *ast.ArrayType:
		// []T — only copy when src is non-empty (presence check via len).
		return fieldDesc{Name: name, IsSlice: true}
	default:
		// Map, interface, etc. — always overwrite.
		return fieldDesc{Name: name, AlwaysWrite: true}
	}
}

// parseStructs reads types.go and returns a map of struct name → []fieldDesc,
// plus a map of named-type aliases used for underlying-type resolution.
func parseStructs(path string) (map[string][]fieldDesc, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}

	// Pass 1: collect type aliases  (type Foo Bar).
	aliases := map[string]string{}
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if id, ok := ts.Type.(*ast.Ident); ok {
				aliases[ts.Name.Name] = id.Name
			}
		}
	}

	// Pass 2: collect struct declarations.
	out := map[string][]fieldDesc{}
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			var fields []fieldDesc
			for _, ff := range st.Fields.List {
				jsonTag := extractJSONTag(ff.Tag)
				for _, nm := range ff.Names {
					fd := classifyField(nm.Name, ff.Type, aliases)
					fd.JSONTag = jsonTag
					fields = append(fields, fd)
				}
			}
			out[ts.Name.Name] = fields
		}
	}
	return out, nil
}

// extractJSONTag returns the JSON field name from a struct tag literal.
// Returns "" when the tag is absent, "-", or empty.
func extractJSONTag(tag *ast.BasicLit) string {
	if tag == nil {
		return ""
	}
	// tag.Value includes backticks: `json:"foo,omitempty"`
	raw := tag.Value
	const key = `json:"`
	idx := strings.Index(raw, key)
	if idx < 0 {
		return ""
	}
	rest := raw[idx+len(key):]
	if end := strings.IndexAny(rest, `",`); end > 0 {
		return rest[:end]
	}
	return ""
}

// ---- templates --------------------------------------------------------------

const fileHeader = `// Code generated by cmd/gen; DO NOT EDIT.

package qbittorrent
`

var mergeTmpl = template.Must(template.New("merge").Funcs(template.FuncMap{
	"lower": strings.ToLower,
}).Parse(`
{{ range .Structs }}
// mergePartial{{ .Name }} copies fields from src into dst using presence-aware
// semantics:
//   - pointer fields (*T): copied only when src value is non-nil; low-cardinality
//     string fields are interned via internStr to deduplicate backing arrays
//   - plain bool fields: always overwritten (false is a meaningful value)
//   - slice fields ([]T): copied only when src slice is non-nil and non-empty
//   - value fields (int, string, float): copied only when src value is non-zero
//
// Generated by cmd/gen — edit types.go then re-run go generate.
func mergePartial{{ .Name }}(dst, src *{{ .Name }}) {
{{ range .Fields }}
{{- if .IsPointer }}
	if src.{{ .Name }} != nil {
		{{- if .InternString }}
		{{- if eq .PointeeType "string" }}
		v := internStr(*src.{{ .Name }})
		dst.{{ .Name }} = &v
		{{- else }}
		v := {{ .PointeeType }}(internStr(string(*src.{{ .Name }})))
		dst.{{ .Name }} = &v
		{{- end }}
		{{- else }}
		dst.{{ .Name }} = src.{{ .Name }}
		{{- end }}
	}
{{- else if .IsSlice }}
	if len(src.{{ .Name }}) > 0 {
		dst.{{ .Name }} = src.{{ .Name }}
	}
{{- else if .AlwaysWrite }}
	dst.{{ .Name }} = src.{{ .Name }}
{{- else }}
	if src.{{ .Name }} != {{ .Zero }} {
		dst.{{ .Name }} = src.{{ .Name }}
	}
{{- end }}
{{ end -}}
}
{{ end }}
`))

var deepCopyTmpl = template.Must(template.New("deepcopy").Parse(`
{{ range .Structs }}
// deepCopy{{ .Name }} returns a pointer to a {{ .Name }} whose pointer fields are
// independently allocated so callers cannot mutate internal state through them.
// Returns nil if t is nil.
// Generated by cmd/gen — edit types.go then re-run go generate.
func deepCopy{{ .Name }}(t *{{ .Name }}) *{{ .Name }} {
	if t == nil {
		return nil
	}
	c := *t
{{ range .Fields }}
{{- if .IsPointer }}
	if c.{{ .Name }} != nil {
		{{- if .InternString }}
		{{- if eq .PointeeType "string" }}
		v := internStr(*c.{{ .Name }})
		{{- else }}
		v := {{ .PointeeType }}(internStr(string(*c.{{ .Name }})))
		{{- end }}
		{{- else }}
		v := *c.{{ .Name }}
		{{- end }}
		c.{{ .Name }} = &v
	}
{{ end }}
{{- end -}}
	return &c
}
{{ end }}
`))

// ---- codegen ----------------------------------------------------------------

// sortTarget is the struct for which sort comparison code is generated.
const sortTarget = "Torrent"

type structData struct {
	Name   string
	Fields []fieldDesc
}

// sortFieldDesc holds the data needed to generate a single sort comparison function.
type sortFieldDesc struct {
	Name       string // Go field name, e.g. "AddedOn"
	JSONTag    string // JSON tag name, e.g. "added_on"
	IsPointer  bool
	IsBool     bool
	Underlying string // e.g. "int64", "string", "float64", "bool"
}

var sortTmpl = template.Must(template.New("sort").Parse(`
import "sort"

// TorrentSort defines sort criteria for a torrent slice.
type TorrentSort struct {
	// Field is the torrent field to sort by. Use the JSON field names
	// that the qBittorrent API accepts (e.g. "name", "added_on", "size").
	// An empty or unrecognised field falls back to sorting by "hash".
	Field string

	// Reverse sorts in descending order when true.
	Reverse bool
}

// SortTorrents sorts a torrent slice in-place by the given criteria.
// An empty or unrecognised Field falls back to sorting by hash.
func SortTorrents(torrents []Torrent, opts TorrentSort) {
	s := &torrentSorter{data: torrents, cmp: torrentCmpFunc(opts.Field), less: lessAsc}
	if opts.Reverse {
		s.less = lessDesc
	}
	sort.Sort(s)
}

// torrentSorter implements sort.Interface. Pointer receiver avoids copying
// the struct on every Len/Less/Swap call through the interface.
type torrentSorter struct {
	data []Torrent
	cmp  func(a, b *Torrent) int
	less func(s *torrentSorter, i, j int) bool
}

func (s *torrentSorter) Len() int      { return len(s.data) }
func (s *torrentSorter) Swap(i, j int) { s.data[i], s.data[j] = s.data[j], s.data[i] }
func (s *torrentSorter) Less(i, j int) bool { return s.less(s, i, j) }
func lessAsc(s *torrentSorter, i, j int) bool  { return s.cmp(&s.data[i], &s.data[j]) < 0 }
func lessDesc(s *torrentSorter, i, j int) bool { return s.cmp(&s.data[i], &s.data[j]) > 0 }

// torrentCmpFunc returns a comparison function for the given sort field.
// Lookup is O(1) via map. Falls back to hash comparison for unrecognised fields.
func torrentCmpFunc(field string) func(a, b *Torrent) int {
	if fn, ok := torrentSortFields[field]; ok {
		return fn
	}
	return cmpTorrentHash
}

// cmpPtr compares two pointer values using native operators. nil sorts first.
// Single nil guard covers the rare case; hot path (both non-nil) is one branch.
// Equality is checked first because for strings == short-circuits on length
// mismatch (O(1)) before falling through to a single < for direction.
func cmpPtr[T interface{ ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64 | ~string }](a, b *T) int {
	if a == nil || b == nil {
		if a == b { return 0 }
		if a == nil { return -1 }
		return 1
	}
	if *a == *b { return 0 }
	if *a < *b { return -1 }
	return 1
}

// cmpPtrBool compares two *bool values. nil < false < true.
// Single nil guard covers the rare case; hot path (both non-nil) is one branch.
func cmpPtrBool(a, b *bool) int {
	if a == nil || b == nil {
		if a == b { return 0 }
		if a == nil { return -1 }
		return 1
	}
	if *a == *b { return 0 }
	if !*a { return -1 }
	return 1
}

// torrentSortFields maps JSON field names to named comparison functions.
// Generated by cmd/gen — edit types.go then re-run go generate.
var torrentSortFields = map[string]func(a, b *Torrent) int{
{{- range .Fields }}
	"{{ .JSONTag }}": cmpTorrent{{ .Name }},
{{- end }}
}
{{ range .Fields }}
func cmpTorrent{{ .Name }}(a, b *Torrent) int {
{{- if .IsBool }}
	return cmpPtrBool(a.{{ .Name }}, b.{{ .Name }})
{{- else }}
	return cmpPtr(a.{{ .Name }}, b.{{ .Name }})
{{- end }}
}
{{ end }}
`))

func render(tmpl *template.Template, structs []structData) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(fileHeader)
	if err := tmpl.Execute(&buf, map[string]any{"Structs": structs}); err != nil {
		return nil, err
	}
	return format.Source(buf.Bytes())
}

func renderSort(fields []sortFieldDesc) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(fileHeader)
	if err := sortTmpl.Execute(&buf, map[string]any{"Fields": fields}); err != nil {
		return nil, err
	}
	return format.Source(buf.Bytes())
}

func writeFile(path string, data []byte) {
	if err := os.WriteFile(path, data, 0o644); err != nil {
		log.Fatalf("write %s: %v", path, err)
	}
	fmt.Printf("wrote %s\n", path)
}

func main() {
	structs, err := parseStructs("types.go")
	if err != nil {
		log.Fatalf("parse types.go: %v", err)
	}

	// ── merge_gen.go ─────────────────────────────────────────────────────────
	var mergeData []structData
	for _, name := range mergeTargets {
		fields, ok := structs[name]
		if !ok {
			log.Fatalf("struct %q not found in types.go", name)
		}
		// Apply InternString annotations for low-cardinality string fields.
		internSet := internFields[name]
		if len(internSet) > 0 {
			marked := make([]fieldDesc, len(fields))
			copy(marked, fields)
			for i := range marked {
				if marked[i].IsPointer && marked[i].Underlying == "string" && internSet[marked[i].Name] {
					marked[i].InternString = true
				}
			}
			mergeData = append(mergeData, structData{Name: name, Fields: marked})
		} else {
			mergeData = append(mergeData, structData{Name: name, Fields: fields})
		}
	}
	mergeOut, err := render(mergeTmpl, mergeData)
	if err != nil {
		log.Fatalf("render merge: %v", err)
	}
	writeFile("merge_gen.go", mergeOut)

	// ── deepcopy_gen.go ───────────────────────────────────────────────────────
	// Include every struct that has at least one pointer field.
	var dcData []structData
	for _, name := range allStructNames(structs) {
		fields := structs[name]
		if !hasPointers(fields) {
			continue
		}
		// Apply InternString annotations for low-cardinality string fields.
		internSet := internFields[name]
		if len(internSet) > 0 {
			marked := make([]fieldDesc, len(fields))
			copy(marked, fields)
			for i := range marked {
				if marked[i].IsPointer && marked[i].Underlying == "string" && internSet[marked[i].Name] {
					marked[i].InternString = true
				}
			}
			dcData = append(dcData, structData{Name: name, Fields: marked})
		} else {
			dcData = append(dcData, structData{Name: name, Fields: fields})
		}
	}
	if len(dcData) == 0 {
		log.Fatal("no structs with pointer fields found")
	}
	dcOut, err := render(deepCopyTmpl, dcData)
	if err != nil {
		log.Fatalf("render deepcopy: %v", err)
	}
	writeFile("deepcopy_gen.go", dcOut)

	// ── sort_gen.go ──────────────────────────────────────────────────────────
	// Generate comparison functions for every sortable Torrent field.
	torrentFields, ok := structs[sortTarget]
	if !ok {
		log.Fatalf("struct %q not found in types.go", sortTarget)
	}
	var sortFields []sortFieldDesc
	for _, f := range torrentFields {
		if f.JSONTag == "" || f.JSONTag == "-" {
			continue // skip untagged or excluded fields
		}
		if !f.IsPointer {
			continue // only pointer fields are sortable (slices, maps, etc. are not)
		}
		sortFields = append(sortFields, sortFieldDesc{
			Name:       f.Name,
			JSONTag:    f.JSONTag,
			IsPointer:  f.IsPointer,
			IsBool:     f.IsBool,
			Underlying: f.Underlying,
		})
	}
	sortOut, err := renderSort(sortFields)
	if err != nil {
		log.Fatalf("render sort: %v", err)
	}
	writeFile("sort_gen.go", sortOut)
}

func hasPointers(fields []fieldDesc) bool {
	for _, f := range fields {
		if f.IsPointer {
			return true
		}
	}
	return false
}

// allStructNames returns struct names in a stable, declaration order.
// We preserve insertion order by re-parsing just for names.
func allStructNames(m map[string][]fieldDesc) []string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "types.go", nil, 0)
	if err != nil {
		log.Fatalf("allStructNames re-parse: %v", err)
	}
	var names []string
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if _, isStruct := ts.Type.(*ast.StructType); !isStruct {
				continue
			}
			if _, inMap := m[ts.Name.Name]; inMap {
				names = append(names, ts.Name.Name)
			}
		}
	}
	return names
}
