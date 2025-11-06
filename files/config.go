package files

import (
	"go/ast"
	"go/token"
	"instant-layer/factory"
)

func ConfigFile(service *Service) *ast.File {
	return factory.NewFileNode(
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
}
