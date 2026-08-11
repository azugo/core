package cache

// InstrGet returns cache key if the operation is cache get event.
func InstrGet(op string, args ...any) (string, bool) {
	if op != InstrumentationGet || len(args) == 0 {
		return "", false
	}

	key, ok := args[0].(string)

	return key, ok
}

// InstrGetHit returns cache key if the operation is cache get hit event.
func InstrGetHit(op string, args ...any) (string, bool) {
	if op != InstrumentationGetHit || len(args) == 0 {
		return "", false
	}

	key, ok := args[0].(string)

	return key, ok
}

// InstrSet returns cache key if the operation is cache set event.
func InstrSet(op string, args ...any) (string, bool) {
	if op != InstrumentationSet || len(args) == 0 {
		return "", false
	}

	key, ok := args[0].(string)

	return key, ok
}

// InstrDelete returns cache key if the operation is cache delete event.
func InstrDelete(op string, args ...any) (string, bool) {
	if op != InstrumentationDelete || len(args) == 0 {
		return "", false
	}

	key, ok := args[0].(string)

	return key, ok
}

// InstrBackend returns the cache backend type from instrumentation event
// arguments if present.
func InstrBackend(args ...any) (Type, bool) {
	if len(args) < 2 {
		return "", false
	}

	t, ok := args[1].(string)

	return Type(t), ok
}

// InstrInstance returns the cache instance name from instrumentation event
// arguments if present.
func InstrInstance(args ...any) (string, bool) {
	if len(args) < 3 {
		return "", false
	}

	name, ok := args[2].(string)

	return name, ok
}

// InstrLoader returns cache key if the operation is cache loader event.
func InstrLoader(op string, args ...any) (string, bool) {
	if op != InstrumentationLoader || len(args) == 0 {
		return "", false
	}

	key, ok := args[0].(string)

	return key, ok
}
