package defaults

import (
	"go/ast"
	"go/token"

	"github.com/flaviogonzalez/instant-layer/internal/factory"
	"github.com/flaviogonzalez/instant-layer/internal/types"
)

// ListenerService creates a listener service with appropriate options
func ListenerService(opts ...Option) *types.Service {
	s := &types.Service{Port: 0} // Listeners don't need an HTTP port

	// Apply user options first
	for _, o := range opts {
		o(s)
	}

	// Listener services need: main, event package (for consuming)
	// No routes or HTTP server - just connects to RabbitMQ and listens
	baseOpts := []Option{
		WithListenerConfig(),
		WithListenerMain(),
		WithListenerEvent(),
	}

	return applyOptions(s, baseOpts...)
}

// ListenerEventPackage generates the event package for a listener service
// Contains: consumer.go, event.go
func ListenerEventPackage(s *types.Service) *types.Package {
	return &types.Package{
		Name: "event",
		Files: []*types.File{
			listenerConsumerFile(),
			listenerEventFile(),
		},
	}
}

// ListenerConfigFile generates config.go for a listener service (connects to RabbitMQ, no HTTP)
func ListenerConfigFile(s *types.Service) *types.File {
	imports := factory.NewImportDecl(
		factory.NewImport(s.Name+"/event", ""),
		factory.NewImport("log", ""),
		factory.NewImport("github.com/rabbitmq/amqp091-go", "amqp"),
	)

	// type Config struct { Conn *amqp.Connection }
	configStruct := factory.NewTypeStruct("Config", factory.NewFieldList(
		factory.NewField("Conn", &ast.StarExpr{X: factory.NewSelector("amqp", "Connection")}),
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
			factory.NewDefine("conn", factory.NewSelectorCall("event", "ConnectToRabbit")),
			factory.NewReturn(
				&ast.UnaryExpr{
					Op: token.AND,
					X: factory.NewCompositeLit(
						ast.NewIdent("Config"),
						factory.NewKeyValue("Conn", ast.NewIdent("conn")),
					),
				},
			),
		),
	)

	// func (app *Config) StartListening() error
	startListeningFunc := factory.NewFuncDecl(
		"StartListening",
		factory.NewFieldList(
			factory.NewField("app", &ast.StarExpr{X: ast.NewIdent("Config")}),
		),
		factory.NewFuncType(
			factory.NewFieldList(),
			factory.NewFieldList(factory.NewField("", ast.NewIdent("error"))),
		),
		factory.NewBodyStmt(
			// handlers := event.handler{ ... }
			factory.NewDefine("handlers",
				factory.NewCompositeLit(
					factory.NewSelector("event", "handler"),
					// Empty handlers map - to be filled by developer
				),
			),
			// consumer := event.NewConsumer(app.Conn, "logs_topic", handlers)
			factory.NewDefine("consumer",
				factory.NewSelectorCall("event", "NewConsumer",
					factory.NewSelector("app", "Conn"),
					factory.NewBasicLit("logs_topic"),
					ast.NewIdent("handlers"),
				),
			),
			// err := consumer.Setup()
			factory.NewDefineExpectsError("_",
				factory.NewSelectorCall("consumer", "Setup"),
			),
			factory.NewIfError(
				factory.NewReturn(ast.NewIdent("err")),
			),
			// topics := []string{"log.INFO", "log.WARNING", "log.ERROR"}
			factory.NewDefine("topics",
				&ast.CompositeLit{
					Type: &ast.ArrayType{Elt: ast.NewIdent("string")},
					Elts: []ast.Expr{
						factory.NewBasicLit("log.INFO"),
						factory.NewBasicLit("log.WARNING"),
						factory.NewBasicLit("log.ERROR"),
					},
				},
			),
			// log.Printf("Listening for topics: %v", topics)
			factory.NewExprStmt(
				factory.NewSelectorCall("log", "Printf",
					factory.NewBasicLit("Listening for topics: %v"),
					ast.NewIdent("topics"),
				),
			),
			factory.NewReturn(
				factory.NewSelectorCall("consumer", "Listen", ast.NewIdent("topics")),
			),
		),
	)

	return &types.File{
		Name: "config.go",
		Content: factory.NewFileNode("config",
			imports,
			configStruct,
			initConfigFunc,
			startListeningFunc,
		),
	}
}

// ListenerMainFile generates main.go for a listener service
func ListenerMainFile(s *types.Service) *types.File {
	imports := factory.NewImportDecl(
		factory.NewImport(s.Name+"/config", ""),
		factory.NewImport("log", ""),
	)

	// func main() { ... }
	mainFunc := factory.NewFuncDecl(
		"main",
		factory.NewFieldList(),
		factory.NewFuncType(factory.NewFieldList(), factory.NewFieldList()),
		factory.NewBodyStmt(
			factory.NewDefine("app", factory.NewSelectorCall("config", "InitConfig")),
			factory.NewDefineExpectsError("_",
				factory.NewSelectorCall("app", "StartListening"),
			),
			factory.NewIfError(
				factory.NewExprStmt(factory.NewSelectorCall("log", "Fatal", ast.NewIdent("err"))),
			),
		),
	)

	return &types.File{
		Name:    "main.go",
		Content: factory.NewFileNode("main", imports, mainFunc),
	}
}

// listenerConsumerFile generates consumer.go for listener service
func listenerConsumerFile() *types.File {
	imports := factory.NewImportDecl(
		factory.NewImport("encoding/json", ""),
		factory.NewImport("log", ""),
		factory.NewImport("github.com/rabbitmq/amqp091-go", "amqp"),
	)

	// type handler map[string]func(event json.RawMessage) ([]byte, error)
	handlerType := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent("handler"),
				Type: &ast.MapType{
					Key: ast.NewIdent("string"),
					Value: &ast.FuncType{
						Params: factory.NewFieldList(
							&ast.Field{
								Names: []*ast.Ident{ast.NewIdent("event")},
								Type:  factory.NewSelector("json", "RawMessage"),
							},
						),
						Results: factory.NewFieldList(
							factory.NewField("", &ast.ArrayType{Elt: ast.NewIdent("byte")}),
							factory.NewField("", ast.NewIdent("error")),
						),
					},
				},
			},
		},
	}

	// type Consumer struct
	consumerStruct := factory.NewTypeStruct("Consumer", factory.NewFieldList(
		factory.NewField("conn", &ast.StarExpr{X: factory.NewSelector("amqp", "Connection")}),
		factory.NewField("exchange", ast.NewIdent("string")),
		factory.NewField("handlers", ast.NewIdent("handler")),
	))

	// func NewConsumer(conn *amqp.Connection, Exchange string, handlers handler) *Consumer
	newConsumerFunc := factory.NewFuncDecl(
		"NewConsumer",
		factory.NewFieldList(),
		factory.NewFuncType(
			factory.NewFieldList(
				factory.NewField("conn", &ast.StarExpr{X: factory.NewSelector("amqp", "Connection")}),
				factory.NewField("Exchange", ast.NewIdent("string")),
				factory.NewField("handlers", ast.NewIdent("handler")),
			),
			factory.NewFieldList(
				factory.NewField("", &ast.StarExpr{X: ast.NewIdent("Consumer")}),
			),
		),
		factory.NewBodyStmt(
			factory.NewReturn(
				&ast.UnaryExpr{
					Op: token.AND,
					X: factory.NewCompositeLit(
						ast.NewIdent("Consumer"),
						factory.NewKeyValue("conn", ast.NewIdent("conn")),
						factory.NewKeyValue("exchange", ast.NewIdent("Exchange")),
						factory.NewKeyValue("handlers", ast.NewIdent("handlers")),
					),
				},
			),
		),
	)

	// func (c *Consumer) Setup() error
	setupFunc := factory.NewFuncDecl(
		"Setup",
		factory.NewFieldList(
			factory.NewField("c", &ast.StarExpr{X: ast.NewIdent("Consumer")}),
		),
		factory.NewFuncType(
			factory.NewFieldList(),
			factory.NewFieldList(
				factory.NewField("", ast.NewIdent("error")),
			),
		),
		factory.NewBodyStmt(
			factory.NewDefineExpectsError("ch",
				factory.NewSelectorCall("c", "conn.Channel"),
			),
			factory.NewIfError(
				factory.NewReturn(ast.NewIdent("err")),
			),
			&ast.DeferStmt{
				Call: factory.NewSelectorCall("ch", "Close"),
			},
			factory.NewReturn(
				factory.NewSelectorCall("ch", "ExchangeDeclare",
					factory.NewSelector("c", "exchange"),
					factory.NewBasicLit("topic"),
					ast.NewIdent("true"),
					ast.NewIdent("false"),
					ast.NewIdent("false"),
					ast.NewIdent("false"),
					ast.NewIdent("nil"),
				),
			),
		),
	)

	// func (c *Consumer) Listen(topics []string) error
	listenFunc := factory.NewFuncDecl(
		"Listen",
		factory.NewFieldList(
			factory.NewField("c", &ast.StarExpr{X: ast.NewIdent("Consumer")}),
		),
		factory.NewFuncType(
			factory.NewFieldList(
				factory.NewField("topics", &ast.ArrayType{Elt: ast.NewIdent("string")}),
			),
			factory.NewFieldList(
				factory.NewField("", ast.NewIdent("error")),
			),
		),
		factory.NewBodyStmt(
			factory.NewDefineExpectsError("ch",
				factory.NewSelectorCall("c", "conn.Channel"),
			),
			factory.NewIfError(
				factory.NewReturn(ast.NewIdent("err")),
			),
			&ast.DeferStmt{
				Call: factory.NewSelectorCall("ch", "Close"),
			},
			// q, err := ch.QueueDeclare(...)
			factory.NewDefineExpectsError("q",
				factory.NewSelectorCall("ch", "QueueDeclare",
					factory.NewBasicLit(""),
					ast.NewIdent("true"),
					ast.NewIdent("false"),
					ast.NewIdent("false"),
					ast.NewIdent("false"),
					ast.NewIdent("nil"),
				),
			),
			factory.NewIfError(
				factory.NewReturn(ast.NewIdent("err")),
			),
			// for _, topic := range topics
			&ast.RangeStmt{
				Key:   ast.NewIdent("_"),
				Value: ast.NewIdent("topic"),
				Tok:   token.DEFINE,
				X:     ast.NewIdent("topics"),
				Body: factory.NewBodyStmt(
					factory.NewAssignExpectsError(
						factory.NewSelectorCall("ch", "QueueBind",
							factory.NewSelector("q", "Name"),
							ast.NewIdent("topic"),
							factory.NewSelector("c", "exchange"),
							ast.NewIdent("false"),
							ast.NewIdent("nil"),
						),
					),
					factory.NewIfError(
						factory.NewReturn(ast.NewIdent("err")),
					),
				),
			},
			// messages, err := ch.Consume(...)
			factory.NewDefineExpectsError("messages",
				factory.NewSelectorCall("ch", "Consume",
					factory.NewSelector("q", "Name"),
					factory.NewBasicLit(""),
					ast.NewIdent("false"),
					ast.NewIdent("false"),
					ast.NewIdent("false"),
					ast.NewIdent("false"),
					ast.NewIdent("nil"),
				),
			),
			// forever := make(chan bool)
			factory.NewDefine("forever",
				factory.NewCall(ast.NewIdent("make"),
					&ast.ChanType{Dir: ast.SEND | ast.RECV, Value: ast.NewIdent("bool")},
				),
			),
			factory.NewExprStmt(
				factory.NewSelectorCall("log", "Println", factory.NewBasicLit("listening for messages!")),
			),
			// for d := range messages
			&ast.RangeStmt{
				Key: ast.NewIdent("d"),
				Tok: token.DEFINE,
				X:   ast.NewIdent("messages"),
				Body: factory.NewBodyStmt(
					// go func(d amqp.Delivery) { ... }(d)
					&ast.GoStmt{
						Call: &ast.CallExpr{
							Fun: &ast.FuncLit{
								Type: factory.NewFuncType(
									factory.NewFieldList(
										factory.NewField("d", factory.NewSelector("amqp", "Delivery")),
									),
									factory.NewFieldList(),
								),
								Body: factory.NewBodyStmt(
									factory.NewExprStmt(
										factory.NewSelectorCall("log", "Println",
											factory.NewBasicLit("new message from: "),
											factory.NewSelector("d", "Exchange"),
										),
									),
									// var eventPayload EventPayload
									&ast.DeclStmt{
										Decl: &ast.GenDecl{
											Tok: token.VAR,
											Specs: []ast.Spec{
												&ast.ValueSpec{
													Names: []*ast.Ident{ast.NewIdent("eventPayload")},
													Type:  ast.NewIdent("EventPayload"),
												},
											},
										},
									},
									factory.NewAssignExpectsError(
										factory.NewSelectorCall("json", "Unmarshal",
											factory.NewSelector("d", "Body"),
											&ast.UnaryExpr{Op: token.AND, X: ast.NewIdent("eventPayload")},
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
												factory.NewSelectorCall("d", "Nack",
													ast.NewIdent("false"),
													ast.NewIdent("false"),
												),
											),
											&ast.ReturnStmt{},
										),
									},
									factory.NewExprStmt(
										factory.NewSelectorCall("log", "Println", ast.NewIdent("eventPayload")),
									),
									// go c.handlePayload(eventPayload, ch, d)
									&ast.GoStmt{
										Call: factory.NewSelectorCall("c", "handlePayload",
											ast.NewIdent("eventPayload"),
											ast.NewIdent("ch"),
											ast.NewIdent("d"),
										),
									},
									factory.NewExprStmt(
										factory.NewSelectorCall("d", "Ack", ast.NewIdent("true")),
									),
								),
							},
							Args: []ast.Expr{ast.NewIdent("d")},
						},
					},
				),
			},
			// <-forever
			&ast.ExprStmt{
				X: &ast.UnaryExpr{Op: token.ARROW, X: ast.NewIdent("forever")},
			},
			factory.NewReturn(ast.NewIdent("nil")),
		),
	)

	return &types.File{
		Name: "consumer.go",
		Content: factory.NewFileNode("event",
			imports,
			handlerType,
			consumerStruct,
			newConsumerFunc,
			setupFunc,
			listenFunc,
		),
	}
}

// listenerEventFile generates event.go for listener service
func listenerEventFile() *types.File {
	imports := factory.NewImportDecl(
		factory.NewImport("encoding/json", ""),
		factory.NewImport("log", ""),
		factory.NewImport("math", ""),
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

	// func (c *Consumer) handlePayload(payload EventPayload, ch *amqp.Channel, msg amqp.Delivery)
	handlePayloadFunc := factory.NewFuncDecl(
		"handlePayload",
		factory.NewFieldList(
			factory.NewField("c", &ast.StarExpr{X: ast.NewIdent("Consumer")}),
		),
		factory.NewFuncType(
			factory.NewFieldList(
				factory.NewField("payload", ast.NewIdent("EventPayload")),
				factory.NewField("ch", &ast.StarExpr{X: factory.NewSelector("amqp", "Channel")}),
				factory.NewField("msg", factory.NewSelector("amqp", "Delivery")),
			),
			factory.NewFieldList(),
		),
		factory.NewBodyStmt(
			// var response []byte
			&ast.DeclStmt{
				Decl: &ast.GenDecl{
					Tok: token.VAR,
					Specs: []ast.Spec{
						&ast.ValueSpec{
							Names: []*ast.Ident{ast.NewIdent("response")},
							Type:  &ast.ArrayType{Elt: ast.NewIdent("byte")},
						},
					},
				},
			},
			// function, ok := c.handlers[payload.Name]
			&ast.AssignStmt{
				Lhs: []ast.Expr{ast.NewIdent("function"), ast.NewIdent("ok")},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{
					&ast.IndexExpr{
						X:     factory.NewSelector("c", "handlers"),
						Index: factory.NewSelector("payload", "Name"),
					},
				},
			},
			// if !ok
			&ast.IfStmt{
				Cond: &ast.UnaryExpr{Op: token.NOT, X: ast.NewIdent("ok")},
				Body: factory.NewBodyStmt(
					factory.NewExprStmt(
						factory.NewSelectorCall("log", "Println",
							factory.NewBasicLit("error trying to execute a function method for the event"),
						),
					),
					factory.NewExprStmt(
						factory.NewSelectorCall("msg", "Nack",
							ast.NewIdent("false"),
							ast.NewIdent("false"),
						),
					),
					&ast.ReturnStmt{},
				),
			},
			factory.NewExprStmt(
				factory.NewSelectorCall("log", "Println",
					factory.NewBasicLit("Event detected, executing function"),
				),
			),
			// r, err := function(payload.Data)
			factory.NewDefineExpectsError("r",
				factory.NewCall(ast.NewIdent("function"),
					factory.NewSelector("payload", "Data"),
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
							factory.NewBasicLit("error executing event: "),
							ast.NewIdent("err"),
						),
					),
					factory.NewExprStmt(
						factory.NewSelectorCall("msg", "Nack",
							ast.NewIdent("false"),
							ast.NewIdent("false"),
						),
					),
					&ast.ReturnStmt{},
				),
			},
			// response = r
			&ast.AssignStmt{
				Lhs: []ast.Expr{ast.NewIdent("response")},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{ast.NewIdent("r")},
			},
			// if msg.ReplyTo == ""
			&ast.IfStmt{
				Cond: &ast.BinaryExpr{
					X:  factory.NewSelector("msg", "ReplyTo"),
					Op: token.EQL,
					Y:  factory.NewBasicLit(""),
				},
				Body: factory.NewBodyStmt(
					factory.NewExprStmt(
						factory.NewSelectorCall("log", "Printf",
							factory.NewBasicLit("No ReplyTo queue specified for event: %s, correlationId: %s"),
							factory.NewSelector("payload", "Name"),
							factory.NewSelector("msg", "CorrelationId"),
						),
					),
					factory.NewExprStmt(
						factory.NewSelectorCall("msg", "Nack",
							ast.NewIdent("false"),
							ast.NewIdent("false"),
						),
					),
					&ast.ReturnStmt{},
				),
			},
			// err = ch.Publish(...)
			factory.NewAssignExpectsError(
				factory.NewSelectorCall("ch", "Publish",
					factory.NewBasicLit(""),
					factory.NewSelector("msg", "ReplyTo"),
					ast.NewIdent("false"),
					ast.NewIdent("false"),
					factory.NewCompositeLit(
						factory.NewSelector("amqp", "Publishing"),
						factory.NewKeyValue("ContentType", factory.NewBasicLit("application/json")),
						factory.NewKeyValue("CorrelationId", factory.NewSelector("msg", "CorrelationId")),
						factory.NewKeyValue("Body", ast.NewIdent("response")),
					),
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
						factory.NewSelectorCall("log", "Printf",
							factory.NewBasicLit("Failed to publish response to %s: %v"),
							factory.NewSelector("msg", "ReplyTo"),
							ast.NewIdent("err"),
						),
					),
					factory.NewExprStmt(
						factory.NewSelectorCall("msg", "Nack",
							ast.NewIdent("false"),
							ast.NewIdent("false"),
						),
					),
					&ast.ReturnStmt{},
				),
			},
			factory.NewExprStmt(
				factory.NewSelectorCall("log", "Printf",
					factory.NewBasicLit("Successfully published response to %s for event: %s, correlationId: %s"),
					factory.NewSelector("msg", "ReplyTo"),
					factory.NewSelector("payload", "Name"),
					factory.NewSelector("msg", "CorrelationId"),
				),
			),
		),
	)

	return &types.File{
		Name: "event.go",
		Content: factory.NewFileNode("event",
			imports,
			eventPayloadStruct,
			connectFunc,
			handlePayloadFunc,
		),
	}
}
