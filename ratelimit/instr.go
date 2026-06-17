package ratelimit

// InstrAllow returns the rate limiter key and the result the operation fills in
// if op is an allow event. The result is only populated once the finish
// callback is invoked.
func InstrAllow(op string, args ...any) (string, *Result, bool) {
	if op != InstrumentationAllow || len(args) != 2 {
		return "", nil, false
	}

	key, ok := args[0].(string)
	if !ok {
		return "", nil, false
	}

	res, ok := args[1].(*Result)

	return key, res, ok
}

// InstrPeek returns the rate limiter key and the result the operation fills in
// if op is a peek event. The result is only populated once the finish callback
// is invoked.
func InstrPeek(op string, args ...any) (string, *Result, bool) {
	if op != InstrumentationPeek || len(args) != 2 {
		return "", nil, false
	}

	key, ok := args[0].(string)
	if !ok {
		return "", nil, false
	}

	res, ok := args[1].(*Result)

	return key, res, ok
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
