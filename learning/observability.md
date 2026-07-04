# Observability

# Three pillars of observability
1. Metrics - Produces quantitive insights into system performance.
    - Helps team to understand the "what" of system issues.
    - Examples: Host metrics (Memory, disk, CPU usage), Network performance metrics (uptime, latency, throughput), App metrics (response times, request and error rates), Server pool metrics (total instances, number of running instances), External dependency metrics (availability, service status)
    - Often aggregated
    - Limited context
2. Logs - Immutable, exhaustive records of discrete events that occur within a system.
    - Helps team to understand the "why" of system issues.
3. Traces - Combine some of the features of metrics and logs, map data across network components to show a request's workflow.
    - Helps teams to understand the "where" and "how" of system events and issues.
    - Useful in microservices architectures
    - Essential for latency analyses, identifying problematic components and underperforming services that create performance bottlenecks.

## How they work together?
Metrics are used to alert teams to problems, traces show their path of execution and logs provide the context needed to resolve them.

