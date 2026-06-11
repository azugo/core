package ratelimit

// InstrAllow returns rate limiter key if the operation is allow event.
func InstrAllow(op string, args ...any) (string, bool) {
	if op != InstrumentationAllow || len(args) != 1 {
		return "", false
	}

	key, ok := args[0].(string)

	return key, ok
}

// InstrPeek returns rate limiter key if the operation is peek event.
func InstrPeek(op string, args ...any) (string, bool) {
	if op != InstrumentationPeek || len(args) != 1 {
		return "", false
	}

	key, ok := args[0].(string)

	return key, ok
}

// InstrWait returns rate limiter key if the operation is wait event.
func InstrWait(op string, args ...any) (string, bool) {
	if op != InstrumentationWait || len(args) != 1 {
		return "", false
	}

	key, ok := args[0].(string)

	return key, ok
}

// InstrReset returns rate limiter key if the operation is reset event.
func InstrReset(op string, args ...any) (string, bool) {
	if op != InstrumentationReset || len(args) != 1 {
		return "", false
	}

	key, ok := args[0].(string)

	return key, ok
}

// InstrAcquire returns semaphore key if the operation is acquire event.
func InstrAcquire(op string, args ...any) (string, bool) {
	if op != InstrumentationAcquire || len(args) != 1 {
		return "", false
	}

	key, ok := args[0].(string)

	return key, ok
}

// InstrRelease returns semaphore key if the operation is release event.
func InstrRelease(op string, args ...any) (string, bool) {
	if op != InstrumentationRelease || len(args) != 1 {
		return "", false
	}

	key, ok := args[0].(string)

	return key, ok
}
