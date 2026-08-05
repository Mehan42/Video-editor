# ADR-005: Isolate Untrusted Media Processing

**Status:** DRAFT / WAITING FOR HUMAN APPROVAL

## Decision

ffmpeg and Whisper run in bounded worker processes without network or access to unrelated paths. Media parsing is never performed in the main orchestration process.

## Reason

Media containers and parser dependencies are an attack surface. Isolation limits the impact of parser compromise.
