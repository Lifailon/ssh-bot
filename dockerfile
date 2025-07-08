# Build image
FROM golang:1.23-alpine3.20 AS build
WORKDIR /ssh-bot
COPY . .
RUN go mod download
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /ssh-bot/ssh-bot

# Final image
FROM alpine:3.20
WORKDIR /ssh-bot
COPY --from=build /ssh-bot/ssh-bot ./

ENTRYPOINT ["/ssh-bot/ssh-bot"]