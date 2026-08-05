# Event Envelope Contract

**Status:** DRAFT / WAITING FOR HUMAN APPROVAL

Every NATS event must contain:

```text
event_id
run_id
source_id
occurred_at
schema_version
producer
payload
evidence_refs
```

Consumers must be idempotent by `event_id` and bounded by a retry budget. An event is evidence of a stage transition, not permission to perform an external mutation.
