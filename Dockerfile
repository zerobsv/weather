FROM golang:1.22 AS build

RUN mkdir /app
WORKDIR /app

COPY . .

RUN go get .
RUN go mod tidy
RUN go build -o weather_server main.go
RUN chmod 500 weather_server

FROM debian:bookworm-slim
COPY --from=build /app .
RUN useradd -u 1001 new_user
USER new_user
CMD ["./weather_server"]