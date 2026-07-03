# GGS frontend

React + TypeScript + Vite SPA, embedded into the Go binary at build time
via `embed.go` (`dist/` is committed empty and produced by `make web`).

```sh
pnpm install --ignore-scripts
pnpm dev      # dev server proxying /api and /auth to :8080 (see vite.config.ts)
pnpm build    # production bundle into dist/
```

Pair `pnpm dev` with `make run` at the repo root; use `cmd/devseed` for a
fake session instead of a real Steam login.
