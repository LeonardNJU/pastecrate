FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod main.go ./
RUN CGO_ENABLED=0 go build -o /pastecrate .
FROM alpine:3.20
RUN adduser -D -u 10001 app && mkdir -p /data && chown app:app /data
COPY --from=build /pastecrate /pastecrate
USER app
VOLUME ["/data"]
EXPOSE 8080
ENTRYPOINT ["/pastecrate"]
