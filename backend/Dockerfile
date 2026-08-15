FROM golang:1.22-bookworm AS build
WORKDIR /app
COPY go.mod ./
COPY . .
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    tesseract-ocr \
    tesseract-ocr-tur \
    tesseract-ocr-eng \
    poppler-utils \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd -r baum && useradd -r -g baum -d /app -s /usr/sbin/nologin baum

WORKDIR /app
COPY --from=build /app/server ./server
ENV STORAGE_DIR=/data/storage
RUN mkdir -p /data/storage && chown -R baum:baum /data/storage /app

USER baum
EXPOSE 8080
CMD ["./server"]
