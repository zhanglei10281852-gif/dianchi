FROM golang:1.23 AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -o /out/dianchi ./cmd/server
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/dianchi /dianchi
COPY --from=build /src/migrations /migrations
EXPOSE 8080
ENTRYPOINT ["/dianchi"]
