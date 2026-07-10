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

## Go Context (`context` package)

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



