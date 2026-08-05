# LLM Egress Policy

**Status:** DRAFT / WAITING FOR HUMAN APPROVAL

External LLM transmission is disabled by default. A provider adapter must declare its endpoint, model, data classification, capabilities, retention behavior when known, and egress status.

Before egress, the gateway validates destination allowlist, payload size, secret redaction, source classification, provider readiness, and Human Gate status. If any check is unknown, the request becomes `BLOCKED_EGRESS`.
