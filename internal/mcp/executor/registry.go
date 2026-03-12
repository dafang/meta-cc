package executor

// specialToolRegistry maps tool names to their handlers.
var specialToolRegistry = map[string]SpecialToolHandler{}

// registerHandler adds a handler to the registry.
func registerHandler(name string, h SpecialToolHandler) {
	specialToolRegistry[name] = h
}
