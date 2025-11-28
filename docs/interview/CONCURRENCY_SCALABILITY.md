## ⚡ Performance Comparison

### SQLite (Your Implementation)
```
Single Writer Scenario:
┌─────────────────────────────────────────┐
│ Transfer Processing:                    │
│ - BEGIN IMMEDIATE: ~1ms                 │
│ - SELECT account: ~1ms                  │
│ - UPDATE balance: ~1ms                  │
│ - INSERT transfers: ~5ms               │
│ - COMMIT: ~1ms                          │
│                                         │
│ Total: ~10ms per transfer                │
│ Max throughput: ~100 transfers/second    │
└─────────────────────────────────────────┘
```

### PostgreSQL (Production)
```
Concurrent Writer Scenario:
┌─────────────────────────────────────────┐
│ Multiple transfers across multiple accounts: │
│ - All process concurrently              │
│ - Row-level locking                    │
│ - No global serialization               │
│                                         │
│ Total time: ~10ms (vs 1000ms SQLite)    │
│ Max throughput: 50,000-100,000+ TPS     │
│ (depends on hardware & tuning)           │
│                                         │
│ 500x-1000x improvement!                 │
└─────────────────────────────────────────┘
```

### Throughput Analysis

| Scenario | SQLite | PostgreSQL | Improvement |
|----------|--------|------------|-------------|
| Single account | 100 TPS | 100 TPS | 1x (same) |
| 10 accounts | 100 TPS | 1,000 TPS | 10x |
| 100 accounts | 100 TPS | 10,000 TPS | 100x |
| 1000 accounts | 100 TPS | 50,000+ TPS | 500x+ |

**Key Insight**: PostgreSQL scales linearly with account count because it can process different accounts concurrently.

**Scaling Logic**:
- SQLite: Always serialized → 100 TPS regardless of account count
- PostgreSQL: Scales with hardware + account distribution
- Vertical scaling: Better CPU/RAM/SSD → higher TPS
- Horizontal scaling: More accounts → more concurrent operations

---

## 🚀 PostgreSQL Vertical Scaling

### Hardware Impact on Performance

| Hardware Component | Impact on TPS | Typical Range |
|-------------------|---------------|---------------|
| **CPU Cores** | Linear scaling | 8 cores: ~20K TPS<br/>16 cores: ~40K TPS<br/>32 cores: ~80K TPS |
| **RAM** | Buffer cache, connection pooling | 32GB: Good<br/>64GB: Better<br/>128GB+: Excellent |
| **Storage** | Write latency critical | SATA SSD: ~5K TPS<br/>NVMe SSD: ~20K+ TPS<br/>NVMe RAID: ~50K+ TPS |
| **Network** | Connection overhead | 1Gbps: Sufficient<br/>10Gbps: Better |

### Real-World PostgreSQL Performance

**Conservative Estimates** (single instance):
- **Basic server** (8 cores, 32GB RAM, NVMe): ~20,000 TPS
- **High-end server** (32 cores, 128GB RAM, NVMe RAID): ~80,000 TPS
- **Enterprise server** (64+ cores, 256GB+ RAM): ~100,000+ TPS

**Optimized Configurations**:
- Connection pooling (PgBouncer)
- Write-ahead log tuning
- Checkpoint optimization
- Parallel query processing

**Key Point**: PostgreSQL scales much better than SQLite because it can utilize multiple CPU cores and doesn't serialize all writes globally.

---

## 🏗️ Production Architecture

### Horizontal Scaling Setup
```
┌─────────────────────────────────────────┐
│ Load Balancer (HAProxy)                  │
│ ┌─────────────────────────────────────┐ │
│ │ Health checks                        │ │
│ │ SSL termination                      │ │
│ │ Request routing                      │ │
│ └─────────────────────────────────────┘ │
└─────────────────────────────────────────┘
                    │
┌─────────────────────────────────────────┐
│ Application Tier                         │
│ ┌─────────┐ ┌─────────┐ ┌─────────┐     │
│ │ App 1   │ │ App 2   │ │ App 3   │     │
│ │ (3 pods)│ │ (3 pods)│ │ (3 pods)│     │
│ └─────────┘ └─────────┘ └─────────┘     │
└─────────────────────────────────────────┘
                    │
┌─────────────────────────────────────────┐
│ Database Tier                            │
│ ┌─────────────────────────────────────┐ │
│ │ PostgreSQL Primary                   │ │
│ │ - Row-level locking                 │ │
│ │ - MVCC                              │ │
│ │ - Connection pooling                │ │
│ └─────────────────────────────────────┘ │
└─────────────────────────────────────────┘
```

---

## 🎯 When to Scale: Decision Framework

### Prioritize **Sharding** When:

| Symptom | Evidence | Threshold |
|---------|----------|-----------|
| **Database bottleneck** | CPU/memory/connections maxed out | >80% utilization |
| **Concurrency limit** | Only X users can process simultaneously | >20 concurrent users |
| **Write throughput** | Can't process more transfers/second | >50,000 TPS |

**Key Metrics**:
- Database CPU: >80% consistently
- Connection pool: >90% consistently
- Write queue: >100 pending writes
- Database timeouts: >10/hour

### Prioritize **Asynchronous** When:

| Symptom | Evidence | Threshold |
|---------|----------|-----------|
| **Response time complaints** | Users complain about speed | P95 > 1 second |
| **System failure data loss** | Users must resubmit failed requests | >0.1% failure rate |
| **Business requirements** | Need immediate feedback | Business demands it |

**Key Metrics**:
- P95 response time: >1 second consistently
- Timeout rate: >1%
- User abandonment: >5%
- Data loss incidents: >1/month

---

## 💡 Interview Talking Points

### Why BEGIN IMMEDIATE?

> "Without BEGIN IMMEDIATE, two transactions can read the same balance simultaneously, both pass validation, then both try to debit. This creates a race condition where the second transaction fails after doing work. BEGIN IMMEDIATE serializes transactions, so the second transaction reads the updated balance and gets rejected immediately."

### SQLite vs PostgreSQL Scaling?

> "SQLite serializes all writes globally, so throughput is limited to ~100 TPS regardless of account count. PostgreSQL uses row-level locking, so it can process transfers to different accounts concurrently. This gives 10x-1000x improvement depending on account distribution."

### When Would You Shard?

> "I'd prioritize sharding when the database becomes the bottleneck - CPU/memory/connections maxed out, or when we hit concurrency limits. The key metrics are database utilization >80%, connection pool >90%, and write queue backing up. Sharding distributes load across multiple databases."

### When Would You Go Asynchronous?

> "I'd prioritize asynchronous processing when users complain about response times or when system failures cause data loss. Key metrics are P95 response time >1 second, timeout rate >1%, and user abandonment >5%. Async processing improves user experience and system reliability."

---

## 🎯 Key Takeaways

1. **Race conditions** happen when transactions read stale data
2. **BEGIN IMMEDIATE** prevents race conditions by serializing transactions
3. **SQLite** limits throughput due to global serialization
4. **PostgreSQL** scales linearly with account count via row-level locking
5. **Sharding** solves database bottlenecks and concurrency limits
6. **Asynchronous** solves response time and reliability issues
7. **Monitor metrics** to know when to implement each solution

---

## 🔧 Your Implementation Benefits

**For Assessment**:
- ✅ **Demonstrates transaction concepts** clearly
- ✅ **Shows race condition understanding** 
- ✅ **Proves BEGIN IMMEDIATE works** (tested with 5 concurrent goroutines)
- ✅ **Simple and correct** for single-instance scenario

**For Production Evolution**:
- ✅ **Clear path to PostgreSQL** (swap adapter, core unchanged)
- ✅ **Horizontal scaling ready** (stateless service)
- ✅ **Monitoring points identified** (database metrics, response times)
- ✅ **Architecture supports** both sharding and async patterns
