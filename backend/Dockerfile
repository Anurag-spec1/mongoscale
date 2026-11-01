FROM golang:1.24-alpine3.21 AS build
WORKDIR /src
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/api /app/api
COPY --from=build /out/seed /app/seed
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/api"]
