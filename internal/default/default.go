package defaults

import (
	"go/ast"
	"go/token"

	"github.com/flaviogonzalez/instant-layer/internal/factory"
	"github.com/flaviogonzalez/instant-layer/internal/types"
)

type Template struct {
	ID          string
	Name        string
	Description string
	Service     *types.Service
}

var AvailableTemplates = []*Template{
	{
		ID:          "auth",
		Name:        "auth-service",
		Description: "preconfigured auth-service with the following routes: /login, /logout, /register (/send if email-service available)",
		Service: DefaultService(
			WithName("auth-service"),
			WithPort(8080),
		),
	},
	{
		ID:          "custom",
		Name:        "custom api-service",
		Description: "configurable scaffolding types.",
		Service: DefaultService(
			WithName("custom-api-service"),
			WithPort(8081),
		),
	},
	{
		ID:          "broker",
		Name:        "broker-service",
		Description: "preconfigured broker-service with no connections.",
		Service: DefaultService(
			WithName("broker-service"),
			WithPort(8082),
		),
	},
	{
		ID:          "listener",
		Name:        "listener-service",
		Description: "preconfigured listener-types.",
		Service: DefaultService(
			WithName("listener-service"),
			WithPort(8083),
		),
	},
}

func DefaultService(opts ...Option) *types.Service {
	s := &types.Service{Port: 8080}

	// Apply user options first (like WithName) so s.Name is set
	// before generating files that depend on it
	for _, o := range opts {
		o(s)
	}

	// Now apply base options that generate files using s.Name
	baseOpts := []Option{
		WithPostgres(),
		WithMain(),
		WithRoutes(),
	}

	return applyOptions(s, baseOpts...)
}

func DefaultConfigFile(s *types.Service) *types.File {
	// Base imports (always included)
	importSpecs := []*ast.ImportSpec{
		factory.NewImport(s.Name+"/routes", ""),
		factory.NewImport("database/sql", ""),
		factory.NewImport("log", ""),
		factory.NewImport("net/http", ""),
		factory.NewImport("os", ""),
		factory.NewImport("time", ""),
	}

	// Driver name for sql.Open
	var driverName string

	// Only pgx supported for now
	if s.DB != nil && s.DB.Driver == "pgx" {
		driverName = "pgx"
		importSpecs = append(importSpecs,
			factory.NewImport("github.com/jackc/pgconn", "_"),
			factory.NewImport("github.com/jackc/pgx/v5", "_"),
			factory.NewImport("github.com/jackc/pgx/v5/stdlib", "_"),
		)
	}

	imports := factory.NewImportDecl(importSpecs...)

	// var counts int
	countsVar := factory.NewVarDecl("counts", ast.NewIdent("int"))

	// type Config struct { Db *sql.DB }
	configStruct := factory.NewTypeStruct("Config", factory.NewFieldList(
		factory.NewField("Db", &ast.StarExpr{X: factory.NewSelector("sql", "DB")}),
	))

	// func InitConfig() *Config
	initConfigFunc := factory.NewFuncDecl(
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
	)

	// func (app *Config) InitServer()
	initServerFunc := factory.NewFuncDecl(
		"InitServer",
		factory.NewFieldList(
			factory.NewField("app", &ast.StarExpr{X: ast.NewIdent("Config")}),
		),
		factory.NewFuncType(factory.NewFieldList(), factory.NewFieldList()),
		factory.NewBodyStmt(
			factory.NewDefine("server",
				&ast.UnaryExpr{
					Op: token.AND,
					X: factory.NewCompositeLit(
						factory.NewSelector("http", "Server"),
						factory.NewKeyValue("Addr", factory.NewBasicLit(":80")),
						factory.NewKeyValue("Handler",
							factory.NewSelectorCall("routes", "Routes",
								factory.NewSelector("app", "Db"),
							),
						),
					),
				},
			),
			factory.NewIfError(
				factory.NewExprStmt(factory.NewSelectorCall("log", "Fatal", ast.NewIdent("err"))),
			),
		),
	)

	// Patch: the if needs init statement
	initServerFunc.Body.List[1] = &ast.IfStmt{
		Init: factory.NewAssignExpectsError(factory.NewSelectorCall("server", "ListenAndServe")),
		Cond: &ast.BinaryExpr{
			X:  ast.NewIdent("err"),
			Op: token.NEQ,
			Y:  ast.NewIdent("nil"),
		},
		Body: factory.NewBodyStmt(
			factory.NewExprStmt(factory.NewSelectorCall("log", "Fatal", ast.NewIdent("err"))),
		),
	}

	// func openDB(dsn string) (*sql.DB, error)
	openDBFunc := factory.NewFuncDecl(
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
			factory.NewDefineExpectsError("db",
				factory.NewSelectorCall("sql", "Open",
					factory.NewBasicLit(driverName),
					ast.NewIdent("dsn"),
				),
			),
			factory.NewIfError(
				factory.NewReturn(ast.NewIdent("nil"), ast.NewIdent("err")),
			),
			factory.NewAssignExpectsError(factory.NewSelectorCall("db", "Ping")),
			factory.NewIfError(
				factory.NewReturn(ast.NewIdent("nil"), ast.NewIdent("err")),
			),
			factory.NewReturn(ast.NewIdent("db"), ast.NewIdent("nil")),
		),
	)

	// func connectToDB() *sql.DB
	connectToDBFunc := factory.NewFuncDecl(
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
							&ast.IncDecStmt{X: ast.NewIdent("counts"), Tok: token.INC},
						),
						Else: factory.NewBodyStmt(
							factory.NewExprStmt(factory.NewSelectorCall("log", "Println", factory.NewBasicLit("conectado a postgres"))),
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
							factory.NewExprStmt(factory.NewSelectorCall("log", "Println", ast.NewIdent("err"))),
							factory.NewReturn(ast.NewIdent("nil")),
						),
					},
					factory.NewExprStmt(factory.NewSelectorCall("log", "Println", factory.NewBasicLit("esperando por dos segundos"))),
					factory.NewExprStmt(factory.NewSelectorCall("time", "Sleep",
						&ast.BinaryExpr{
							X:  factory.NewBasicLitInt(2),
							Op: token.MUL,
							Y:  factory.NewSelector("time", "Second"),
						},
					)),
					&ast.BranchStmt{Tok: token.CONTINUE},
				),
			},
		),
	)

	return &types.File{
		Name: "config.go",
		Content: factory.NewFileNode("config",
			imports,
			countsVar,
			configStruct,
			initConfigFunc,
			initServerFunc,
			openDBFunc,
			connectToDBFunc,
		),
	}
}

func DefaultMainFile(s *types.Service) *types.File {
	// import "{service-name}/config"
	imports := factory.NewImportDecl(
		factory.NewImport(s.Name+"/config", ""),
	)

	// func main() { config.InitConfig().InitServer() }
	mainFunc := factory.NewFuncDecl(
		"main",
		factory.NewFieldList(),
		factory.NewFuncType(factory.NewFieldList(), factory.NewFieldList()),
		factory.NewBodyStmt(
			factory.NewExprStmt(
				factory.NewCall(&ast.SelectorExpr{
					X: factory.NewCall(&ast.SelectorExpr{
						X:   ast.NewIdent("config"),
						Sel: ast.NewIdent("InitConfig"),
					}),
					Sel: ast.NewIdent("InitServer"),
				}),
			),
		),
	)

	return &types.File{
		Name:    "main.go",
		Content: factory.NewFileNode("main", imports, mainFunc),
	}
}

func DefaultRoutesFile(s *types.Service) *types.File {
	// Build imports dynamically
	importSpecs := []*ast.ImportSpec{
		factory.NewImport(s.Name+"/handlers", ""),
		factory.NewImport("database/sql", ""),
		factory.NewImport("net/http", ""),
		factory.NewImport("github.com/go-chi/chi/v5", ""),
		factory.NewImport("github.com/go-chi/chi/v5/middleware", ""),
		factory.NewImport("github.com/go-chi/cors", ""),
	}

	// Add middleware import if there are route groups (assumes AuthMiddleware pattern)
	if s.RoutesConfig != nil && len(s.RoutesConfig.RoutesGroup) > 0 {
		importSpecs = append(importSpecs,
			factory.NewImport(s.Name+"/middleware", "AuthMiddleware"),
		)
	}

	imports := factory.NewImportDecl(importSpecs...)

	// Build function body statements
	bodyStmts := []ast.Stmt{
		// mux := chi.NewRouter()
		factory.NewDefine("mux", factory.NewSelectorCall("chi", "NewRouter")),
		// mux.Use(middleware.Logger)
		factory.NewExprStmt(
			factory.NewSelectorCall("mux", "Use",
				factory.NewSelector("middleware", "Logger"),
			),
		),
	}

	// Add CORS middleware if configured
	if s.RoutesConfig != nil && s.RoutesConfig.CORS != nil {
		cors := s.RoutesConfig.CORS

		// Build cors.Options composite literal
		corsElts := []ast.Expr{}

		if len(cors.AllowedOrigins) > 0 {
			corsElts = append(corsElts,
				factory.NewKeyValue("AllowedOrigins", factory.NewStringSliceLit(cors.AllowedOrigins...)),
			)
		}
		if len(cors.AllowedMethods) > 0 {
			corsElts = append(corsElts,
				factory.NewKeyValue("AllowedMethods", factory.NewStringSliceLit(cors.AllowedMethods...)),
			)
		}
		if len(cors.AllowedHeaders) > 0 {
			corsElts = append(corsElts,
				factory.NewKeyValue("AllowedHeaders", factory.NewStringSliceLit(cors.AllowedHeaders...)),
			)
		}
		if cors.AllowCredentials {
			corsElts = append(corsElts,
				factory.NewKeyValue("AllowCredentials", ast.NewIdent("true")),
			)
		}
		if cors.MaxAge > 0 {
			corsElts = append(corsElts,
				factory.NewKeyValue("MaxAge", factory.NewBasicLitInt(cors.MaxAge)),
			)
		}

		// mux.Use(cors.Handler(cors.Options{...}))
		bodyStmts = append(bodyStmts,
			factory.NewExprStmt(
				factory.NewSelectorCall("mux", "Use",
					factory.NewSelectorCall("cors", "Handler",
						factory.NewCompositeLit(
							factory.NewSelector("cors", "Options"),
							corsElts...,
						),
					),
				),
			),
		)
	}

	// Add AuthMiddleware.HandlerWrapper(db) as global middleware
	if s.RoutesConfig != nil && len(s.RoutesConfig.RoutesGroup) > 0 {
		bodyStmts = append(bodyStmts,
			factory.NewExprStmt(
				factory.NewSelectorCall("mux", "Use",
					factory.NewSelectorCall("AuthMiddleware", "HandlerWrapper",
						ast.NewIdent("db"),
					),
				),
			),
		)
	}

	// Add routes from RoutesConfig
	if s.RoutesConfig != nil {
		for _, group := range s.RoutesConfig.RoutesGroup {
			// Routes without middleware go directly on mux
			for _, route := range group.Routes {
				methodCall := routeMethodCall("mux", route.Method, route.Path, route.Handler)
				if methodCall != nil {
					bodyStmts = append(bodyStmts, factory.NewExprStmt(methodCall))
				}
			}
		}
	}

	// return mux
	bodyStmts = append(bodyStmts, factory.NewReturn(ast.NewIdent("mux")))

	// func Routes(db *sql.DB) http.Handler
	routesFunc := factory.NewFuncDecl(
		"Routes",
		factory.NewFieldList(),
		factory.NewFuncType(
			factory.NewFieldList(
				factory.NewField("db", &ast.StarExpr{X: factory.NewSelector("sql", "DB")}),
			),
			factory.NewFieldList(
				factory.NewField("", factory.NewSelector("http", "Handler")),
			),
		),
		factory.NewBodyStmt(bodyStmts...),
	)

	return &types.File{
		Name:    "routes.go",
		Content: factory.NewFileNode("routes", imports, routesFunc),
	}
}

// routeMethodCall generates mux.Post("/path", handlers.Handler) etc.
func routeMethodCall(muxName, method, path, handler string) *ast.CallExpr {
	// Normalize method to proper chi method name (Post, Get, Put, Delete)
	var methodName string
	switch method {
	case "POST":
		methodName = "Post"
	case "GET":
		methodName = "Get"
	case "PUT":
		methodName = "Put"
	case "DELETE":
		methodName = "Delete"
	case "PATCH":
		methodName = "Patch"
	case "OPTIONS":
		methodName = "Options"
	default:
		return nil
	}

	return factory.NewSelectorCall(muxName, methodName,
		factory.NewBasicLit(path),
		factory.NewSelector("handlers", handler),
	)
}

// DefaultHandlersPackage generates a handlers package with one file per handler.
// Each handler file contains an empty function stub for business logic.
// Duplicate handler names are skipped.
func DefaultHandlersPackage(s *types.Service) *types.Package {
	if s.RoutesConfig == nil {
		return nil
	}

	// Track seen handler names to detect duplicates
	seen := make(map[string]bool)
	var files []*types.File

	for _, group := range s.RoutesConfig.RoutesGroup {
		for _, route := range group.Routes {
			handlerName := route.Handler
			if handlerName == "" {
				continue
			}

			// Skip duplicates
			if seen[handlerName] {
				continue
			}
			seen[handlerName] = true

			// Create handler file
			file := createHandlerFile(handlerName)
			if file != nil {
				files = append(files, file)
			}
		}
	}

	if len(files) == 0 {
		return nil
	}

	return &types.Package{
		Name:  "handlers",
		Files: files,
	}
}

// createHandlerFile creates a single handler file with an empty function body.
// File name: {handlerName}.go (e.g., LoginHandler.go)
// Function signature: func {handlerName}(w http.ResponseWriter, r *http.Request)
func createHandlerFile(handlerName string) *types.File {
	imports := factory.NewImportDecl(
		factory.NewImport("net/http", ""),
	)

	// func {handlerName}(w http.ResponseWriter, r *http.Request) {}
	handlerFunc := factory.NewFuncDecl(
		handlerName,
		factory.NewFieldList(),
		factory.NewFuncType(
			factory.NewFieldList(
				factory.NewField("w", factory.NewSelector("http", "ResponseWriter")),
				factory.NewField("r", &ast.StarExpr{X: factory.NewSelector("http", "Request")}),
			),
			factory.NewFieldList(),
		),
		factory.NewBodyStmt(), // Empty body - business logic placeholder
	)

	return &types.File{
		Name:    handlerName + ".go",
		Content: factory.NewFileNode("handlers", imports, handlerFunc),
	}
}
