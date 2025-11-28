package factory

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/printer"
	"go/token"
	"strings"
	"testing"
)

// renderNode renders an AST node to string for testing
func renderNode(node interface{}) string {
	var buf bytes.Buffer
	fset := token.NewFileSet()

	if err := printer.Fprint(&buf, fset, node); err != nil {
		return ""
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return buf.String()
	}
	return string(formatted)
}

// TestNewFileNode tests file node creation
func TestNewFileNode(t *testing.T) {
	file := NewFileNode("main",
		NewImportDecl(NewImport("fmt", "")),
	)

	if file.Name.Name != "main" {
		t.Errorf("File name = %q, want %q", file.Name.Name, "main")
	}
	if len(file.Decls) != 1 {
		t.Errorf("Decls = %d, want 1", len(file.Decls))
	}
}

// TestNewImport tests import spec creation
func TestNewImport(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		alias        string
		wantPath     string
		wantHasAlias bool
	}{
		{"simple import", "fmt", "", "\"fmt\"", false},
		{"with alias", "database/sql", "db", "\"database/sql\"", true},
		{"blank import", "github.com/lib/pq", "_", "\"github.com/lib/pq\"", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imp := NewImport(tt.path, tt.alias)

			if imp.Path.Value != tt.wantPath {
				t.Errorf("Path = %q, want %q", imp.Path.Value, tt.wantPath)
			}
			hasAlias := imp.Name != nil
			if hasAlias != tt.wantHasAlias {
				t.Errorf("HasAlias = %v, want %v", hasAlias, tt.wantHasAlias)
			}
			if tt.wantHasAlias && imp.Name.Name != tt.alias {
				t.Errorf("Alias = %q, want %q", imp.Name.Name, tt.alias)
			}
		})
	}
}

// TestNewImportDecl tests import declaration creation
func TestNewImportDecl(t *testing.T) {
	// Single import
	single := NewImportDecl(NewImport("fmt", ""))
	if single.Tok != token.IMPORT {
		t.Errorf("Tok = %v, want %v", single.Tok, token.IMPORT)
	}
	if single.Lparen != 0 {
		t.Error("Single import should not have parentheses")
	}

	// Multiple imports
	multi := NewImportDecl(
		NewImport("fmt", ""),
		NewImport("net/http", ""),
	)
	if multi.Lparen == 0 {
		t.Error("Multiple imports should have parentheses")
	}
	if len(multi.Specs) != 2 {
		t.Errorf("Specs = %d, want 2", len(multi.Specs))
	}
}

// TestNewFuncDecl tests function declaration creation
func TestNewFuncDecl(t *testing.T) {
	// Simple function
	fn := NewFuncDecl(
		"Hello",
		NewFieldList(),
		NewFuncType(NewFieldList(), NewFieldList()),
		NewBodyStmt(),
	)

	if fn.Name.Name != "Hello" {
		t.Errorf("Name = %q, want %q", fn.Name.Name, "Hello")
	}
	if fn.Recv != nil {
		t.Error("Simple function should not have receiver")
	}

	// Method with receiver
	method := NewFuncDecl(
		"Hello",
		NewFieldList(NewField("s", &ast.StarExpr{X: ast.NewIdent("Server")})),
		NewFuncType(NewFieldList(), NewFieldList()),
		NewBodyStmt(),
	)

	if method.Recv == nil {
		t.Error("Method should have receiver")
	}
}

// TestNewFuncType tests function type creation
func TestNewFuncType(t *testing.T) {
	ft := NewFuncType(
		NewFieldList(NewField("name", ast.NewIdent("string"))),
		NewFieldList(NewField("", ast.NewIdent("error"))),
	)

	if len(ft.Params.List) != 1 {
		t.Errorf("Params = %d, want 1", len(ft.Params.List))
	}
	if len(ft.Results.List) != 1 {
		t.Errorf("Results = %d, want 1", len(ft.Results.List))
	}
}

// TestNewFieldList tests field list creation
func TestNewFieldList(t *testing.T) {
	empty := NewFieldList()
	if len(empty.List) != 0 {
		t.Error("Empty field list should have 0 fields")
	}

	withFields := NewFieldList(
		NewField("a", ast.NewIdent("int")),
		NewField("b", ast.NewIdent("string")),
	)
	if len(withFields.List) != 2 {
		t.Errorf("Fields = %d, want 2", len(withFields.List))
	}
}

// TestNewField tests field creation
func TestNewField(t *testing.T) {
	// Named field
	named := NewField("count", ast.NewIdent("int"))
	if len(named.Names) != 1 || named.Names[0].Name != "count" {
		t.Error("Named field should have name 'count'")
	}

	// Unnamed field (for return types)
	unnamed := NewField("", ast.NewIdent("error"))
	if len(unnamed.Names) != 0 {
		t.Error("Unnamed field should have no names")
	}
}

// TestNewStructField tests struct field creation with tags
func TestNewStructField(t *testing.T) {
	field := NewStructField("Name", ast.NewIdent("string"), `json:"name"`)

	if len(field.Names) != 1 || field.Names[0].Name != "Name" {
		t.Error("Field should have name 'Name'")
	}
	if field.Tag == nil {
		t.Fatal("Field should have tag")
	}
	if !strings.Contains(field.Tag.Value, "json") {
		t.Error("Tag should contain 'json'")
	}
}

// TestNewSelector tests selector expression creation
func TestNewSelector(t *testing.T) {
	sel := NewSelector("http", "Handler")

	ident, ok := sel.X.(*ast.Ident)
	if !ok || ident.Name != "http" {
		t.Errorf("X = %v, want 'http'", sel.X)
	}
	if sel.Sel.Name != "Handler" {
		t.Errorf("Sel = %q, want %q", sel.Sel.Name, "Handler")
	}
}

// TestNewCall tests call expression creation
func TestNewCall(t *testing.T) {
	call := NewCall(ast.NewIdent("println"), NewBasicLit("hello"))

	if len(call.Args) != 1 {
		t.Errorf("Args = %d, want 1", len(call.Args))
	}
}

// TestNewSelectorCall tests selector call creation
func TestNewSelectorCall(t *testing.T) {
	call := NewSelectorCall("fmt", "Println", NewBasicLit("hello"))

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		t.Fatal("Fun should be SelectorExpr")
	}
	if sel.Sel.Name != "Println" {
		t.Errorf("Sel = %q, want %q", sel.Sel.Name, "Println")
	}
}

// TestNewDefine tests define statement creation
func TestNewDefine(t *testing.T) {
	def := NewDefine("x", NewBasicLitInt(42))

	if def.Tok != token.DEFINE {
		t.Errorf("Tok = %v, want %v", def.Tok, token.DEFINE)
	}
	if len(def.Lhs) != 1 {
		t.Errorf("Lhs = %d, want 1", len(def.Lhs))
	}
}

// TestNewBasicLit tests basic literal creation
func TestNewBasicLit(t *testing.T) {
	str := NewBasicLit("hello")
	if str.Kind != token.STRING {
		t.Errorf("Kind = %v, want %v", str.Kind, token.STRING)
	}
	if str.Value != `"hello"` {
		t.Errorf("Value = %q, want %q", str.Value, `"hello"`)
	}
}

// TestNewBasicLitInt tests integer literal creation
func TestNewBasicLitInt(t *testing.T) {
	num := NewBasicLitInt(42)
	if num.Kind != token.INT {
		t.Errorf("Kind = %v, want %v", num.Kind, token.INT)
	}
	if num.Value != "42" {
		t.Errorf("Value = %q, want %q", num.Value, "42")
	}
}

// TestNewStringSliceLit tests string slice literal creation
func TestNewStringSliceLit(t *testing.T) {
	slice := NewStringSliceLit("a", "b", "c")

	if len(slice.Elts) != 3 {
		t.Errorf("Elts = %d, want 3", len(slice.Elts))
	}

	arr, ok := slice.Type.(*ast.ArrayType)
	if !ok {
		t.Fatal("Type should be ArrayType")
	}
	elt, ok := arr.Elt.(*ast.Ident)
	if !ok || elt.Name != "string" {
		t.Error("Element type should be 'string'")
	}
}

// TestNewCompositeLit tests composite literal creation
func TestNewCompositeLit(t *testing.T) {
	comp := NewCompositeLit(
		ast.NewIdent("Config"),
		NewKeyValue("Port", NewBasicLitInt(8080)),
	)

	if len(comp.Elts) != 1 {
		t.Errorf("Elts = %d, want 1", len(comp.Elts))
	}
}

// TestNewKeyValue tests key-value expression creation
func TestNewKeyValue(t *testing.T) {
	kv := NewKeyValue("Port", NewBasicLitInt(8080))

	key, ok := kv.Key.(*ast.Ident)
	if !ok || key.Name != "Port" {
		t.Errorf("Key = %v, want 'Port'", kv.Key)
	}
}

// TestNewVarDecl tests variable declaration creation
func TestNewVarDecl(t *testing.T) {
	varDecl := NewVarDecl("count", ast.NewIdent("int"))

	if varDecl.Tok != token.VAR {
		t.Errorf("Tok = %v, want %v", varDecl.Tok, token.VAR)
	}
	if len(varDecl.Specs) != 1 {
		t.Errorf("Specs = %d, want 1", len(varDecl.Specs))
	}
}

// TestNewTypeStruct tests type struct declaration creation
func TestNewTypeStruct(t *testing.T) {
	structDecl := NewTypeStruct("Config",
		NewFieldList(
			NewField("Port", ast.NewIdent("int")),
			NewField("Host", ast.NewIdent("string")),
		),
	)

	if structDecl.Tok != token.TYPE {
		t.Errorf("Tok = %v, want %v", structDecl.Tok, token.TYPE)
	}

	spec := structDecl.Specs[0].(*ast.TypeSpec)
	if spec.Name.Name != "Config" {
		t.Errorf("Name = %q, want %q", spec.Name.Name, "Config")
	}

	structType := spec.Type.(*ast.StructType)
	if len(structType.Fields.List) != 2 {
		t.Errorf("Fields = %d, want 2", len(structType.Fields.List))
	}
}

// TestNewIfError tests if error statement creation
func TestNewIfError(t *testing.T) {
	ifErr := NewIfError(
		NewReturn(ast.NewIdent("nil"), ast.NewIdent("err")),
	)

	binExpr, ok := ifErr.Cond.(*ast.BinaryExpr)
	if !ok {
		t.Fatal("Cond should be BinaryExpr")
	}
	if binExpr.Op != token.NEQ {
		t.Errorf("Op = %v, want %v", binExpr.Op, token.NEQ)
	}

	xIdent, ok := binExpr.X.(*ast.Ident)
	if !ok || xIdent.Name != "err" {
		t.Error("X should be 'err'")
	}
}

// TestNewReturn tests return statement creation
func TestNewReturn(t *testing.T) {
	// No values
	empty := NewReturn()
	if len(empty.Results) != 0 {
		t.Error("Empty return should have no results")
	}

	// With values
	withValues := NewReturn(ast.NewIdent("nil"), ast.NewIdent("err"))
	if len(withValues.Results) != 2 {
		t.Errorf("Results = %d, want 2", len(withValues.Results))
	}
}

// TestNewAssignExpectsError tests error assignment creation
func TestNewAssignExpectsError(t *testing.T) {
	assign := NewAssignExpectsError(NewSelectorCall("db", "Ping"))

	if assign.Tok != token.ASSIGN {
		t.Errorf("Tok = %v, want %v", assign.Tok, token.ASSIGN)
	}
	if len(assign.Lhs) != 1 {
		t.Errorf("Lhs = %d, want 1", len(assign.Lhs))
	}
	lhs, ok := assign.Lhs[0].(*ast.Ident)
	if !ok || lhs.Name != "err" {
		t.Error("Lhs should be 'err'")
	}
}

// TestNewDefineExpectsError tests define with error creation
func TestNewDefineExpectsError(t *testing.T) {
	define := NewDefineExpectsError("db", NewSelectorCall("sql", "Open", NewBasicLit("postgres")))

	if define.Tok != token.DEFINE {
		t.Errorf("Tok = %v, want %v", define.Tok, token.DEFINE)
	}
	if len(define.Lhs) != 2 {
		t.Errorf("Lhs = %d, want 2", len(define.Lhs))
	}

	first, ok := define.Lhs[0].(*ast.Ident)
	if !ok || first.Name != "db" {
		t.Error("First Lhs should be 'db'")
	}
	second, ok := define.Lhs[1].(*ast.Ident)
	if !ok || second.Name != "err" {
		t.Error("Second Lhs should be 'err'")
	}
}

// TestNewStructDecl tests struct declaration creation
func TestNewStructDecl(t *testing.T) {
	decl := NewStructDecl("User",
		NewField("ID", ast.NewIdent("int")),
		NewField("Name", ast.NewIdent("string")),
	)

	if decl.Tok != token.TYPE {
		t.Errorf("Tok = %v, want %v", decl.Tok, token.TYPE)
	}

	spec := decl.Specs[0].(*ast.TypeSpec)
	if spec.Name.Name != "User" {
		t.Errorf("Name = %q, want %q", spec.Name.Name, "User")
	}
}

// TestNewJsonField tests JSON field creation
func TestNewJsonField(t *testing.T) {
	field := NewJsonField("Email", "string", "email,omitempty")

	if len(field.Names) != 1 || field.Names[0].Name != "Email" {
		t.Error("Field should have name 'Email'")
	}
	if field.Tag == nil {
		t.Fatal("Field should have tag")
	}
	if !strings.Contains(field.Tag.Value, `json:"email,omitempty"`) {
		t.Errorf("Tag = %s, should contain 'json:\"email,omitempty\"'", field.Tag.Value)
	}
}

// TestNewBodyStmt tests block statement creation
func TestNewBodyStmt(t *testing.T) {
	empty := NewBodyStmt()
	if len(empty.List) != 0 {
		t.Error("Empty body should have no statements")
	}

	withStmts := NewBodyStmt(
		NewExprStmt(NewSelectorCall("fmt", "Println")),
		NewReturn(),
	)
	if len(withStmts.List) != 2 {
		t.Errorf("Statements = %d, want 2", len(withStmts.List))
	}
}

// TestNewExprStmt tests expression statement creation
func TestNewExprStmt(t *testing.T) {
	stmt := NewExprStmt(NewSelectorCall("fmt", "Println"))

	if stmt.X == nil {
		t.Error("ExprStmt should have X")
	}
}

// TestNewFuncLit tests function literal creation
func TestNewFuncLit(t *testing.T) {
	fn := NewFuncLit(
		NewFuncType(NewFieldList(), NewFieldList()),
		NewBodyStmt(NewReturn()),
	)

	if fn.Type == nil {
		t.Error("FuncLit should have Type")
	}
	if fn.Body == nil {
		t.Error("FuncLit should have Body")
	}
}

// TestCollectImports tests import collection
func TestCollectImports(t *testing.T) {
	packages := map[string]string{
		"fmt":      "",
		"net/http": "",
		"os":       "",
	}

	imports := CollectImports(packages)

	if imports.Tok != token.IMPORT {
		t.Errorf("Tok = %v, want %v", imports.Tok, token.IMPORT)
	}
	if len(imports.Specs) != 3 {
		t.Errorf("Specs = %d, want 3", len(imports.Specs))
	}
}

// TestRenderableOutput tests that generated AST can be rendered to valid Go code
func TestRenderableOutput(t *testing.T) {
	file := NewFileNode("main",
		NewImportDecl(
			NewImport("fmt", ""),
		),
		NewFuncDecl(
			"main",
			NewFieldList(),
			NewFuncType(NewFieldList(), NewFieldList()),
			NewBodyStmt(
				NewExprStmt(NewSelectorCall("fmt", "Println", NewBasicLit("Hello, World!"))),
			),
		),
	)

	output := renderNode(file)

	if !strings.Contains(output, "package main") {
		t.Error("Output should contain 'package main'")
	}
	if !strings.Contains(output, "import") {
		t.Error("Output should contain 'import'")
	}
	if !strings.Contains(output, "func main()") {
		t.Error("Output should contain 'func main()'")
	}
	if !strings.Contains(output, "fmt.Println") {
		t.Error("Output should contain 'fmt.Println'")
	}
}
