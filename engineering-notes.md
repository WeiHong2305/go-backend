# Database Migration

## What I learned
1. Why Migration:
   - Single source of truth
   - Prevent schema drift
   - Rollback
   - Versioned
   - Separation of concern


# JWT

## What I learned
1. Authentication approaches - Challenge: HTTP is stateless
    1. Cookie-based authenticaiton
        - Server stores and send session ID to client, client stores it in cookie and send with every request
    2. Basic Authentication - base64 username:password in Authorization header
        - this is encoding
        - why base64? because http header only support certain characters
    3. Bearer Token - (Bearer = HOW you send it / transport method)
        - can be with opaque token, JWT token, UUID token, etc.
        - opaque: server needs to store in shared persistance storage
    4. JWT
        - Self-contained
        - To have invalidation capability, needs short-lived access token and long-lived refresh token

# Middleware

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