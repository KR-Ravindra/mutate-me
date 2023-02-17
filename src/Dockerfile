FROM golang:alpine as dev-env

WORKDIR /app

FROM dev-env as build-env
COPY go.mod /go.sum /app/
RUN go mod download

COPY . /app/

RUN CGO_ENABLED=0 go build -o /mutate-me

FROM alpine:3.10 as runtime

COPY --from=build-env /mutate-me /usr/local/bin/mutate-me
RUN chmod +x /usr/local/bin/mutate-me

ENTRYPOINT ["mutate-me"]