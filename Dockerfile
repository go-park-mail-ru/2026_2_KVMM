FROM golang:1.27-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/kvmm-backend ./cmd/server

FROM alpine:3.20

RUN adduser -D -H -s /sbin/nologin appuser
RUN mkdir -p /data/media && chown -R appuser:appuser /data

COPY --from=build /out/kvmm-backend /kvmm-backend
COPY --from=build /src/internal/store/media /data/media
RUN chown -R appuser:appuser /data/media

ENV HTTP_ADDR=0.0.0.0:8080 \
    FRONTEND_ORIGIN=http://localhost:8001 \
    FRONTEND_URL=http://localhost:8001 \
    COOKIE_SECURE=false \
    COOKIE_SAMESITE=lax \
    LOG_MODE=prod

EXPOSE 8080

USER appuser
ENTRYPOINT ["/kvmm-backend"]
