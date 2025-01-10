FROM golang:1.22.5 AS build
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main

FROM scratch
WORKDIR /app
COPY --from=build /app/main .
ENTRYPOINT ["/app/main"]
