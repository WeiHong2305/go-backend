# Reliability Engineering

Making systems keep woring correctly under real-world conditions - controlling the blast radius when things go wrong.

## Core concerns

| Concern | Question it answers |
| -- | -- |
| Availability | Does the system respond at all? |
| Graceful degradation | What part of it fails, does the rest still work? |
| Load shedding | When overwhelmed, does it reject excess work cleanly? |
| Timeout management | Does a slow dependency bring everything down? |
| Resource management | Do connections, memory, goroutines stay bounded? |
| Observability | Can you see what's happening when thing's go wrong? |

### Leaks
A **leak** is something allocated but never freed - it stays alive consuming resources forever (or until the process dies). Like a water leak: slow, invisible, cumulative.

# Go Context (`context` package)

Controls cancellation, deadlines, and request-scoped values across goroutine boundaries.

### Why context is a reliability primitives

Context is the **mechanism** that enables several of these concerns in Go. It connects "something went wrong up here" to "stop doing unnecessary work down there." Without context, a Go service has no coordinated way to stop work - and a system that can't stop work can't degrade gracefully. It just fails over.

## Core Problem

When a caller gives up (user disconnects, timeout fires), downstream goroutines keep workingly uselessly - wasting CPU, holding connections, causing cascading load. Context gives you a **cancellation tree** so all downstream work stops together.

## Why Context Exists: Cooperative vs Preemptive Concurrency

Go uses **cooperative concellation** because goroutines cannot be forcibly killed.

In preemptive languages (Java, C#), you can forcibly terminate a thread. Go deliberately avoids this because:

1. **Forcible termination is unsafe** - a killed goroutine might be holding a mutex, halfway through a write, or have allocated memory it never frees.
2. **Goroutines are cheap and numerous** - you might run 100,000+. Managing safe preemptive termination at the scale is impractical.
3. **Explicit over magic** - only the goroutine knows when it's safe to stop and what cleanup is needed.

Sp `context` is the cooperative protocol: instead of "I'm killing you," it's "please stop when you're ready."

## The Context Interface

```go
type Context interface {
    Deadline() (deadline time.Time, ok bool)
    Done() <-chan struct{} // closed when context is cancalled
    Err() error // nil until Done is closed, then Cancelled or DeadlineExceeded
    Value(key any) any // request-scoped data (use sparingly)
}
```

## Creating Contexts

| Constructor | Use case |
| -- | -- |
| `context.Background() | Root of any context tree (main, init, top-level) |
| `context.TODO() | Placeholder when you haven't decided which context to use |
| `context.WithCancel(parent) | Manual cancellation - `cancel()` |
| `context.WithTimeout(parent, d) | Auto-cancel after duration `d` |
| `context.WithDeadline(parent, t) | Auto-cancel at time `t` |
| `context.WithValue(parent, k, v) | Attach request-scoped value (trace ID, auth) |

## Patterns

### Checking cancellation in a select

```go
select {
case <-ctx.Done():
    return ctx.Err()
case result := <-ch:
    return process(result)
}
```

### Checking cancellation in a loop

```go
for _, item := range items {
    if ctx.Err() != nil {
        return ctx.Err()
    }
    process(item)
}
```

### Passing context through layers

```go
func (h *Handler) GetMovie(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    movie, err := h.service.GetMovie(ctx, id) // propagates down
}
```

### Timeout for external calls

```go
ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
defer cancel()

resp, err := http.NewRequestWithContext(ctx, "GET", url, nil)
```

## Rules

1. `context.Context` is always the **first parameter**, named `ctx`
2. Never store context in a struct - pass explicitly
3. Never use `context.Background()` deep in the call chain
4. Always `defer cancel()` after `WithCancel`/`WithTimeout`/`WithDeadline`
5. `context.WithValue` is NOT a general-purpose map - only for cross-cutting request-scoped data

## `context.Err()` Return Values

| Value | Meaning |
|---|---|
| `nil` | Context is still active |
| `context.Canceled` | Someone called `cancel()` |
| `context.DeadlineExceeded` | Timeout or deadline was reached |

## `context.Cause(ctx)` (Go 1.20+)

```go
ctx, cancel := context.WithCancelCause(parent)
cancel(fmt.Errorf("rate limited"))

// Later
context.Cause(ctx) // retuns "rate limited" instead of generic Canceled
```

## `context.AfterFunc` (Go 1.21+)

```go
stop := cancel.AfterFunc(ctx, func() {
    conn.Close() // runs when ctx is called
})
defer stop()
```

Registers cleanup that fires on cancellation - alternative to spawning a goroutine just to watch`ctx.Done()`.


# Errors

## Generic Errors vs Wrapped Errors

A generic error is a bare, context-free error - either returned raw from a library or created with a vague message:

```go
// Returning a raw library error - no context about what YOUR code was doing
func (s *userService) SignUp(ctx context.Context, user model.User) (model.User, error) {
    hash, err := bcrypt.GenerateFromPassword(...)
    if err != nil {
        return model.User{}, err // just "crypto/bcrypt: hasher is not available"
    }
}
```

```go
// Or a generic sentinel with no trace
return errors.New("something went wrong")
```

When this hits your logs: `"error": "crypto/bcrypt: hasher is not available"` - you have no idea which function, which user, or which step failed.


## Wrapped erros

A wrapped error adds context at each layer while preserving the original cause:

```go
return fmt.Errorf("hash password: %w", err)
// produces: "hash password: crypto/bcrypt: hasher is not available"
```

Each layer adds its own context:
`repository: "failed to fetch user: connection refused"`
`service: "get user: failed to fetch user: connection refused"`

The `%w` verb is key - it wraps (not replaces) the original, so `errors.Is()` and `errors.As()` still work through the chains.

### When to wrap vs when not to
| Situation | Do |
|---|---|
| Error crosses a layer boundary (repo -> service -> handler) | Wrap with context |
| Error is translated to a different sentinel | Don't wrap the original - return the new sentinel |
| Error is being logged and discarded | Don't wrap, just log |
| Error message is already clear enough | Don't double-wrap ("fetch user: failed to fetch user: ...") |
| Returning to the same package/function | Usually don't wrap |

> The rule of thumb: **wrap when the error crosses a boundary where the reader would lose context about what operation failed.**

> Go has stack trace for **panic** (crashes), but not for **errors** (normal return values). Errors in go are common, and will be expensive to have stack trace for each of them.

## Avoiding Internal Details in HTTP Responses

### Problem

Your service errors carry context for debugging, if forward `err.Error()` to the client, you leak:
- Infrastructure details - database IPs, queue depths, driver names
- Stack context - which internal function failed
- Technology choices - which libraries you use (attach surface info)

### Solution

A **translation layer** at the HTTP boundary that maps internal errors to safe client messages:
```go
func mapServiceError(w http.ResponseWriter, r *http.Request, err error) {
    // Log the FULL error (for debugging)
    slog.ErrorContext(r.Context(), "request failed", "error", err)

    // Return only SAFE messages to the client
    switch {
    case errors.Is(err, service.ErrNotFound):
        respondError(w, 404, "not found")
    case errors.Is(err, service.ErrValidaiton):
        respondError(w, 400, err.Error()) // safe: these messages are user-facing by design
    default:
        respondError(w, 500, "internal server error")
    }
}
```

## Retry Storms

A retry storm happens when many clients retry failed requests simultaneously, amplifying load on an already struggling services.

### Exponential backoff

Spreads retries over time and gives the service breathing room. But still **Synchronized**, just less frequently.

**Tradeoff**
1. Slower recovery for the client
2. Longer total time to exhaust retries (cap solves this)
3. Full jitter can produce near-zero delays
4. Stale requests complete after user has given up (context solves this)
5. Resource holding during wait (gorouting, connection, request context)

### Jitter solves synchronization

Jitter adds randomness to the wait

```go
// Widest spread (Full jitter) - Completes all work fastest and creates the lowest peak load on the server, because it fills the time windows evenly rather than clustering
wait = random(0, baseDelay * 2^attempt)

// Common alternative (Equal jitter)
temp = baseDelay * 2^attempt
wait = temp/2 + random(0, temp/2)
```


# Engineering Thinking

1. Why shouldn't every operation retry forever?

- Multipling load - retry storms cause cascading failures on an already struggling service
- Resource exhaustion - each pending retry holds a goroutine, connection, memory indefinitely
- Stale work - by the time a late retry succeeds, the caller has moved on
- Permanent failures exist - retrying a 404, validation error, or permission denied will never succeed

2. Why are timeouts essential even when everything usually works?

- Timeouts **bound the worst case** - without them, a normally-50ms call can hang for minutes, holding resources the entire time.
- Systems must be designed for the 99.9th percentile, not the average
- Timeouts free resources back to serve other requests, preventing one slow dependency from taking the whole system down.

3. What happens if a request is cancelled halfway through updating multiple resources?
-
- Single-system (multiple DB writes): wrap in a transaction - all succeed or all roll back, no partial state
- Cross-system (DB + cache, DB + external API): transactions don't help across boundaries. Use:
    - Idempotency keys - so retries don't cause duplicates
    - Saga pattern - compensating actions to undo partial work (e.g., refund a charge)
    - Outbox pattern - write event to DB in the same transaction, publish to other systems separately

4. If a background worker crashes while processing a job, how can the system recover?

- Jobs stay in a persistent queue and reappear if no acknowledgment within a deadline (visibility timeout)
- This gives **at-least-once-delivery** - the job will eventually be processed
- Handlers must be **idempotent** - since redelivery means the job might run twice, repeated execution must produce the same result (e.g., checkpoint which rows are done, skip on retry)