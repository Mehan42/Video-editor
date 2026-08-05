# Evidence and Provenance

**Status:** DRAFT / WAITING FOR HUMAN APPROVAL

Each run has a manifest linking input hash, tool versions, model identity, configuration hash, prompt/context hash, event IDs, chunk IDs, output hash, decision status, and verification results.

Raw artifacts are preserved separately from derived projections. Raw evidence may contain sensitive plaintext and must not be exported to external memory, MCP, cloud providers, or public outputs until the encrypted-storage and egress gates pass.

Evidence proves what happened; it does not grant permission to do it.
