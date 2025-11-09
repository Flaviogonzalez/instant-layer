package factory

import (
	"fmt"
	"go/ast"
	"go/token"
)

func NewFileNode(name string, decls ...ast.Decl) *ast.File {
	return &ast.File{
		Name:  ast.NewIdent(name),
		Decls: decls,
	}
}

func NewImport(path, alias string) *ast.ImportSpec {
	spec := &ast.ImportSpec{
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: "\"" + path + "\"",
		},
	}
	if alias != "" {
		spec.Name = ast.NewIdent(alias)
	}
	return spec
}

func NewImportDecl(imports ...*ast.ImportSpec) *ast.GenDecl {
	specs := make([]ast.Spec, len(imports))
	for i, v := range imports {
		specs[i] = v
	}

	return &ast.GenDecl{
		Tok: token.IMPORT,
		Lparen: func() token.Pos {
			if len(imports) > 1 {
				return 1
			}
			return 0
		}(),
		Specs: specs,
	}
}

func NewFuncDecl(name string, recv *ast.FieldList, functype *ast.FuncType, body *ast.BlockStmt) *ast.FuncDecl {
	newVar := &ast.FuncDecl{
		Name: ast.NewIdent(name),
		Type: functype,
		Body: body,
	}

	if len(recv.List) != 0 {
		newVar.Recv = recv
	}

	return newVar
}

func NewFuncType(params *ast.FieldList, results *ast.FieldList) *ast.FuncType {
	return &ast.FuncType{
		Params:  params,
		Results: results,
	}
}

func NewFieldList(fields ...*ast.Field) *ast.FieldList {
	if len(fields) == 0 {
		return &ast.FieldList{}
	}
	return &ast.FieldList{
		List: fields,
	}
}

func NewField(name string, typeExpr ast.Expr) *ast.Field {
	field := &ast.Field{
		Type: typeExpr,
	}
	if name != "" {
		field.Names = []*ast.Ident{ast.NewIdent(name)}
	}
	return field
}

func NewStructField(name string, typeExpr ast.Expr, tagContent string) *ast.Field {
	field := NewField(name, typeExpr)

	if tagContent != "" {
		field.Tag = &ast.BasicLit{
			Kind:  token.STRING,
			Value: "`" + tagContent + "`",
		}
	}
	return field
}

func NewSelector(x, sel string) *ast.SelectorExpr {
	return &ast.SelectorExpr{
		X:   ast.NewIdent(x),
		Sel: ast.NewIdent(sel),
	}
}

func NewCall(fun ast.Expr, args ...ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun:  fun,
		Args: args,
	}
}

func NewSelectorCall(x, sel string, args ...ast.Expr) *ast.CallExpr {
	return NewCall(NewSelector(x, sel), args...)
}

func NewDefine(lhs string, rhs ast.Expr) *ast.AssignStmt {
	return &ast.AssignStmt{
		Lhs: []ast.Expr{
			ast.NewIdent(lhs),
		},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{
			rhs,
		},
	}
}

func NewBodyStmt(stmts ...ast.Stmt) *ast.BlockStmt {
	return &ast.BlockStmt{
		List: stmts,
	}
}

func NewExprStmt(X ast.Expr) *ast.ExprStmt {
	return &ast.ExprStmt{
		X: X,
	}
}

func NewBasicLit(path string) *ast.BasicLit {
	return &ast.BasicLit{
		Kind:  token.STRING,
		Value: "\"" + path + "\"",
	}
}

func NewBasicLitInt(value int) *ast.BasicLit {
	return &ast.BasicLit{
		Kind:  token.INT,
		Value: fmt.Sprintf("%d", value),
	}
}

func NewStringSliceLit(values ...string) *ast.CompositeLit {
	elts := make([]ast.Expr, len(values))
	for i, v := range values {
		elts[i] = NewBasicLit(v)
	}

	return &ast.CompositeLit{
		Type: &ast.ArrayType{Elt: ast.NewIdent("string")},
		Elts: elts,
	}
}

func NewCompositeLit(typ ast.Expr, elts ...ast.Expr) *ast.CompositeLit {
	return &ast.CompositeLit{
		Type: typ,
		Elts: elts,
	}
}

func NewFuncLit(fun *ast.FuncType, body *ast.BlockStmt) *ast.FuncLit {
	return &ast.FuncLit{
		Type: fun,
		Body: body,
	}
}

func NewKeyValue(key string, value ast.Expr) *ast.KeyValueExpr {
	return &ast.KeyValueExpr{
		Key:   ast.NewIdent(key),
		Value: value,
	}
}

func NewVarDecl(name string, typ ast.Expr) *ast.GenDecl {
	return &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{
					ast.NewIdent(name),
				},
				Type: typ,
			},
		},
	}
}

func NewTypeStruct(name string, typ *ast.FieldList) *ast.GenDecl {
	return &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent(name),
				Type: &ast.StructType{
					Fields: typ,
				},
			},
		},
	}
}

func NewIfError(stmt ...ast.Stmt) *ast.IfStmt {
	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  ast.NewIdent("err"),
			Op: token.NEQ,
			Y:  ast.NewIdent("nil"),
		},
		Body: &ast.BlockStmt{
			List: stmt,
		},
	}
}

func NewReturn(values ...ast.Expr) *ast.ReturnStmt {
	return &ast.ReturnStmt{
		Results: values,
	}
}

// assign -> token: = | example: err = db.ping()
func NewAssignExpectsError(value ast.Expr) *ast.AssignStmt {
	return &ast.AssignStmt{
		Lhs: []ast.Expr{
			ast.NewIdent("err"),
		},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{
			value,
		},
	}
}

// define -> token: := | example: db, err := db.Open(...)
func NewDefineExpectsError(lhs string, rhs ast.Expr) *ast.AssignStmt {
	return &ast.AssignStmt{
		Lhs: []ast.Expr{
			ast.NewIdent(lhs),
			ast.NewIdent("err"),
		},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{
			rhs,
		},
	}
}

func NewStructDecl(name string, fields ...*ast.Field) *ast.GenDecl {
	return &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent(name),
				Type: &ast.StructType{
					Fields: &ast.FieldList{
						List: fields,
					},
				},
			},
		},
	}
}

func NewJsonField(name, typ, jsonTag string) *ast.Field {
	field := &ast.Field{
		Names: []*ast.Ident{ast.NewIdent(name)},
		Type:  ast.NewIdent(typ),
	}
	if jsonTag != "" {
		field.Tag = &ast.BasicLit{
			Kind:  token.STRING,
			Value: fmt.Sprintf("`json:\"%s\"`", jsonTag),
		}
	}
	return field
}

func NewJsonDecode(target ast.Expr, bodyExpr ast.Expr) *ast.AssignStmt {
	return &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent("err")},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{
			NewSelectorCall("json", "NewDecoder", bodyExpr),
			NewSelectorCall("", "Decode", &ast.UnaryExpr{Op: token.AND, X: target}),
		},
	}
}

func NewContextValue(key ast.Expr, typ ast.Expr) *ast.AssignStmt {
	return &ast.AssignStmt{
		Lhs: []ast.Expr{
			ast.NewIdent("db"),
			ast.NewIdent("ok"),
		},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{
			NewCall(
				NewSelector("r", "Context"),
				NewSelectorCall("", "Value", key),
			),
			NewCall(ast.NewIdent(""), &ast.StarExpr{X: typ}),
		},
	}
}

func NewBcryptCompare(hashExpr, plainExpr ast.Expr) *ast.AssignStmt {
	return &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent("err")},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{
			NewSelectorCall("bcrypt", "CompareHashAndPassword",
				NewCompositeLit(&ast.ArrayType{Elt: ast.NewIdent("byte")}, hashExpr),
				NewCompositeLit(&ast.ArrayType{Elt: ast.NewIdent("byte")}, plainExpr),
			),
		},
	}
}

func NewJwtNewWithClaims(method ast.Expr, claims ast.Expr) *ast.AssignStmt {
	return &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent("token")},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{
			NewSelectorCall("jwt", "NewWithClaims", method, claims),
		},
	}
}

func NewUuidNew() ast.Expr {
	return NewSelectorCall("uuid", "New")
}

func NewSetCookie(wExpr ast.Expr, cookie *ast.CompositeLit) *ast.ExprStmt {
	return NewExprStmt(NewSelectorCall("http", "SetCookie", wExpr, &ast.UnaryExpr{Op: token.AND, X: cookie}))
}

func NewErrorJson(wExpr ast.Expr, status int, msg ast.Expr) *ast.ExprStmt {
	return NewExprStmt(NewSelectorCall("helpers", "ErrorJSON", wExpr, NewBasicLitInt(status), msg))
}

func NewWriteJson(wExpr ast.Expr, status int, payload ast.Expr, headers ast.Expr) *ast.ExprStmt {
	return NewExprStmt(NewSelectorCall("helpers", "WriteJSON", wExpr, NewBasicLitInt(status), payload, headers))
}

func NewHttpHeader() ast.Expr {
	return NewCall(ast.NewIdent("make"), NewSelector("http", "Header"))
}

func CollectImports(usedPackages map[string]string) *ast.GenDecl {
	imports := make([]*ast.ImportSpec, 0)
	for path, alias := range usedPackages {
		imports = append(imports, NewImport(path, alias))
	}
	return NewImportDecl(imports...)
}
