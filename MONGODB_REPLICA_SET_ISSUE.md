# MongoDB Replica Set Configuration Issue

**Date:** 2025-10-03  
**Status:** 🔴 UNRESOLVED  
**Impact:** Backend cannot report ladder matches due to transaction failure

---

## 📋 Problem Summary

### Original Error
When attempting to report a match in ladder series format, the following error occurs:

```
LADDER_UPDATE_FAILED: (IllegalOperation)
Transaction numbers are only allowed on a replica set member or mongos
```

### Root Cause
- MongoDB **transactions require replica set mode**
- Development environment was running MongoDB in **standalone mode**
- Ladder position updates use transactions for atomic operations
- Standalone MongoDB does not support transactions

---

## 🔧 Attempted Fixes

### Fix #1: Configure MongoDB as Single-Node Replica Set

#### Changes Made to `docker-compose.yml`

```yaml
services:
  mongodb:
    image: mongo:7
    command: >
      bash -c "
      openssl rand -base64 756 > /tmp/mongodb-keyfile &&
      chmod 400 /tmp/mongodb-keyfile &&
      chown 999:999 /tmp/mongodb-keyfile &&
      mongod --replSet rs0 --bind_ip_all --keyFile /tmp/mongodb-keyfile &
      MONGOD_PID=\$$! &&
      sleep 5 &&
      mongosh --eval \"rs.initiate({ _id: 'rs0', members: [{ _id: 0, host: 'localhost:27017' }] })\" &&
      sleep 2 &&
      mongosh --eval \"
        db = db.getSiblingDB('admin');
        db.createUser({
          user: 'root',
          pwd: 'pingis123',
          roles: [{ role: 'root', db: 'admin' }]
        });
      \" &&
      echo 'MongoDB initialized with replica set and root user' &&
      wait \$$MONGOD_PID
      "
    ports:
      - "27017:27017"
    environment:
      MONGO_INITDB_ROOT_USERNAME: root
      MONGO_INITDB_ROOT_PASSWORD: pingis123
      MONGO_INITDB_DATABASE: pingis
    volumes:
      - mongo_data:/data/db
    healthcheck:
      test: |
        mongosh -u root -p pingis123 --authenticationDatabase admin --quiet --eval "
          try {
            const status = rs.status();
            if (status.ok === 1) {
              print('OK');
            } else {
              quit(1);
            }
          } catch(e) {
            quit(1);
          }
        " || exit 1
      interval: 5s
      timeout: 5s
      retries: 10
      start_period: 30s
```

**Result:** ✅ MongoDB starts as replica set and healthcheck passes

#### Changes Made to `Makefile`

```makefile
# Updated MONGO_URI to include replicaSet parameter
MONGO_URI="mongodb://root:pingis123@127.0.0.1:27017/pingis?authSource=admin&replicaSet=rs0"
```

**Changed from:**
```makefile
MONGO_URI="mongodb://root:pingis123@localhost:27017/pingis?authSource=admin"
```

**Reason:** Added `&replicaSet=rs0` parameter

#### Changes Made to `backend/internal/config/config.go`

```go
// Changed default from localhost to 127.0.0.1
MongoURI: getenv("MONGO_URI", "mongodb://127.0.0.1:27017"),
```

**Reason:** Avoid IPv6 resolution issues with `localhost`

---

## 🐛 Current Issues

### Issue #1: IPv6 vs IPv4 DNS Resolution

**Problem:**
Backend attempts to connect to MongoDB via IPv6 (`[::1]:27017`) but MongoDB container only listens on IPv4.

**Evidence:**
```
dial tcp [::1]:27017: connect: connection refused
```

**Attempted Fix:**
- Changed `localhost` to `127.0.0.1` in Makefile ✅
- Changed default in config.go to `127.0.0.1` ✅

**Result:** 🔴 Still failing - backend logs show mixed success/failure

### Issue #2: Backend Connection Instability

**Symptoms:**
1. Initial connection succeeds:
   ```
   {"action":"/klubbspel.v1.ClubService/ListClubs","result":"SUCCESS"}
   ```

2. Subsequent requests fail:
   ```
   server selection error: server selection timeout
   Type: ReplicaSetNoPrimary
   dial tcp [::1]:27017: connect: connection refused
   ```

3. curl requests hang and timeout (exit code 52)

**Possible Causes:**
- MongoDB driver may be caching DNS resolution
- Replica set configuration may not be stable
- Health check passes but actual queries fail
- Network configuration issue between host and Docker

### Issue #3: Replica Set Member Hostname

**Potential Problem:**
Replica set is initialized with `host: 'localhost:27017'` but backend connects via `127.0.0.1:27017`.

```javascript
// In docker-compose.yml startup script:
rs.initiate({ 
  _id: 'rs0', 
  members: [{ _id: 0, host: 'localhost:27017' }]  // ⚠️ Uses localhost
})
```

But connection string uses:
```
mongodb://root:pingis123@127.0.0.1:27017/...  // Uses 127.0.0.1
```

**Impact:** Replica set hostname mismatch may cause connection issues

---

## 🔍 Diagnostic Commands

### Check MongoDB Container Status
```bash
docker compose ps mongodb
```

Expected: `STATUS: Up X seconds (healthy)`

### Check Replica Set Status
```bash
docker exec klubbspel-mongodb-1 mongosh -u root -p pingis123 --authenticationDatabase admin --eval "rs.status()"
```

Expected: 
```json
{ "ok": 1, "members": [...] }
```

### Test Backend Health
```bash
curl http://localhost:8082/healthz
```

Expected:
```json
{"status":"healthy","checks":{"mongodb":{"status":"healthy"}}}
```

### Test API Endpoint
```bash
curl -s 'http://localhost:8080/v1/clubs?pageSize=10' \
  -H 'Authorization: Bearer 34c23cfb-e6cf-45e6-8318-8e4f9e28bec4' \
  -H 'Accept: application/json'
```

Current: **Hangs and times out (exit code 52)**

### Check Backend Logs
```bash
# In the terminal running make host-dev
# Look for connection errors
```

---

## 🎯 Next Steps to Try

### Option 1: Fix Replica Set Hostname Mismatch
Update docker-compose.yml to use `127.0.0.1` in replica set init:

```javascript
rs.initiate({ 
  _id: 'rs0', 
  members: [{ _id: 0, host: '127.0.0.1:27017' }]  // Match connection string
})
```

### Option 2: Use Docker Network Name
Instead of localhost/127.0.0.1, use Docker service name:
- Replica set member: `mongodb:27017`
- But this won't work for host-based backend

### Option 3: Disable IPv6 in System
Force Go MongoDB driver to only use IPv4:
```go
// In mongo/client.go
clientOptions := options.Client().
    SetDirect(true).  // Disable replica set discovery
    SetHosts([]string{"127.0.0.1:27017"})
```

### Option 4: Run Backend in Docker Too
Change from `make host-dev` to `make dev-start`:
- Backend runs in Docker
- Can use service name `mongodb:27017`
- Network isolation prevents IPv6 issues

### Option 5: Simplified Replica Set Init Script
Create `scripts/init-replica-set.sh`:
```bash
#!/bin/bash
sleep 3
mongosh <<EOF
rs.initiate({
  _id: 'rs0',
  members: [
    { _id: 0, host: '127.0.0.1:27017' }
  ]
})
EOF
```

---

## 📊 Configuration Summary

### Files Modified

1. **docker-compose.yml**
   - Added `--replSet rs0` flag
   - Added keyfile generation
   - Added replica set initialization
   - Added root user creation
   - Updated healthcheck for replica set

2. **Makefile**
   - Changed `localhost` → `127.0.0.1`
   - Added `&replicaSet=rs0` to connection string

3. **backend/internal/config/config.go**
   - Changed default MongoDB URI from `localhost` → `127.0.0.1`

### Current Connection String
```
mongodb://root:pingis123@127.0.0.1:27017/pingis?authSource=admin&replicaSet=rs0
```

### Replica Set Configuration
- **Name:** rs0
- **Members:** 1 node (single-node replica set)
- **Member Host:** localhost:27017 ⚠️ (mismatch with connection string)
- **Authentication:** Username/password with keyfile

---

## 🔬 Technical Background

### Why Transactions Need Replica Sets

MongoDB transactions require replica sets because:
1. **Multi-document ACID guarantees** need distributed consensus
2. **Oplog (operations log)** is only available in replica set mode
3. **Majority write concern** requires multiple nodes (or single-node RS)

### Single-Node Replica Set

A single-node replica set is valid for development:
- Still supports transactions
- Oplog is maintained
- No actual replication occurs
- Simpler than multi-node setup

### MongoDB Driver Connection Behavior

The Go driver:
1. Resolves hostname to IP addresses
2. May get both IPv4 and IPv6 for `localhost`
3. Attempts connections in order
4. May cache failed IPv6 attempts

---

## 📝 Observations from Testing

### Successful Operations
- ✅ MongoDB container starts
- ✅ Replica set initializes
- ✅ Root user created
- ✅ Healthcheck passes
- ✅ Backend starts
- ✅ First API call succeeds (ListClubs, ListSeries)

### Failed Operations
- ❌ curl requests hang indefinitely
- ❌ Backend shows alternating success/failure
- ❌ Connection errors reference `[::1]:27017` (IPv6)
- ❌ "ReplicaSetNoPrimary" topology errors

### Timing Pattern
1. Backend starts → SUCCESS (first 2 seconds)
2. Request arrives → FAILURE (after ~30 seconds)
3. Pattern repeats

**Hypothesis:** Initial connection pool works, but replica set discovery fails and marks primary as unavailable.

---

## 🚨 Workaround for Testing

To test ladder functionality without fixing replica set:

### Temporary: Disable Transactions

In `backend/internal/repo/match_repo.go`:
```go
// Comment out transaction wrapper
// session, err := m.client.StartSession()
// if err != nil { return err }
// defer session.EndSession(ctx)

// return mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
    // ... ladder update logic ...
// })

// Just run without transaction (NOT ATOMIC - FOR TESTING ONLY)
// ... ladder update logic directly ...
```

**⚠️ WARNING:** This removes atomicity guarantees. Only for testing UI flow.

---

## 📚 References

- [MongoDB Transactions](https://www.mongodb.com/docs/manual/core/transactions/)
- [MongoDB Replica Sets](https://www.mongodb.com/docs/manual/replication/)
- [Single-Node Replica Set for Dev](https://www.mongodb.com/docs/manual/tutorial/convert-standalone-to-replica-set/)
- [Go MongoDB Driver](https://pkg.go.dev/go.mongodb.org/mongo-driver/mongo)

---

## ✅ Success Criteria

When fixed, the following should work:

```bash
# 1. Clean start
make host-stop && make host-dev

# 2. MongoDB healthy
docker compose ps mongodb
# STATUS: Up X seconds (healthy)

# 3. Backend healthy
curl http://localhost:8082/healthz
# {"status":"healthy", "checks": {"mongodb": {"status": "healthy"}}}

# 4. API responds
curl 'http://localhost:8080/v1/clubs?pageSize=10' \
  -H 'Authorization: Bearer TOKEN'
# Returns JSON without hanging

# 5. Ladder match reporting works
# In UI: Create ladder series → Report match → No transaction error
```

---

**Last Updated:** 2025-10-03 19:35 CET  
**Next Action:** Try fixing replica set hostname mismatch (Option 1)
