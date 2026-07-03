# --- frontend ---
FROM node:22-alpine AS web
RUN npm install -g pnpm@11 --ignore-scripts
WORKDIR /src/web
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile --ignore-scripts
COPY web/ ./
RUN pnpm build

# --- backend ---
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /ggs ./cmd/ggs
RUN mkdir /data

# --- runtime ---
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /ggs /ggs
# Pre-owned /data so fresh named volumes are writable by nonroot (65532).
COPY --from=build --chown=nonroot:nonroot /data /data
EXPOSE 8080
VOLUME /data
ENTRYPOINT ["/ggs"]
