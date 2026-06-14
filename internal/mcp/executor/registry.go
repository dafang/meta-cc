package executor

// specialToolRegistry maps tool names to their handlers.
var specialToolRegistry = map[string]SpecialToolHandler{}

// registerHandler adds a handler to the special tool registry.
func registerHandler(name string, h SpecialToolHandler) {
	specialToolRegistry[name] = h
}

// queryHandlerRegistry maps convenience tool names to their query handlers.
var queryHandlerRegistry = map[string]QueryHandlerFunc{}

// registerQueryHandler adds a handler to the query handler registry.
func registerQueryHandler(name string, h QueryHandlerFunc) {
	queryHandlerRegistry[name] = h
}
