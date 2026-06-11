package cache

// InstrGet returns cache key if the operation is cache get event.
func InstrGet(op string, args ...any) (string, bool) {
	if op != InstrumentationGet || len(args) != 1 {
		return "", false
	}

	key, ok := args[0].(string)

	return key, ok
}

// InstrSet returns cache key if the operation is cache set event.
func InstrSet(op string, args ...any) (string, bool) {
	if op != InstrumentationSet || len(args) != 1 {
		return "", false
	}

	key, ok := args[0].(string)

	return key, ok
}

// InstrDelete returns cache key if the operation is cache delete event.
func InstrDelete(op string, args ...any) (string, bool) {
	if op != InstrumentationDelete || len(args) != 1 {
		return "", false
	}

	key, ok := args[0].(string)

	return key, ok
}

// InstrLoader returns cache key if the operation is cache loader event.
func InstrLoader(op string, args ...any) (string, bool) {
	if op != InstrumentationLoader || len(args) != 1 {
		return "", false
	}

	key, ok := args[0].(string)

	return key, ok
}
