# NATS and MCP Boundary

**Status:** DRAFT / WAITING FOR HUMAN APPROVAL

NATS is the event transport. MCP is the connector/tool boundary. They are complementary and must not be collapsed into one authority layer.

## Event envelope

Every event carries `event_id`, `run_id`, `source_id`, `occurred_at`, `schema_version`, `producer`, `payload`, and `evidence_refs`.

## Initial subjects

- `pagevideo.source.accepted`
- `pagevideo.media.audio_extracted`
- `pagevideo.transcript.ready`
- `pagevideo.chunks.ready`
- `pagevideo.retrieval.requested`
- `pagevideo.llm.requested`
- `pagevideo.llm.completed`
- `pagevideo.artifact.written`

## Connector rules

MCP connectors are capability-scoped and allowlisted. They may return evidence, but they cannot grant permissions, invoke arbitrary shell commands, access broad filesystem paths, or publish externally without a separate policy decision.

The MVP may use an in-process test transport while preserving these event contracts; the production transport boundary remains NATS-shaped from the beginning.
