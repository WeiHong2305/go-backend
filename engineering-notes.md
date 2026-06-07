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