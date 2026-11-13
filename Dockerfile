FROM golang:1.24-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /bot ./cmd/bot

FROM alpine:3.20
RUN apk add --no-cache docker-cli ca-certificates
COPY --from=build /bot /bot
CMD ["/bot"]
