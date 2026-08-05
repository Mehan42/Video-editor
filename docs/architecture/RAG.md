# RAG and Knowledge Boundary

**Status:** DRAFT / WAITING FOR HUMAN APPROVAL

RAG is a retrieval capability, not a provider feature and not automatically an agent. It is composed of `KnowledgeStore`, `Retriever`, and `ContextAssembler`, followed by the LLM Gateway.

```text
query -> Retriever -> context + source_refs -> ContextAssembler -> LLMGateway
```

## MVP

Use a local knowledge store and deterministic retrieval, such as SQLite full-text search. Embeddings and vector storage remain replaceable implementations of the same Retriever contract.

## Chunk identity

Every chunk records `chunk_id`, `source_id`, `source_hash`, timestamps, `content_hash`, `run_id`, `trust_class`, and provenance references.

## Poisoning defense

Retrieved content remains untrusted source material. The LLM receives policy, task, and source context in separate fields. A retrieved instruction cannot invoke tools, alter policy, or authorize egress.
