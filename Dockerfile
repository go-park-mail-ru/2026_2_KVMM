FROM golang:1.27-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG TARGETOS
ARG TARGETARCH
ARG GIT_BRANCH=unknown
ARG GIT_COMMIT=unknown
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w -X main.gitBranch=${GIT_BRANCH} -X main.gitCommit=${GIT_COMMIT}" -o /out/kvmm-backend ./cmd/server

FROM alpine:3.20

RUN adduser -D -H -s /sbin/nologin appuser
RUN mkdir -p /data/media && chown -R appuser:appuser /data

COPY --from=build /out/kvmm-backend /kvmm-backend
COPY deploy/config.yml /deploy/config.yml
RUN chown -R appuser:appuser /data/media

EXPOSE 8080

USER appuser
ENTRYPOINT ["/kvmm-backend"]
