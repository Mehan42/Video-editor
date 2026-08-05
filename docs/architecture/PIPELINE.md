# Processing Pipeline

**Status:** DRAFT / WAITING FOR HUMAN APPROVAL

## Stages

1. `source.accepted`: validate path, size, extension, hash, and run manifest.
2. `media.audio_extracted`: run ffmpeg in the isolated media worker.
3. `transcript.ready`: run Whisper with an explicit model path and bounded resources.
4. `chunks.ready`: normalize text and create deterministic chunks with timestamps and hashes.
5. `retrieval.requested`: optionally retrieve context from the local knowledge store.
6. `llm.requested`: submit a bounded request through the LLM Gateway.
7. `llm.completed`: validate structured output and attach provider evidence.
8. `artifact.written`: write sanitized artifacts below the run output root.

## Failure behavior

Failures are terminal for the current stage and retain the raw diagnostic. The pipeline must not silently continue with empty transcript, missing provider evidence, invalid schema, or unverified output.

Required statuses include `READY`, `DEGRADED`, `REPORT_ONLY`, `BLOCKED_INPUT`, `BLOCKED_LLM_UNAVAILABLE`, `BLOCKED_EGRESS`, and `FAILED`.
