# Untrusted Input Policy

**Status:** DRAFT / WAITING FOR HUMAN APPROVAL

Video, audio, subtitles, metadata, transcript, retrieved chunks, URLs, and LLM responses are untrusted data. They must be labeled as source content and must never be interpreted as system or developer instructions.

No source content may select a provider, invoke an MCP tool, modify configuration, read a file, request a secret, or enable network access.

Sanitization is a defense-in-depth measure. The primary control is authority separation and capability allowlisting.
