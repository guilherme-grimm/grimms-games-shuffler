# GGS — Grimm's Games Shuffler

Picks a game from a player's Steam library based on their current mood, with an explanation of why. Open source and self-hostable; the reference instance lives at games.grimm0.dev.

## Language

**Sync**:
Authenticating via Steam OpenID and importing the player's owned games into GGS. Yields a verified SteamID64.
_Avoid_: link, connect, import

**Player**:
A person identified by a verified SteamID64 who uses GGS to get recommendations.
_Avoid_: user, account (ambiguous with Steam account)

**Shuffle**:
One recommendation produced for one Mood. Every recommendation counts as a Shuffle, the first of the day included. A Player gets 3 Shuffles per day (reset at UTC midnight), and a game already recommended that day is never recommended again the same day.
_Avoid_: retry, re-roll, pick, attempt

**Library**:
The set of games a Player owns on Steam, imported during Sync.
_Avoid_: collection, owned games

**Game Catalog**:
GGS's shared store of per-game metadata (tags, genres, session length signals) that mood filtering runs against. Shared across all Players of an Instance, independent of any Library.
_Avoid_: game database, metadata cache

**Seed Catalog**:
The pre-built Game Catalog snapshot shipped with GGS releases so a fresh Instance can Shuffle immediately. Covers the most-owned Steam games; the long tail is enriched in the background.
_Avoid_: seed data, bootstrap dump

**Candidates**:
The subset of a Player's Library that matches a Mood after filtering against the Game Catalog. Every Shuffle picks from Candidates; when a Mood yields none, filters are relaxed until some exist (and the Why says so).
_Avoid_: matches, pool, shortlist

**Mood**:
A Player's answers to the fixed questionnaire (Energy, Time, Familiarity, …), each a single choice from chunky options. Every Shuffle is produced for exactly one Mood.
_Avoid_: settings, preferences, vibe

**Mood Note**:
An optional free-text addition to a Mood, read only when the Shuffle uses AI. Ignored (or hidden) in non-AI mode.
_Avoid_: comment, prompt

**Instance**:
A single deployment of GGS — the hosted one at games.grimm0.dev or a self-hosted one. An Instance either has AI available (its operator configured a key) or not; when available, each Player still chooses per Shuffle whether to use it.
_Avoid_: server, deployment, site

**Why**:
The explanation attached to every Shuffle result telling the Player why this game fits their mood. Exists in both AI and non-AI modes (templated when non-AI).
_Avoid_: reason, justification, explanation

## Example dialogue

> **Dev:** A Player synced, answered the questionnaire, and got nothing back — bug?
> **Domain expert:** A Mood can never return nothing. If the Candidates come up empty we relax the filters and the Why admits it. The only hard failure is an unusable Library, and that never burns a Shuffle.
> **Dev:** She didn't like the game and hit the button again — is that a retry?
> **Domain expert:** There's no such thing as a retry. That's her second Shuffle of the day, out of three, and it can't be the same game she just got.
> **Dev:** Her Library is 900 games and the Catalog only knows 850 of them so far.
> **Domain expert:** Fine — she Shuffles over the enriched part right now while the tail enriches in the background. The Seed Catalog exists precisely so that gap starts small.

## Flagged ambiguities

- **"Retry"** — resolved: replaced by **Shuffle**. There is no free first pick; all recommendations count against the daily 3.
- **"Mood" / "questions"** — resolved: see **Mood** and **Mood Note**. The exact question list (Energy, Time, Familiarity, possibly Brain) is a design detail, but every question must be answerable by a Catalog or Library filter.
- **"Day"** — resolved: a day is a UTC calendar day; the Shuffle budget resets at UTC midnight.
