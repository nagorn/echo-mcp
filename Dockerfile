FROM golang:1.26 AS build

WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/echo-mcp ./cmd/echo-mcp

FROM scratch
COPY --from=build /out/echo-mcp /echo-mcp

EXPOSE 8080
ENTRYPOINT ["/echo-mcp"]
