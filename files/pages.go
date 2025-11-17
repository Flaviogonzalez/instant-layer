package files

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/flaviogonzalez/instant-layer/factory"
	"github.com/flaviogonzalez/instant-layer/utils"
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
	var Routes []*ast.ExprStmt
	var AllowedHeaders = []string{"Accept", "Content-Type", "Authorization"}
	var AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	var AllowedOrigins = []string{"*"}
	var AllowCredentials = "false"
	var MaxAge = 30

	if _, ok := service.ServerType.(*API); ok {
		apiConfig := service.ServerType.(*API)

		if len(apiConfig.RoutesConfig.CORS.AllowedHeaders) > 0 {
			AllowedHeaders = apiConfig.RoutesConfig.CORS.AllowedHeaders
		}
		if len(apiConfig.RoutesConfig.CORS.AllowedMethods) > 0 {
			AllowedMethods = apiConfig.RoutesConfig.CORS.AllowedMethods
		}
		if len(apiConfig.RoutesConfig.CORS.AllowedOrigins) > 0 {
			AllowedOrigins = apiConfig.RoutesConfig.CORS.AllowedOrigins
		}

		if apiConfig.RoutesConfig.CORS.AllowCredentials {
			AllowCredentials = "true"
		}

		if apiConfig.RoutesConfig.CORS.MaxAge != 0 {
			MaxAge = apiConfig.RoutesConfig.CORS.MaxAge
		}

		for _, group := range apiConfig.RoutesConfig.RoutesGroup {
			var groupRoutes []ast.Stmt
			for _, route := range group.Routes {
				route.Method = strings.ToUpper(route.Method)
				route.Path = strings.ToLower(route.Path)
				route.Handler = utils.FirstLetterToUpper(route.Handler)

				routeStmt := factory.NewExprStmt(
					factory.NewSelectorCall(
						"r",
						route.Method,
						factory.NewBasicLit(route.Path),
						factory.NewSelector("handlers", route.Handler),
					),
				)
				groupRoutes = append(groupRoutes, routeStmt)
			}
			routeGroupStmt := factory.NewExprStmt(
				factory.NewSelectorCall(
					"mux",
					"Route",
					factory.NewBasicLit(group.Prefix),
					factory.NewFuncLit(
						factory.NewFuncType(
							factory.NewFieldList(
								factory.NewField("r", factory.NewSelector("chi", "Router")),
							),
							factory.NewFieldList(),
						),
						factory.NewBodyStmt(groupRoutes...),
					),
				),
			)
			Routes = append(Routes, routeGroupStmt)
		}
	}

	bodyStmts := []ast.Stmt{
		factory.NewDefine("mux", factory.NewSelectorCall("chi", "NewRouter")),
		factory.NewExprStmt(
			factory.NewSelectorCall("mux", "Use", factory.NewSelectorCall("cors", "Handler", factory.NewCompositeLit(
				factory.NewSelector("cors", "Optional"),
				factory.NewKeyValue("AllowedOrigins", factory.NewStringSliceLit(AllowedOrigins...)),
				factory.NewKeyValue("AllowedMethods", factory.NewStringSliceLit(AllowedMethods...)),
				factory.NewKeyValue("AllowedHeaders", factory.NewStringSliceLit(AllowedHeaders...)),
				factory.NewKeyValue("AllowCredentials", ast.NewIdent(AllowCredentials)),
				factory.NewKeyValue("MaxAge", factory.NewBasicLitInt(MaxAge)),
			))),
		),
		factory.NewExprStmt(factory.NewSelectorCall("mux", "Use", factory.NewSelectorCall("middleware", "Heartbeat", factory.NewBasicLit("/ping")))),
	}

	for _, route := range Routes {
		bodyStmts = append(bodyStmts, route)
	}

	bodyStmts = append(bodyStmts, &ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("mux")}})

	file := factory.NewFileNode(
		"routes",
		factory.NewImportDecl(
			factory.NewImport("net/http", ""),
			factory.NewImport("github.com/go-chi/chi/v5", ""),
			factory.NewImport("github.com/go-chi/chi/v5/middleware", ""),
			factory.NewImport("github.com/go-chi/cors", ""),
			factory.NewImport("database/sql", ""),
		),
		factory.NewFuncDecl(
			"Routes",
			factory.NewFieldList(),
			factory.NewFuncType( // functype
				factory.NewFieldList(factory.NewField("db", &ast.StarExpr{X: factory.NewSelector("sql", "DB")})),
				factory.NewFieldList(factory.NewField("", factory.NewSelector("http", "Handler"))),
			),
			factory.NewBodyStmt(bodyStmts...),
		),
	)
	return &File{
		Name: "routes.go",
		Data: file,
	}
}

func ConfigFile(service *Service, genconfig *GenConfig) *File {
	var ImportDrivers []*ast.ImportSpec
	var driver string
	var timeout int

	if _, ok := service.ServerType.(*API); ok {
		if service.ServerType.(*API).DB.Driver == "pgx" {
			ImportDrivers = []*ast.ImportSpec{
				factory.NewImport("github.com/jackc/pgconn", "_"),
				factory.NewImport("github.com/jackc/pgx/v5", "_"),
				factory.NewImport("github.com/jackc/pgx/v5/stdlib", "_"),
			}
		}

		driver = service.ServerType.(*API).DB.Driver
		timeout = service.ServerType.(*API).DB.TimeoutConn

		if service.ServerType.(*API).DB.TimeoutConn == 0 {
			timeout = 10
		}

		if service.ServerType.(*API).DB.Driver == "" {
			driver = "pgx"
		}
	}

	ImportDrivers = append(
		ImportDrivers,
		factory.NewImport(service.Name+"/routes", ""),
		factory.NewImport("database/sql", ""),
		factory.NewImport("log", ""),
		factory.NewImport("net/http", ""),
		factory.NewImport("os", ""),
		factory.NewImport("time", ""),
	)

	imports := factory.NewImportDecl(
		ImportDrivers...,
	)

	if service.Port == 0 {
		service.Port = 8080
	}

	file := factory.NewFileNode(
		"config",
		imports,
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
					factory.NewKeyValue("Addr", factory.NewBasicLit(":"+strconv.Itoa(service.Port))),
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
				factory.NewDefineExpectsError("db", factory.NewSelectorCall("sql", "Open", factory.NewBasicLit(driver), ast.NewIdent("dsn"))),
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
								Y:  factory.NewBasicLitInt(timeout),
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

func createHandlerFile(handlerName string) *File {
	handlerFunc := factory.NewFuncDecl(
		handlerName,
		factory.NewFieldList(),
		factory.NewFuncType( // func(w http.ResponseWriter, r *http.Request)
			factory.NewFieldList(
				factory.NewField("w", factory.NewSelector("http", "ResponseWriter")),
				factory.NewField("r", &ast.StarExpr{X: factory.NewSelector("http", "Request")}),
			),
			factory.NewFieldList(),
		),
		factory.NewBodyStmt(),
	)

	fileNode := factory.NewFileNode(
		"handlers",
		factory.NewImportDecl(
			factory.NewImport("net/http", ""),
		),
		handlerFunc,
	)

	return &File{
		Name: handlerName + "Handler.go", // "{}Handler.go"
		Data: fileNode,
	}
}
