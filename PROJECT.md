# GGS - Grimm's Games Shuffler

This project is a open source AND self hostable version of the one that lives on games.grimm0.dev.
The project aims to make it easy and quick to pick a game from your steam library based in your mood, because, sometimes, we simply don't know what to start to play.
You may choose to use it with OR without AI (because everything nowdays needs AI). Bring your own keys if selfhosting.

## Expected usage

- Sync with steam account (Steam OpenID login)
- Answer the Mood questionnaire
- It'll pick a game for you with a "Why?"
- Three Shuffles per Player per day (see CONTEXT.md)

## Design

- phospor retro design
- arcade 64bits bits feeling

## User flow

- Open
- Sync via Steam OpenID (skipped if a server session already exists)
- Player answers the Mood questionnaire
- Gets the Shuffle result with its Why
