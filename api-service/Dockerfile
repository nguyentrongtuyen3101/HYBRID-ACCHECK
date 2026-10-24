FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/reqsentinel-api ./cmd/server

# Image chạy chỉ chứa 1 binary tĩnh (~20MB)
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/reqsentinel-api /reqsentinel-api
ENV PORT=8000
EXPOSE 8000
ENTRYPOINT ["/reqsentinel-api"]
