---
stepsCompleted: [1, 2, 3, 4]
inputDocuments: []
session_topic: 'Redis vs. In-memory caching for lean schedules and Pub/Sub strategy'
session_goals: '1. Evaluate Redis caching benefits for scalability/consistency. 2. Design/Validate Redis Pub/Sub for data prep.'
selected_approach: 'ai-recommended'
techniques_used: ['Constraint Mapping', 'Chaos Engineering']
ideas_generated: []
context_file: ''
---

# Brainstorming Session Results

**Facilitator:** Nrtdemo
**Date:** 2026-04-24

## Session Overview

**Topic:** Redis vs. In-memory caching for lean schedules and Pub/Sub strategy
**Goals:** 
1. Evaluate Redis caching benefits for scalability/consistency.
2. Design/Validate Redis Pub/Sub for data prep.

### Session Setup

We are exploring moving "lean schedules" (schedules with DecisionRules stripped to avoid duplication) from a local in-memory cache (`s.cacheMemory.Schedules`) to a shared Redis cache. We are also planning to use Redis Pub/Sub to trigger schedule data updates/preparation for the `GET /content` endpoint.

## Technique Selection

**Approach:** AI-Recommended Techniques
**Analysis Context:** Redis vs. In-memory caching for lean schedules and Pub/Sub strategy

**Recommended Techniques:**

- **Constraint Mapping:** Analyzing memory pressure vs. Redis network latency to find the breaking points.
- **Chaos Engineering:** Stress-testing the Pub/Sub flow against race conditions and thundering herds.


## Technique Execution Results

**Constraint Mapping & Chaos Engineering:**

- **Interactive Focus:** Defining RAM vs. Latency boundaries, Ping-and-Pull dynamics, and cluster-wide consistency.
- **Key Breakthroughs:**
    - Shifted from "Redis as a primary store" to "Redis as a Global Mirror backbone."
    - Adopted a "Strict Integrity" model where incorrect data is rejected over availability.
    - Designed a "Lazy Self-Heal" mechanism that uses API traffic to verify cache health.

### Generated Ideas

**[Architecture #1]**: Global Mirror Pattern
_Concept_: Each delivery pod maintains a full, local in-memory copy of all active "lean" schedules. Redis acts as the source of truth and the synchronization backbone via Pub/Sub to push updates to the pods.
_Novelty_: Shifts Redis from being a per-request lookup store to a state-propagation backbone that keeps local RAM eventually consistent.

**[Architecture #2]**: The Ping-and-Pull Signal
_Concept_: Pub/Sub carries only the "Placement ID" and a "Version Hash." Pods check their local version; if it's different, they fetch the full lean-schedule set from Redis.
_Novelty_: Decouples the *event* of a change from the *data* of the change, allowing for version-safe updates without saturating the Pub/Sub channel.

**[Architecture #3]**: Adaptive Re-Sync (Lazy Self-Heal)
_Concept_: In-memory schedules are treated as "stale-at-rest." On every GET /content call, the pod checks the timestamp. If it exceeds a safe window, it kicks off a background goroutine to re-pull from Redis.
_Novelty_: Turns API traffic into a self-healing heartbeat, ensuring consistency even if Pub/Sub messages are dropped.

**[Architecture #4]**: Jittered Synchronization
_Concept_: Upon receiving a Pub/Sub update signal, each delivery pod introduces a localized, random delay (e.g., 50ms - 500ms) before initiating the Redis 'Pull'.
_Novelty_: Prevents distributed "Thundering Herds" on Redis by flattening the load curve during cluster-wide updates.

**[Architecture #5]**: Strict Integrity Fail-Fast
_Concept_: If a pod detects its local mirror is stale and it cannot reach Redis to verify/refresh the state, it intentionally returns an error for requests targeting that placement.
_Novelty_: Prioritizes correctness over availability, a critical choice for KBank security and legal compliance.

**[Architecture #6]**: Deterministic Mirror Fidelity
_Concept_: The delivery pods act as "Stateless Mirrors" of the Redis cache, faithfully executing whatever state exists in Redis without local "safety fuses" or validation delays.
_Novelty_: Eliminates architectural "gray areas," ensuring pod behavior is 100% predictable based on Redis state.

**[Architecture #7]**: Cluster Convergence Monitoring (Hash Gating)
_Concept_: Each delivery pod computes a checksum of its local schedule map and exports it as a Prometheus metric.
_Novelty_: Provides a single-pane-of-glass view of cluster consistency; deviations act as a "Split Brain" indicator for immediate alerting.

### Creative Facilitation Narrative

The session evolved from a fundamental concern about RAM and data completeness to a sophisticated distributed systems architecture. By combining the speed of local RAM with the centralized authority of Redis, the collaboration produced a "Global Mirror" strategy. We navigated the trade-offs of network congestion (Jitter), fault tolerance (Fail-Fast), and observability (Hash Gating), resulting in a design that balances performance with strict integrity.


## Idea Organization and Prioritization

**Thematic Organization:**

**Theme 1: Distribution & Sync (The "Plumbing")**
_Focus: How data moves and stays consistent across the cluster._
- **[Architecture #1]: Global Mirror Pattern** - Low-latency local execution with centralized authority.
- **[Architecture #2]: The Ping-and-Pull Signal** - Lightweight Pub/Sub to minimize network noise.
- **[Architecture #4]: Jittered Synchronization** - Critical protection against Redis thundering herds.

**Theme 2: Resilience & Integrity (The "Shield")**
_Focus: How the system survives failures and prevents bad data._
- **[Architecture #3]: Adaptive Re-Sync (Lazy Self-Heal)** - API traffic as a health-check pulse.
- **[Architecture #5]: Strict Integrity Fail-Fast** - Prioritizing bank-grade correctness over stale uptime.
- **[Architecture #6]: Deterministic Mirror Fidelity** - Eliminating logic "gray areas."

**Theme 3: Observability (The "Eyes")**
_Focus: How you know the system is working._
- **[Architecture #7]: Cluster Convergence Monitoring (Hash Gating)** - Real-time cluster health via Prometheus.

**Prioritization Results:**

- **Top Priority Ideas:**
    - **Global Mirror Foundation**: Essential for the core hybrid caching strategy.
    - **Ping-and-Pull with Jitter**: Required for scalable, safe cluster synchronization.
    - **Strict Integrity Fail-Fast**: Non-negotiable for KBank data consistency standards.

**Action Planning:**

**Priority 1: The Global Mirror Foundation**
- **Immediate Next Steps:** Refactor `MemoryCache` to support full-slice re-population and implement the Redis Subscriber background worker in `CMSDeliveryService`.
- **Resources Needed:** Redis Pub/Sub access, updated Go cache structs.
- **Timeline:** 1-2 Sprints.

**Priority 2: Ping-and-Pull with Jitter**
- **Immediate Next Steps:** Define the Pub/Sub DTO and implement the `rand.Intn` jitter logic in the listener to protect Redis bandwidth.
- **Success Indicators:** Flat Redis network metrics during placement updates.

**Priority 3: Strict Integrity Fail-Fast**
- **Immediate Next Steps:** Add `last_successful_sync` metadata and implement the staleness guard in the primary delivery handler.
- **Success Indicators:** Zero "ghost rule" executions during simulated network partitions.

## Session Summary and Insights

**Key Achievements:**
- Designed a sophisticated, self-healing distributed cache architecture tailored for KBank's security needs.
- Successfully moved from a simple "Redis GET" mindset to a high-performance "Global Mirror" pattern.
- Established clear resilience patterns (Jitter, Fail-Fast) that prepare the service for high-throughput production loads.

**Session Reflections:**
This session demonstrated the power of "Chaos Engineering" as a design tool. By trying to "break" the Pub/Sub flow, we discovered the need for Jitter and Hash-based monitoring, turning a standard cache implementation into a robust distributed system.


## Implementation Review Findings

- [ ] [Review][Decision] **Staleness Guard Threshold Config** — Hardcoded 2.0x multiplier may be too aggressive or inflexible for different environments.
- [x] [Review][Patch] **Redis Subscription Reconnection Handling** [internal/repository/redis_repository.go:164]
- [x] [Review][Patch] **Unbounded Subscriber Channel** [internal/repository/redis_repository.go:172]
- [x] [Review][Patch] **Potential Startup Panic in `NewCMSDeliveryService`** [cmd/svc-contstrat-delivery/service/cms_delivery_service.go:196]
- [x] [Review][Patch] **Targeted Refresh Deletion Handling** [cmd/svc-contstrat-delivery/service/cms_delivery_service.go:580]
- [x] [Review][Patch] **Version Check Race Condition** [cmd/svc-contstrat-delivery/service/cms_delivery_service.go:809]
- [x] [Review][Patch] **Jitter 0ms Boundary** [cmd/svc-contstrat-delivery/service/cms_delivery_service.go:825]
- [x] [Review][Patch] **Targeted Refresh Rule Eviction** [cmd/svc-contstrat-delivery/service/cms_delivery_service.go:680] — Resolved via `PruneOrphanedRules` sweep.
