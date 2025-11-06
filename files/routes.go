package files

import (
	"go/ast"
	"instant-layer/factory"
)

func RoutesFile(service *Service) *ast.File {
	return factory.NewFileNode(
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
}
