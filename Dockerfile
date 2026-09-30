FROM golang:1.27-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/kvmm-backend ./cmd/server

FROM golang:1.27-bookworm

WORKDIR /app

RUN useradd --system --no-create-home --shell /usr/sbin/nologin appuser

COPY --from=build /out/kvmm-backend /kvmm-backend

ENV HTTP_ADDR=0.0.0.0:8080 \
    FRONTEND_ORIGIN=http://localhost:8001 \
    FRONTEND_URL=http://localhost:8001 \
    COOKIE_SECURE=false \
    COOKIE_SAMESITE=lax

EXPOSE 8080

USER appuser
ENTRYPOINT ["/kvmm-backend"]
