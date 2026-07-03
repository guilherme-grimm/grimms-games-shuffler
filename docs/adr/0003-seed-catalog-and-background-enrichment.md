# Catalog cold start: shipped Seed Catalog + background enrichment, Shuffles never block

Steam's storefront `appdetails` endpoint is rate-limited to roughly 40 requests/minute, so enriching a large Library into the Game Catalog from scratch takes 20+ minutes — unacceptable as a first-run experience. We ship a Seed Catalog with each release (tags/genres for the top ~30–50k most-owned appids, generated from SteamSpy dumps by project CI and refreshed periodically), and enrich the remaining long tail in the background after Sync. Shuffling is available immediately over whatever is enriched, with visible progress for the unenriched remainder; a Player is never blocked waiting for enrichment.

Rejected: no seed (sparse, bad first ~20 minutes on any fresh Instance) and blocking first Sync until enrichment completes (kills the first impression, which for a toy like this is most of the product).

## Consequences

- Releases carry a data artifact, and CI owns regenerating it — a dependency on SteamSpy dumps that needs monitoring.
- Early Shuffles on a fresh Instance may draw from a partially enriched Library; the UI shows enrichment progress rather than hiding it.
