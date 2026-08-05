# Artifact Sanitization

**Status:** DRAFT / WAITING FOR HUMAN APPROVAL

Generated Markdown, HTML fragments, Mermaid, URLs, filenames, and JSON values are treated as untrusted output.

The writer must enforce an approved output root, reject traversal, use safe filenames, avoid automatic execution, sanitize active markup, and write atomically. Publication and external synchronization are separate capabilities and are disabled in MVP.
