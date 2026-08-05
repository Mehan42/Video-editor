# ADR-007: Secret and Egress Boundary

**Status:** DRAFT / WAITING FOR HUMAN APPROVAL

## Decision

Secrets come only from an approved secret store. External LLM egress is disabled by default and requires redaction, provider readiness, allowlist, classification, and Human Gate checks.

## Reason

A provider or generated artifact must not become an uncontrolled path for secret exfiltration.
