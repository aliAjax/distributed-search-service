FROM golang:1.23 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/searchd ./cmd/searchd
RUN mkdir -p /out/web && cp -r web/. /out/web/
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/searchd /searchd
COPY --from=build /out/web /web
EXPOSE 8080
ENTRYPOINT ["/searchd"]
