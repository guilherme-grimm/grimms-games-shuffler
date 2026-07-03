# Go backend + SQLite + React frontend, shipped as a single binary

GGS is a Go backend with SQLite persistence and a React (SPA) frontend. For release builds the compiled frontend assets are embedded in the Go binary (`go:embed`), so self-hosting is: one binary or one container image, env vars for config (`STEAM_API_KEY`, optional Anthropic key), and a volume for the SQLite file. In development the React app runs on its own dev server proxying API calls to the Go backend.

Self-hostability is a core product promise, so deployment friction was the deciding factor: docker-compose stacks with Postgres and a separate frontend server were rejected because the write volume (3 Shuffles per Player per day) will never justify the extra moving parts. React was chosen over a Go-templated frontend because it is easier to work with for the interactive questionnaire and the phosphor/arcade UI.
