FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -o /loreline-gateway ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /loreline-gateway /loreline-gateway
EXPOSE 8080
ENTRYPOINT ["/loreline-gateway"]
