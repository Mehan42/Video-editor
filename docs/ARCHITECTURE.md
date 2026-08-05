# PageVideo Architecture

**Status:** DRAFT / WAITING FOR HUMAN APPROVAL

## System shape

```text
Untrusted Input
  -> Ingress Gate
  -> Isolated Media Worker
  -> Whisper Adapter
  -> Transcript Normalizer
  -> Deterministic Chunker
  -> Knowledge Store
  -> Retriever
  -> NATS Event Bus
  -> MCP Connector Layer
  -> Provider Registry / LLM Gateway
  -> Schema Validator
  -> Safe Artifact Writer
  -> Evidence and Human Gate
```

## Architectural rules

- Domain code depends on ports, not provider SDKs or executable paths.
- Every stage is bounded by a run ID, timeout, retry budget, and artifact manifest.
- External data is evidence, never authority.
- Raw inputs and derived outputs have separate storage and retention policies.
- Provider choice is configuration and capability negotiation, not hard-coded behavior.
- An agent is an optional future orchestrator; the MVP uses a deterministic orchestrator.
- External publication and mutating actions require a separate Human Gate.

## Main components

- `Ingress`: accepts local files and, later, downloader adapters.
- `Media Worker`: invokes ffmpeg in a restricted process boundary.
- `Whisper Adapter`: invokes the selected local speech-to-text runtime.
- `Chunker`: creates stable, timestamped, hashed chunks.
- `Knowledge Store`: persists transcripts, chunks, metadata, and provenance.
- `Retriever`: returns context with source references and trust metadata.
- `LLM Gateway`: provider-neutral generation and capability discovery.
- `Artifact Writer`: writes sanitized Markdown/JSON below an approved output root.
- `Evidence Store`: preserves manifests, hashes, decisions, and verification results.

## Explicit non-goals for MVP

- Autonomous agent loops.
- Arbitrary shell or filesystem tools exposed to an LLM.
- Automatic network downloads or publication.
- Treating Mermaid, Markdown, URLs, or model output as executable content.
