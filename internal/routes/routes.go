package routes

type RoutesConfig struct {
	CORS        *CorsOptions   `json:"cors,omitzero"`
	RoutesGroup []*RoutesGroup `json:"routesGroup,omitempty"`
}

type CorsOptions struct {
	AllowedOrigins   []string `json:"allowedOrigins,omitempty"`
	AllowedMethods   []string `json:"allowedMethods,omitempty"`
	AllowedHeaders   []string `json:"allowedHeaders,omitempty"`
	AllowCredentials bool     `json:"allowCredentials,omitempty"`
	MaxAge           int      `json:"maxAge,omitempty"`
}

type RoutesGroup struct {
	Prefix     string     `json:"prefix,omitempty"`
	Middleware Middleware `json:"middleware,omitzero"` // todo: omitempty when middleware fulfill
	Routes     []*Route   `json:"routes,omitempty"`
}

type Middleware struct {
}

type Route struct {
	Path    string `json:"path,omitempty"`    // /user
	Method  string `json:"method,omitempty"`  // POST, DELETE, PUT, GET
	Handler string `json:"handler,omitempty"` // User Implementation.
}
