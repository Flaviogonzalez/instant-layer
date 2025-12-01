package defaults

import (
	"go/ast"
	"go/token"

	"github.com/flaviogonzalez/instant-layer/internal/factory"
	"github.com/flaviogonzalez/instant-layer/internal/types"
)

// BrokerService creates a broker service with appropriate options
func BrokerService(opts ...Option) *types.Service {
	s := &types.Service{Port: 8080}

	// Apply user options first
	for _, o := range opts {
		o(s)
	}

	// Broker services need: main, routes (for receiving requests), event package (for emitting)
	baseOpts := []Option{
		WithBrokerConfig(),
		WithBrokerMain(),
		WithRoutes(),
		WithBrokerEvent(),
	}

	return applyOptions(s, baseOpts...)
}

// BrokerEventPackage generates the event package for a broker service
// Contains: emitter.go, event.go
func BrokerEventPackage(s *types.Service) *types.Package {
	return &types.Package{
		Name: "event",
		Files: []*types.File{
			brokerEmitterFile(),
			brokerEventFile(),
		},
	}
}

// BrokerConfigFile generates config.go for a broker service (no database, with HTTP server)
func BrokerConfigFile(s *types.Service) *types.File {
	imports := factory.NewImportDecl(
		factory.NewImport(s.Name+"/routes", ""),
		factory.NewImport("log", ""),
		factory.NewImport("net/http", ""),
	)

	// type Config struct {}
	configStruct := factory.NewTypeStruct("Config", factory.NewFieldList())

	// func InitConfig() *Config
	initConfigFunc := factory.NewFuncDecl(
		"InitConfig",
		factory.NewFieldList(),
		factory.NewFuncType(
			factory.NewFieldList(),
			factory.NewFieldList(factory.NewField("", &ast.StarExpr{X: ast.NewIdent("Config")})),
		),
		factory.NewBodyStmt(
			factory.NewReturn(
				&ast.UnaryExpr{
					Op: token.AND,
					X:  factory.NewCompositeLit(ast.NewIdent("Config")),
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
						factory.NewKeyValue("Handler", factory.NewSelectorCall("routes", "Routes")),
					),
				},
			),
			&ast.IfStmt{
				Init: factory.NewAssignExpectsError(factory.NewSelectorCall("server", "ListenAndServe")),
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
	)

	return &types.File{
		Name: "config.go",
		Content: factory.NewFileNode("config",
			imports,
			configStruct,
			initConfigFunc,
			initServerFunc,
		),
	}
}

// BrokerMainFile generates main.go for a broker service
func BrokerMainFile(s *types.Service) *types.File {
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

// brokerEmitterFile generates emitter.go for broker service
func brokerEmitterFile() *types.File {
	imports := factory.NewImportDecl(
		factory.NewImport("encoding/json", ""),
		factory.NewImport("fmt", ""),
		factory.NewImport("log", ""),
		factory.NewImport("net/http", ""),
		factory.NewImport("github.com/google/uuid", ""),
		factory.NewImport("github.com/rabbitmq/amqp091-go", "amqp"),
	)

	// type Emitter struct
	emitterStruct := factory.NewTypeStruct("Emitter", factory.NewFieldList(
		factory.NewField("conn", &ast.StarExpr{X: factory.NewSelector("amqp", "Connection")}),
		factory.NewField("exchange", ast.NewIdent("string")),
		factory.NewField("id", factory.NewSelector("uuid", "UUID")),
	))

	// func NewEmitter(conn *amqp.Connection, exchange string) *Emitter
	newEmitterFunc := factory.NewFuncDecl(
		"NewEmitter",
		factory.NewFieldList(),
		factory.NewFuncType(
			factory.NewFieldList(
				factory.NewField("conn", &ast.StarExpr{X: factory.NewSelector("amqp", "Connection")}),
				factory.NewField("exchange", ast.NewIdent("string")),
			),
			factory.NewFieldList(
				factory.NewField("", &ast.StarExpr{X: ast.NewIdent("Emitter")}),
			),
		),
		factory.NewBodyStmt(
			factory.NewDefine("uuid", factory.NewSelectorCall("uuid", "New")),
			factory.NewReturn(
				&ast.UnaryExpr{
					Op: token.AND,
					X: factory.NewCompositeLit(
						ast.NewIdent("Emitter"),
						factory.NewKeyValue("conn", ast.NewIdent("conn")),
						factory.NewKeyValue("exchange", ast.NewIdent("exchange")),
						factory.NewKeyValue("id", ast.NewIdent("uuid")),
					),
				},
			),
		),
	)

	// func (e *Emitter) Push(w http.ResponseWriter, topicPayload TopicPayload) error
	pushFunc := factory.NewFuncDecl(
		"Push",
		factory.NewFieldList(
			factory.NewField("e", &ast.StarExpr{X: ast.NewIdent("Emitter")}),
		),
		factory.NewFuncType(
			factory.NewFieldList(
				factory.NewField("w", factory.NewSelector("http", "ResponseWriter")),
				factory.NewField("topicPayload", ast.NewIdent("TopicPayload")),
			),
			factory.NewFieldList(
				factory.NewField("", ast.NewIdent("error")),
			),
		),
		factory.NewBodyStmt(
			// ch, err := e.conn.Channel()
			factory.NewDefineExpectsError("ch",
				factory.NewSelectorCall("e", "conn.Channel"),
			),
			factory.NewIfError(
				factory.NewReturn(ast.NewIdent("err")),
			),
			// defer ch.Close()
			&ast.DeferStmt{
				Call: factory.NewSelectorCall("ch", "Close"),
			},
			// log.Println("push function: topic payload: ", topicPayload)
			factory.NewExprStmt(
				factory.NewSelectorCall("log", "Println",
					factory.NewBasicLit("push function: topic payload: "),
					ast.NewIdent("topicPayload"),
				),
			),
			// jsonBytes, err := json.Marshal(topicPayload.Event)
			factory.NewDefineExpectsError("jsonBytes",
				factory.NewSelectorCall("json", "Marshal",
					factory.NewSelector("topicPayload", "Event"),
				),
			),
			factory.NewIfError(
				factory.NewReturn(ast.NewIdent("err")),
			),
			// q, err := ch.QueueDeclare("", true, true, false, false, nil)
			factory.NewDefineExpectsError("q",
				factory.NewSelectorCall("ch", "QueueDeclare",
					factory.NewBasicLit(""),
					ast.NewIdent("true"),
					ast.NewIdent("true"),
					ast.NewIdent("false"),
					ast.NewIdent("false"),
					ast.NewIdent("nil"),
				),
			),
			factory.NewIfError(
				factory.NewReturn(ast.NewIdent("err")),
			),
			// err = ch.Publish(...)
			factory.NewAssignExpectsError(
				factory.NewSelectorCall("ch", "Publish",
					factory.NewSelector("e", "exchange"),
					factory.NewSelector("topicPayload", "Name"),
					ast.NewIdent("false"),
					ast.NewIdent("false"),
					factory.NewCompositeLit(
						factory.NewSelector("amqp", "Publishing"),
						factory.NewKeyValue("ContentType", factory.NewBasicLit("application/json")),
						factory.NewKeyValue("CorrelationId",
							factory.NewCall(
								factory.NewSelector("e", "id.String"),
							),
						),
						factory.NewKeyValue("ReplyTo", factory.NewSelector("q", "Name")),
						factory.NewKeyValue("Body", ast.NewIdent("jsonBytes")),
					),
				),
			),
			factory.NewIfError(
				factory.NewReturn(ast.NewIdent("err")),
			),
			// e.SendResponse(w, q)
			factory.NewExprStmt(
				factory.NewSelectorCall("e", "SendResponse",
					ast.NewIdent("w"),
					ast.NewIdent("q"),
				),
			),
			factory.NewReturn(ast.NewIdent("nil")),
		),
	)

	// func (e *Emitter) SendResponse(w http.ResponseWriter, q amqp.Queue) error
	sendResponseFunc := factory.NewFuncDecl(
		"SendResponse",
		factory.NewFieldList(
			factory.NewField("e", &ast.StarExpr{X: ast.NewIdent("Emitter")}),
		),
		factory.NewFuncType(
			factory.NewFieldList(
				factory.NewField("w", factory.NewSelector("http", "ResponseWriter")),
				factory.NewField("q", factory.NewSelector("amqp", "Queue")),
			),
			factory.NewFieldList(
				factory.NewField("", ast.NewIdent("error")),
			),
		),
		factory.NewBodyStmt(
			// ch, err := e.conn.Channel()
			factory.NewDefineExpectsError("ch",
				factory.NewSelectorCall("e", "conn.Channel"),
			),
			factory.NewIfError(
				factory.NewReturn(ast.NewIdent("err")),
			),
			// defer ch.Close()
			&ast.DeferStmt{
				Call: factory.NewSelectorCall("ch", "Close"),
			},
			// msgs, err := ch.Consume(...)
			factory.NewDefineExpectsError("msgs",
				factory.NewSelectorCall("ch", "Consume",
					factory.NewSelector("q", "Name"),
					factory.NewBasicLit(""),
					ast.NewIdent("true"),
					ast.NewIdent("false"),
					ast.NewIdent("false"),
					ast.NewIdent("false"),
					ast.NewIdent("nil"),
				),
			),
			factory.NewIfError(
				&ast.BlockStmt{
					List: []ast.Stmt{
						factory.NewExprStmt(
							factory.NewSelectorCall("http", "Error",
								ast.NewIdent("w"),
								factory.NewBasicLit("Failed to set up consumer"),
								factory.NewSelector("http", "StatusInternalServerError"),
							),
						),
						factory.NewReturn(
							factory.NewSelectorCall("fmt", "Errorf",
								factory.NewBasicLit("failed to set up consumer: %w"),
								ast.NewIdent("err"),
							),
						),
					},
				},
			),
			// msg := <-msgs
			factory.NewDefine("msg", &ast.UnaryExpr{Op: token.ARROW, X: ast.NewIdent("msgs")}),
			// if msg.CorrelationId == e.id.String() { ... }
			&ast.IfStmt{
				Cond: &ast.BinaryExpr{
					X:  factory.NewSelector("msg", "CorrelationId"),
					Op: token.EQL,
					Y: factory.NewCall(
						factory.NewSelector("e", "id.String"),
					),
				},
				Body: factory.NewBodyStmt(
					factory.NewExprStmt(
						factory.NewCall(
							factory.NewSelector("w", "Header().Set"),
							factory.NewBasicLit("Content-Type"),
							factory.NewBasicLit("application/json"),
						),
					),
					factory.NewExprStmt(
						factory.NewSelectorCall("w", "WriteHeader",
							factory.NewSelector("http", "StatusOK"),
						),
					),
					&ast.IfStmt{
						Init: &ast.AssignStmt{
							Lhs: []ast.Expr{ast.NewIdent("_"), ast.NewIdent("err")},
							Tok: token.DEFINE,
							Rhs: []ast.Expr{
								factory.NewSelectorCall("w", "Write",
									factory.NewSelector("msg", "Body"),
								),
							},
						},
						Cond: &ast.BinaryExpr{
							X:  ast.NewIdent("err"),
							Op: token.NEQ,
							Y:  ast.NewIdent("nil"),
						},
						Body: factory.NewBodyStmt(
							factory.NewReturn(
								factory.NewSelectorCall("fmt", "Errorf",
									factory.NewBasicLit("failed to write HTTP response: %w"),
									ast.NewIdent("err"),
								),
							),
						),
					},
					factory.NewReturn(ast.NewIdent("nil")),
				),
			},
			factory.NewReturn(ast.NewIdent("nil")),
		),
	)

	return &types.File{
		Name: "emitter.go",
		Content: factory.NewFileNode("event",
			imports,
			emitterStruct,
			newEmitterFunc,
			pushFunc,
			sendResponseFunc,
		),
	}
}

// brokerEventFile generates event.go for broker service
func brokerEventFile() *types.File {
	imports := factory.NewImportDecl(
		factory.NewImport("encoding/json", ""),
		factory.NewImport("log", ""),
		factory.NewImport("math", ""),
		factory.NewImport("net/http", ""),
		factory.NewImport("time", ""),
		factory.NewImport("github.com/rabbitmq/amqp091-go", "amqp"),
	)

	// type EventPayload struct
	eventPayloadStruct := factory.NewStructDecl("EventPayload",
		factory.NewJsonField("Name", "string", "name"),
		&ast.Field{
			Names: []*ast.Ident{ast.NewIdent("Data")},
			Type:  factory.NewSelector("json", "RawMessage"),
			Tag: &ast.BasicLit{
				Kind:  token.STRING,
				Value: "`json:\"data\"`",
			},
		},
	)

	// type TopicPayload struct
	topicPayloadStruct := factory.NewStructDecl("TopicPayload",
		factory.NewJsonField("Name", "string", "name"),
		&ast.Field{
			Names: []*ast.Ident{ast.NewIdent("Event")},
			Type:  ast.NewIdent("EventPayload"),
			Tag: &ast.BasicLit{
				Kind:  token.STRING,
				Value: "`json:\"event\"`",
			},
		},
	)

	// func ConnectToRabbit() *amqp.Connection
	connectFunc := factory.NewFuncDecl(
		"ConnectToRabbit",
		factory.NewFieldList(),
		factory.NewFuncType(
			factory.NewFieldList(),
			factory.NewFieldList(
				factory.NewField("", &ast.StarExpr{X: factory.NewSelector("amqp", "Connection")}),
			),
		),
		factory.NewBodyStmt(
			// var conn *amqp.Connection
			&ast.DeclStmt{
				Decl: &ast.GenDecl{
					Tok: token.VAR,
					Specs: []ast.Spec{
						&ast.ValueSpec{
							Names: []*ast.Ident{ast.NewIdent("conn")},
							Type:  &ast.StarExpr{X: factory.NewSelector("amqp", "Connection")},
						},
					},
				},
			},
			// var counts int64
			&ast.DeclStmt{
				Decl: &ast.GenDecl{
					Tok: token.VAR,
					Specs: []ast.Spec{
						&ast.ValueSpec{
							Names: []*ast.Ident{ast.NewIdent("counts")},
							Type:  ast.NewIdent("int64"),
						},
					},
				},
			},
			// var backoff time.Duration
			&ast.DeclStmt{
				Decl: &ast.GenDecl{
					Tok: token.VAR,
					Specs: []ast.Spec{
						&ast.ValueSpec{
							Names: []*ast.Ident{ast.NewIdent("backoff")},
							Type:  factory.NewSelector("time", "Duration"),
						},
					},
				},
			},
			// for loop
			&ast.ForStmt{
				Body: factory.NewBodyStmt(
					factory.NewDefineExpectsError("connection",
						factory.NewSelectorCall("amqp", "Dial",
							factory.NewBasicLit("amqp://guest:guest@rabbitmq:5672/"),
						),
					),
					&ast.IfStmt{
						Cond: &ast.BinaryExpr{
							X:  ast.NewIdent("err"),
							Op: token.NEQ,
							Y:  ast.NewIdent("nil"),
						},
						Body: factory.NewBodyStmt(
							factory.NewExprStmt(
								factory.NewSelectorCall("log", "Println",
									factory.NewBasicLit("Error trying to connect. Trying again..."),
								),
							),
							&ast.IncDecStmt{X: ast.NewIdent("counts"), Tok: token.INC},
						),
						Else: factory.NewBodyStmt(
							&ast.AssignStmt{
								Lhs: []ast.Expr{ast.NewIdent("conn")},
								Tok: token.ASSIGN,
								Rhs: []ast.Expr{ast.NewIdent("connection")},
							},
							&ast.BranchStmt{Tok: token.BREAK},
						),
					},
					&ast.IfStmt{
						Cond: &ast.BinaryExpr{
							X:  ast.NewIdent("counts"),
							Op: token.GTR,
							Y:  factory.NewBasicLitInt(10),
						},
						Body: factory.NewBodyStmt(
							factory.NewExprStmt(
								factory.NewSelectorCall("log", "Println",
									factory.NewBasicLit("Error trying to connect."),
								),
							),
						),
					},
					// backoff = time.Duration(math.Pow(float64(counts), 2)) * time.Second
					&ast.AssignStmt{
						Lhs: []ast.Expr{ast.NewIdent("backoff")},
						Tok: token.ASSIGN,
						Rhs: []ast.Expr{
							&ast.BinaryExpr{
								X: factory.NewCall(
									factory.NewSelector("time", "Duration"),
									factory.NewSelectorCall("math", "Pow",
										factory.NewCall(ast.NewIdent("float64"), ast.NewIdent("counts")),
										factory.NewBasicLitInt(2),
									),
								),
								Op: token.MUL,
								Y:  factory.NewSelector("time", "Second"),
							},
						},
					},
					factory.NewExprStmt(
						factory.NewSelectorCall("time", "Sleep", ast.NewIdent("backoff")),
					),
					&ast.BranchStmt{Tok: token.CONTINUE},
				),
			},
			factory.NewReturn(ast.NewIdent("conn")),
		),
	)

	// func SendToListener(w http.ResponseWriter, exchange string, topicPayload TopicPayload) error
	sendToListenerFunc := factory.NewFuncDecl(
		"SendToListener",
		factory.NewFieldList(),
		factory.NewFuncType(
			factory.NewFieldList(
				factory.NewField("w", factory.NewSelector("http", "ResponseWriter")),
				factory.NewField("exchange", ast.NewIdent("string")),
				factory.NewField("topicPayload", ast.NewIdent("TopicPayload")),
			),
			factory.NewFieldList(
				factory.NewField("", ast.NewIdent("error")),
			),
		),
		factory.NewBodyStmt(
			factory.NewDefine("conn", factory.NewCall(ast.NewIdent("ConnectToRabbit"))),
			factory.NewDefine("e", factory.NewCall(ast.NewIdent("NewEmitter"), ast.NewIdent("conn"), ast.NewIdent("exchange"))),
			factory.NewDefineExpectsError("_",
				factory.NewSelectorCall("e", "Push", ast.NewIdent("w"), ast.NewIdent("topicPayload")),
			),
			factory.NewIfError(
				factory.NewReturn(ast.NewIdent("err")),
			),
			factory.NewReturn(ast.NewIdent("nil")),
		),
	)

	return &types.File{
		Name: "event.go",
		Content: factory.NewFileNode("event",
			imports,
			eventPayloadStruct,
			topicPayloadStruct,
			connectFunc,
			sendToListenerFunc,
		),
	}
}
