FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /loreline-gateway ./cmd/server \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /lorelinectl ./cmd/lorelinectl

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /loreline-gateway /loreline-gateway
COPY --from=build /lorelinectl /lorelinectl
EXPOSE 8080
ENTRYPOINT ["/loreline-gateway"]
