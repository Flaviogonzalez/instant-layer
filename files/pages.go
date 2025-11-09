package files

import (
	"go/ast"
	"go/token"
	"instant-layer/factory"
)

func MainFile(service *Service, genconfig *GenConfig) *File {
	imports := factory.CollectImports(map[string]string{
		service.Name + "/config": "",
	})
	file := factory.NewFileNode(
		"main",
		imports,
		factory.NewFuncDecl(
			"main",
			factory.NewFieldList(),
			factory.NewFuncType(
				factory.NewFieldList(),
				factory.NewFieldList(),
			),
			factory.NewBodyStmt(
				factory.NewExprStmt(factory.NewCall(&ast.SelectorExpr{
					X:   factory.NewSelectorCall("config", "InitConfig"),
					Sel: ast.NewIdent("InitServer"),
				})),
			),
		),
	)
	return &File{
		Name: "main.go",
		Data: file,
	}
}

func RoutesFile(service *Service, genconfig *GenConfig) *File {
	file := factory.NewFileNode(
		"routes",
		factory.NewImportDecl(
			factory.NewImport("net/http", ""),
			factory.NewImport("github.com/go-chi/chi/v5", ""),
			factory.NewImport("github.com/go-chi/chi/v5/middleware", ""),
			factory.NewImport("github.com/go-chi/cors", ""),
		),
		factory.NewFuncDecl(
			"Routes",
			factory.NewFieldList(),
			factory.NewFuncType( // functype
				factory.NewFieldList(factory.NewField("db", &ast.StarExpr{X: factory.NewSelector("sql", "DB")})),
				factory.NewFieldList(factory.NewField("", factory.NewSelector("http", "Handler"))),
			),
			factory.NewBodyStmt(
				factory.NewDefine("mux", factory.NewSelectorCall("chi", "NewRouter")),
				factory.NewExprStmt(
					factory.NewSelectorCall("mux", "Use", factory.NewSelectorCall("cors", "Handler", factory.NewCompositeLit(
						factory.NewSelector("cors", "Optional"),
						factory.NewKeyValue("AllowedOrigins", factory.NewStringSliceLit("https://*", "http://*")),
						factory.NewKeyValue("AllowedMethods", factory.NewStringSliceLit("POST", "GET", "DELETE", "PUT", "OPTIONS")),
						factory.NewKeyValue("AllowedHeaders", factory.NewStringSliceLit("Accept", "Content-Type", "Authorization")),
						factory.NewKeyValue("AllowCredentials", ast.NewIdent("true")),
						factory.NewKeyValue("MaxAge", factory.NewBasicLitInt(30)),
					))),
				),
				factory.NewExprStmt(factory.NewSelectorCall("mux", "Use", factory.NewSelectorCall("middleware", "Heartbeat", factory.NewBasicLit("/ping")))),
				factory.NewExprStmt(factory.NewSelectorCall("mux", "Post", factory.NewBasicLit("/authenticate"), factory.NewSelector("handlers", "Authenticate"))),
				&ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("mux")}},
			),
		),
	)
	return &File{
		Name: "routes.go",
		Data: file,
	}
}

func ConfigFile(service *Service, genconfig *GenConfig) *File {
	file := factory.NewFileNode(
		"config",
		factory.NewImportDecl(
			factory.NewImport("auth-service/routes", ""),
			factory.NewImport("database/sql", ""),
			factory.NewImport("log", ""),
			factory.NewImport("net/http", ""),
			factory.NewImport("os", ""),
			factory.NewImport("time", ""),
			factory.NewImport("github.com/jackc/pgconn", "_"),
			factory.NewImport("github.com/jackc/pgx/v5", "_"),
			factory.NewImport("github.com/jackc/pgx/v5/stdlib", "_"),
		),
		factory.NewVarDecl("counts", ast.NewIdent("int")),
		factory.NewTypeStruct("Config", factory.NewFieldList(factory.NewField("Db", &ast.StarExpr{X: factory.NewSelector("sql", "DB")}))),
		factory.NewFuncDecl(
			"InitConfig",
			factory.NewFieldList(),
			factory.NewFuncType(
				factory.NewFieldList(),
				factory.NewFieldList(factory.NewField("", &ast.StarExpr{X: ast.NewIdent("Config")})),
			),
			factory.NewBodyStmt(
				factory.NewDefine("db", factory.NewCall(ast.NewIdent("connectToDB"))),
				factory.NewReturn(
					&ast.UnaryExpr{
						Op: token.AND,
						X: factory.NewCompositeLit(
							ast.NewIdent("Config"),
							factory.NewKeyValue("Db", ast.NewIdent("db")),
						),
					},
				),
			),
		),
		factory.NewFuncDecl(
			"InitServer",
			factory.NewFieldList(
				factory.NewField(
					"app",
					&ast.StarExpr{X: ast.NewIdent("Config")},
				),
			),
			factory.NewFuncType(
				factory.NewFieldList(),
				factory.NewFieldList(),
			),
			factory.NewBodyStmt(
				factory.NewDefine("server", &ast.UnaryExpr{Op: token.AND, X: factory.NewCompositeLit(
					ast.NewIdent("http.Server"),
					factory.NewKeyValue("Addr", factory.NewBasicLit(":80")),
					factory.NewKeyValue("Handler", factory.NewSelectorCall("routes", "Routes", factory.NewSelector("app", "Db"))),
				)}),
				&ast.IfStmt{
					Init: factory.NewDefine(
						"err",
						factory.NewSelectorCall("server", "ListenAndServe"),
					),
					Cond: &ast.BinaryExpr{
						X:  ast.NewIdent("err"),
						Op: token.NEQ,
						Y:  ast.NewIdent("nil"),
					},
					Body: factory.NewBodyStmt(
						factory.NewExprStmt(factory.NewSelectorCall("log", "Fatal", ast.NewIdent("err"))),
					),
				},
			),
		),
		factory.NewFuncDecl(
			"openDB",
			factory.NewFieldList(),
			factory.NewFuncType(
				factory.NewFieldList(factory.NewField("dsn", ast.NewIdent("string"))),
				factory.NewFieldList(
					factory.NewField("", &ast.StarExpr{X: factory.NewSelector("sql", "DB")}),
					factory.NewField("", ast.NewIdent("error")),
				),
			),
			factory.NewBodyStmt(
				factory.NewDefineExpectsError("db", factory.NewSelectorCall("sql", "Open", factory.NewBasicLit("pgx"), ast.NewIdent("dsn"))),
				factory.NewIfError(
					factory.NewReturn(
						ast.NewIdent("nil"),
						ast.NewIdent("err"),
					),
				),
				factory.NewAssignExpectsError(factory.NewSelectorCall("db", "Ping")),
				factory.NewIfError(
					factory.NewReturn(
						ast.NewIdent("nil"),
						ast.NewIdent("err"),
					),
				),
				factory.NewReturn(
					ast.NewIdent("db"),
					ast.NewIdent("nil"),
				),
			),
		),
		factory.NewFuncDecl(
			"connectToDB",
			factory.NewFieldList(),
			factory.NewFuncType(
				factory.NewFieldList(),
				factory.NewFieldList(factory.NewField("", &ast.StarExpr{X: factory.NewSelector("sql", "DB")})),
			),
			factory.NewBodyStmt(
				factory.NewDefine("dsn", factory.NewSelectorCall("os", "Getenv", factory.NewBasicLit("DATABASE_URL"))),
				&ast.ForStmt{
					Body: factory.NewBodyStmt(
						factory.NewDefineExpectsError("conn", factory.NewCall(ast.NewIdent("openDB"), ast.NewIdent("dsn"))),
						&ast.IfStmt{
							Cond: &ast.BinaryExpr{
								X:  ast.NewIdent("err"),
								Op: token.NEQ,
								Y:  ast.NewIdent("nil"),
							},
							Body: factory.NewBodyStmt(
								factory.NewExprStmt(factory.NewSelectorCall("log", "Println", factory.NewBasicLit("postgres no parece estar listo..."))),
								factory.NewExprStmt(ast.NewIdent("counts++")),
							),
							Else: factory.NewBodyStmt(
								factory.NewExprStmt(factory.NewSelectorCall("log", "Println", factory.NewBasicLit("postgres conectado"))),
								factory.NewReturn(ast.NewIdent("conn")),
							),
						},
						&ast.IfStmt{
							Cond: &ast.BinaryExpr{
								X:  ast.NewIdent("counts"),
								Op: token.GTR,
								Y:  factory.NewBasicLitInt(10),
							},
							Body: factory.NewBodyStmt(
								factory.NewExprStmt(factory.NewSelectorCall("log", "Println", factory.NewBasicLit("err"))),
								factory.NewReturn(ast.NewIdent("nil")),
							),
						},
						factory.NewExprStmt(factory.NewSelectorCall("log", "Println", factory.NewBasicLit("esperando por dos segundos"))),
						factory.NewExprStmt(factory.NewSelectorCall("time", "Sleep", &ast.BinaryExpr{X: factory.NewBasicLitInt(2), Op: token.MUL, Y: factory.NewSelector("time", "Second")})),
						factory.NewExprStmt(ast.NewIdent("continue")),
					),
				},
			),
		),
	)
	return &File{
		Name: "config.go",
		Data: file,
	}
}
