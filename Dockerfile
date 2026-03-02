# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /teamwork ./cmd/teamwork

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache git curl \
    && GH_VERSION=2.67.0 \
    && curl -sL "https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_linux_amd64.tar.gz" \
       | tar xz -C /tmp \
    && mv /tmp/gh_*/bin/gh /usr/local/bin/ \
    && rm -rf /tmp/gh_*

COPY --from=builder /teamwork /usr/local/bin/teamwork

WORKDIR /project
ENTRYPOINT ["teamwork"]
CMD ["--help"]
