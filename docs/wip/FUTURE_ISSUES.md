# Future Issues

## Thundering Herd on Mass Reconciliation

When the master restarts and `apply` is called (or if we add a reconciliation loop),
all dead agents get transitioned to pending and started at once. At small scale this
is fine, but at larger scale this creates backpressure / thundering herd problems.

Kubernetes handles this with:
- **Rate-limited work queues** — process N reconciliation items at a time, not all at once
- **Exponential backoff (CrashLoopBackOff)** — failed starts back off 10s, 20s, 40s... up to 5min
- **maxSurge / maxUnavailable** — cap concurrent creates/destroys (e.g. 25% at a time)
- **Jitter** — randomize backoff timers so retries don't synchronize

When we need this, add a concurrency limit to `StartPending` and stagger agent launches.
