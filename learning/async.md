# Background Jobs & Asynchronous Processing

## Terminologies

1. **Concurrency** and **Parallelism** - Conceptual models of how work is structured.
    - Concurrency - Dealing with multiple tasks at the same time. It is the concept of managing multiple things by rapidly switching between them, but not necessarily executing them at the exact same instant.
    - Parallelism - Literally doing multiple things at the exact same time. It requires multiple CPU cores to actively process separate tasks simultaneously.

2. **Multithreading** and **Async** - Programming implementations used to achieve those models.
    - Multithreading - Spawning multiple lightweight execution threads within a single process to achieve concurrency or parallelism. These threads share the same memory but can take turns executing on a processor core.
    - Async - A programming approach where a task pauses while waiting for a long operation (like downloading a file or querying a database). Instead of blocking the whole system, the program switches to do other work and resumes the paused task later.

## Concurrency Design Patterns
> https://dev.to/kanywst/concurrency-design-patterns-from-fundamental-theory-to-architecture-35j7

### State Management
- Shared Memory Model (sync.Mutex)
- Message Passing Model (channel)

### Design Patterns are trying to avoid:
- Race Condition
- Deadlock
- Livelock
- Starvation

### 3 Layers
_Concurrency design patterns are not conflicting, but rather meant to be combined across layers_

| Layer | Role | Decision to make |
| --- | --- | --- |
| Lv 3. | Architecture |	Overall Structure	Data flow of the entire system |
| Lv 2. | Task Decomposition | Tactics	How to handle large tasks |
| Lv 1. | State Management |	Communication	Rules for data transfer |

### Lv 2 Pattern: Task Decomposition

1. Fork-Join Pattern
2. MapReduce Pattern
3. Work Stealing Pattern
4. Master-Worker Pattern

### Lv 3 Pattern: Architecture and Data Flow

1. Pipeline Pattern
2. Producer-Consumer Pattern
    - Decouples the process of generating data from the process of processing it, by using a shared buffer or queue.
    - Core Challenges and Solutions:
        - Queue Full: Producers must be blocked or throtlled (backpressure) to prevent memory crashes
        - Thread Safety: Locking mechanisms on queue
        - Idempotency: Messages may be processed more than once (failures & retries), ensuring that processing is idempotent (same effect)
3. Scatter-Gather Pattern

### Lv 3 / Lv 1: Async and Messaging
> Patterns for minimizing I/O wait time without blocking threads.
1. Actor Model
2. CSP (Communicating Sequential Processes) - A model adopted by the Go language
3. Future / Promise
4. Reactor / Proactor Pattern

### Lv 3 / Lv2
1. Worker Pool Pattern
    - Maintain a fixed number of reusable workers to process tasks from a shared queue.
    - Benefits:
        - Resource control: Protects memory and CPU
        - Cost Efficiency: Eliminates the performance overhead of constantly creating and destroying threads for short-lived tasks.
        - Backpressure Handling: When tasks arrive faster than they can be processed, they sit safely in the queue without crashing the application.


## Best Practice
1. Persistence - jobs service process restarts (Redis/Postgres/Kafka, not just memory)
2. Retry with backoff - exponential, with jitter to avoid thundering herd
3. Idempotency - safe to re-run (DB constraints, idempotency keys, or checkpointing)
4. Dead-letter queue - after max retries, move to a DLQ for human inspection instead of silently dropping
5. Timeout - jobs don't run forever; kill after a deadline
6. Observability - structured logs, metrics (queue depth, processing time, failure rate)
7. Graceful shutdown - finish in-flight jobs before exiting
8. Concurrency control - limit parallel workers to avoid overwhelming downstream services
