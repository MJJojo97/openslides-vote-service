FROM golang:1.16.3-alpine3.13 as basis
LABEL maintainer="OpenSlides Team <info@openslides.com>"
WORKDIR /root/

RUN apk add git

COPY go.mod go.sum ./
RUN go mod download

COPY cmd cmd
COPY internal internal

# Build service in seperate stage.
FROM basis as builder
RUN CGO_ENABLED=0 go build ./cmd/vote


# Test build.
FROM basis as testing

RUN apk add build-base

CMD go vet ./... && go test ./...


# Development build.
FROM basis as development

RUN ["go", "install", "github.com/githubnemo/CompileDaemon@latest"]
EXPOSE 9012
ENV MESSAGING redis
ENV AUTH ticket

CMD CompileDaemon -log-prefix=false -build="go build ./cmd/vote" -command="./vote"


# Productive build
FROM scratch

COPY --from=builder /root/vote .
EXPOSE 9013
ENV MESSAGING redis
ENV AUTH ticket

ENTRYPOINT ["/vote"]
