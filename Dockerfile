FROM golang:1.22-bookworm AS build
WORKDIR /src
COPY go.mod ./
COPY main.go ./
COPY internal ./internal
COPY web ./web
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/e5renewx .

FROM debian:bookworm-slim
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata \
  && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/e5renewx /app/e5renewx
COPY web /app/web
ENV PORT=1066 \
    TZ=Asia/Shanghai \
    DATA_PATH=/app/data/store.json
EXPOSE 1066
VOLUME ["/app/data"]
ENTRYPOINT ["/app/e5renewx"]
