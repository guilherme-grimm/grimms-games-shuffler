# AI is optional garnish over a deterministic recommendation pipeline

Every Shuffle runs the same deterministic pipeline: the Mood is translated into filters over the Game Catalog and the Player's Library, producing a candidate set. In non-AI mode, GGS picks from the candidates by weighted random and generates a templated Why. In AI mode, the LLM receives only the pre-filtered candidates (plus the Mood Note) and does just the final pick and the Why prose — it never sees the raw Library and cannot recommend outside the candidate set.

We chose this over an AI-first design (send the whole library to the LLM, let it do everything) so that non-AI mode is a first-class equal, not a degraded fallback — "with OR without AI" is a product promise. It also bounds token cost and keeps recommendations explainable and testable.

## Consequences

- Whether AI is available is an Instance-level config (the operator provides the API key — the hosted instance's operator or the self-hoster). When available, each Player still toggles AI per Shuffle.
- Mood questions must remain answerable by deterministic filters; only the Mood Note is AI-exclusive.
- The 3-Shuffles-per-day cap doubles as the Instance operator's LLM cost ceiling.
- Single gateway (OpenRouter, OpenAI-compatible API) initially — free models first, paid ones if the project gets traction. The gateway sits behind a thin Picker interface so swapping providers/models is config, not a redesign. (Originally Anthropic direct; switched to OpenRouter before implementation for zero-cost experimentation.)
