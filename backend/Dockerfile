FROM golang:1.22-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/whisper ./cmd/server

FROM alpine:3.20
WORKDIR /app
RUN adduser -D -H whisper
COPY --from=build /bin/whisper /app/whisper
COPY migrations /app/migrations
USER whisper
EXPOSE 8080
CMD ["/app/whisper"]
