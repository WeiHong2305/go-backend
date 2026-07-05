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

# Beyond the three pillars

## Fourth pillar: Profiling
Continuous capture of resource usage at the code level (CPU flame graphs, memory allocations, lock contention). Answers "which function/line is causing the bottleneck?" - deeper than traces, which show timing but not internal code hotsports. OpenTelemetry officially added profiling as a signal type alongside traces/metrics/logs.

## Enabling practices (make the pillars useful)
- Context - Metadata that enriches telemetry (e.g., environment, version, tenant, user ID). Without context, raw data is hard to interpret.
- Correlation - The ability to link signals together (e.g., connecting a trace to its logs and metrics using a shared request/correlation ID). Makes it possible to jump between pillars during investigation.

## Operational capabilities (built on top of the pillars)
- Alerting - Rules that proactively notify teams when metrics cross thresholds or anomalies are detected. Turns passive data into actionable signals before users report issues.
- Dashboarding - Visual representation of metrics and traces for at-a-glance system health.
- Anomaly detection - Automatic identification of unusual patterns without manually defined thresholds.

# Additional Learnings
1. Metric attributes should have low, bounded cardinality (method, status class, job type, cache name). Unbounded values (IDs, keys, user IDs) belongs in logs and traces, not metrics