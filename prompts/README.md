# Prompt registry (file-backed)

Versioned prompt templates for Atlas services. Loaded by `services/agent-orchestrator` `LoadPromptDir`.

## Layout
- One JSON (or YAML) file per prompt version under this tree (subdirs allowed).
- Fields: `id`, `version`, `status` (`draft`|`active`|`retired`), `text` with `{{var}}`, optional `variables[]`.

See [Prompt Registry Design](../docs/architecture/prompt-registry.md).
