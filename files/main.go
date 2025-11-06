package files

import (
	"go/ast"
	"instant-layer/factory"
)

func MainFile(service *Service) *ast.File {
	return factory.NewFileNode(
		"main",
		factory.NewImportDecl(
			&ast.ImportSpec{
				Path: factory.NewBasicLit(service.Name + "/config"),
			},
		),
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
}
