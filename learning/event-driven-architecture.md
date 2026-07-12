# HTTP vs Message Broker

- HTTP is a synchronous request/response protocol for direct client-server commuication.
- Message brokers handle asynchronous, decoupled messaging by routing data through queues.

| | HTTP | Message Broker |
|---|---|---|
| Coupling | Tight - client must know the server, server must be up | Loose - producer doesn't know or care who consumes |
| Timing | Synchronous - blocked until response | Asynchronous - fire and forget |
| Failure handling | Server down = client gets error immediately | Server down = message waits in queue, processed when consumer recovers |
| Scaling | More load = server needs to handle it now | More load = queue grows, consumers process at their own pace |
| Delivery guarantee | None built-in - if it fails, client must retry | Built-in - queue holds message until acknowledged |
| Response | Immediate result back to caller | No response to producer (unless you build a reply pattern) |

### When to use which
**HTTP** - when the caller needs an answer now:
- User hits "get my profile" -> needs data immediately
- Frontend submits a form -> needs "success" or validation erros back

**Message Broker** - when the work can happen later:
- User uploads a CSV -> "accepted" immediatelt, imports runs in background
- Order confirmed -> send confirmation email, decrease inventory, notify warehouse (all async)
- Work is expensive and you need to control throughput (workers consume at their own rate)

# RabbitMQ Architecture

`Producer -> publishes to an Exchange (with a routing key) -> Exchange checks Bindings (routing rules) -> Routes to matching Queue(s) -> Consumer subscribes and receives from Queue -> Ack/Nack back to broker`

**Exchange**

- An exchange is a router that sits between producers and queues. Producer sends messages to the exchange. The exchange decides which queue(s) to forward the message to, based on rules.
- Without exchange, producer would need to know about every queue. With one, the producer just says "here's a message about an order", and the exchange routes it to whoever cares.

**Exchange types**
|Type|Behavior|Example|
|---|---|---|
|Direct|Route by exact matching key|`payment.success` -> payment queue only|
|Fanout|Broadcast to ALL bound queues|Order placed -> email queue, inventory queue, analytics queue all get it|
|Topic|Route by pattern matching (`*` = one word, `#` = zero or more)|`order.*` matches `order.created`; `order.#` matches `order.created.us`|
|Headers|Match on message headers (key-value pairs), ignores routing key|Route by `{ "format": "pdf", "region": "us" }` with `x-match: all/any`| 

**Binding**

A binding is the rule that connects an exchange to a queue. Without it, the exchange has nowhere to route messages

**Binding + Exchange type**

- Direct exchange - binding key must match routing key exactly
- Fanout exchange - binding key is ignored, all bound queues get every message
- Topic exchange - binding key is a pattern with wildcards

**Topic / Channel**

The named category a message belongs tp. Producers publish to a topic, consumers subscribe to topics they care about.

**Consumer Group**

Multiple instances of the same consumer sharing the load. Each message goes to **one** of them.

**Dead Letter Exchange (CLX) / Dead Letter Queue (DLQ)**

A DLX is a normal exchange designated as the fallback for a queue. Messages are dead-lettered when:
- Consumer rejects (nack without requeue)
- TTL expires
- Queue length limit exceeded

The DLQ routes to a DQL for manual inspection. Can have its own bindings for different routing of failed messages.

**Acknowledgment (Ack)**

Consumer tells the broker "I processed this message successfully, delete it." Without ack, the broker re-delivers it (at-least-once delivery).

**Backpressure**

When consumers can't keep up, the queue grows. Backpressure is the mechanism to signal "slow down" upstream.

**Ordering guarantees**
|Guarantee|Meaning|Cost|
|---|---|---|
|No ordering| Messages processed in any order | Maximum parallelism|
|Partition ordering| Messages with same key processed in order (e.g., all events for user123 are sequential) | Less parallelism |
|Total ordering| Every message processed in exact publish order| Single consumer, no parallelism|

**At-least-once vs At-most-once vs Exactly-once**
|Delivery|Behavior|Risk|
|---|---|---|
|At-most-once| Fire-and-forget, no retry|Message can be lost|
|At-least-once| Retry until ack'd | Message can be processed twice (duplicates) |
|Exactly-once| Deduplication + ack | Hard to achieve, expension, often "effectively once" via idempotency|
> Many systems uses at-least-once + idempotent handlers

**Event vs Command**

| | Event | Command |
|---|---|---|
|Meaning| "This happened" | "Do this" |
|Example| `OrderPlaced` | `SendConfirmationEmail` |
|Coupling| Producer doesn't know/care who listens | Producer knows what action it wants |
|Failure| Not the producer's problem|Producer may need to know if it failed|

# Kafka vs RabbitMQ

| | Kafka | RabbitMQ |
|---|---|---|
| Mental model | Append-only log that everyone reads | Post office that delivers and forgets |
| Message lifetime | Retained regardless of consumption | Deleted after ack |
| Consumer model | Pull (consumer controls pace) | Push (broker delivers to consumer) |
| Replay | Yes - any consumer can rewind | No - once ack'd, gone |
| Routing complexity | Minimal (topic + partition key) | Rich (exchanges, bindings, patterns) |
| Ordering | Per-partition guarantee | Per-queue guarantee |
| Throughput | Very high (append-only disk writes) | Moderate (per-message broker overhead)