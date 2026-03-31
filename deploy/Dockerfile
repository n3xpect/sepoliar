FROM golang:1.26.1 AS builder

ARG MODULE_PATH=sepoliar
ARG VERSION=dev
ARG ENTRY_POINT_PATH=.

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN apt-get update && \
    apt-get install -y upx-ucl --no-install-recommends && \
    rm -rf /var/lib/apt/lists/* && \
    CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -buildvcs=false \
    -v -x \
    -ldflags "-w -s" \
    -o sepoliar . && \
    upx --best --lzma sepoliar || true

ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright
RUN go run github.com/playwright-community/playwright-go/cmd/playwright install chromium


FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y \
    ca-certificates \
    tzdata \
    libgbm1 \
    libnss3 \
    libatk-bridge2.0-0 \
    libgtk-3-0 \
    libx11-xcb1 \
    libxcomposite1 \
    libxdamage1 \
    libxrandr2 \
    libasound2 \
    --no-install-recommends && \
    rm -rf /var/lib/apt/lists/* && \
    adduser --disabled-password --gecos '' --uid 1000 appuser

WORKDIR /app
COPY --from=builder /src/sepoliar .
COPY --from=builder /ms-playwright /ms-playwright
COPY --from=builder /root/.cache/ms-playwright-go /home/appuser/.cache/ms-playwright-go
COPY data/ ./data/

ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright
ENV TZ=UTC

RUN chown -R appuser:appuser /app /home/appuser/.cache
USER appuser

CMD ["./sepoliar", "--claim"]
