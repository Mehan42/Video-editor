# ADR-006: Treat Transcript and Retrieval as Untrusted Content

**Status:** DRAFT / WAITING FOR HUMAN APPROVAL

## Decision

Transcript and retrieved chunks are data fields, never instructions. LLM output cannot change policy or invoke tools directly.

## Reason

Audio, subtitles, metadata, and retrieved material can contain prompt injection and false operational instructions.
