# Threat Model

**Status:** DRAFT / WAITING FOR HUMAN APPROVAL

## Assets

- API keys and provider credentials.
- Local files outside the project.
- Private video and transcript content.
- Raw evidence, prompts, manifests, and generated knowledge artifacts.
- Provider identity and egress policy.

## Threats

- Media parser exploitation through malformed video/container data.
- Prompt injection in audio, subtitles, metadata, retrieved chunks, or provider responses.
- Secret exfiltration through LLM prompts, logs, events, Markdown, or external connectors.
- RAG poisoning and false authority from untrusted source material.
- Path traversal or output overwrite through generated filenames.
- Supply-chain tampering of binaries, models, adapters, or connector manifests.
- Excessive MCP/NATS capabilities or accidental external publication.

## Security objective

Untrusted input must not gain access to secrets, arbitrary execution, unrestricted network, policy mutation, or publication authority.
