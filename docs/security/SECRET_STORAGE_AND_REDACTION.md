# Secret Storage and Redaction

**Status:** DRAFT / WAITING FOR HUMAN APPROVAL

Secrets must come from an OS-backed secret store or an explicitly approved encrypted store. `.env`, source files, prompts, transcript files, NATS payloads, raw evidence, and generated Markdown are not secret stores.

Redact keys and values matching token, secret, password, API key, authorization, cookie, credential, private key, and provider key patterns before logs or derived evidence are persisted.

Raw sensitive artifacts remain local until encrypted storage, retention, and egress gates pass. File permissions alone are not encryption.
