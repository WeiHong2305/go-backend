# Caching

## Caching Strategies
1. Cache-Aside (Lazy Loading)
    - Application manages both the cache and primary datastore
2. Read-Through
    - Application reads from cache, cache retrieve from db if not exist, then return to application.
3. Write-Through
    - Application writes to cache, cache writes to db synchronously
4. Write-Behind
    - Application writes to cache, cache writes to db asynchronously
5. Write-Around
    - Application writes to db directly, skipping cache. When data is requested later, application performs cache check, and load from db to cache if not exist.

## Engineering Thinking

### Why not cache everything?

- Memory cost - RAM is far more expensive than disk.
- Cache invalidation complexity - more cached data = more things to keep in async
- Diminishing returns - the hot/cold access pattern means ~20% of data serves ~80% of reads. Caching cold data wastes memory with near-zero hit rate.

### What happens if Redis crashes?

- Read path: fall back to DB (latency spike as all traffic hits the DB, potentially causing a cache stampede).
- Write path: If doing cache-aside, Redis is down during invalidation, stale entry gone with Redis, so still consistent.

### Why not use Redis instead of PostgreSQL?
- Durability - Redis is primarily in-memory; data can be lost on crash (even with AOF/RDB, weaker recovery guarantees).
- Relational queries - joins, aggregations, filtering on arbitrary columns, transactions with ACID guarantees.
- Data size - PostgreSQL handles datasets far larger than available RAM efficiently via disk-based storage.
- Constraints & integrity - foreign keys, unique constraints, check constraints enforce correctness at the storage layer.