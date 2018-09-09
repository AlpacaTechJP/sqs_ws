### build stage
FROM golang:alpine AS build-env
ADD . /go/src/github.com/AlpacaDB/sqs_ws
WORKDIR /go/src/github.com/AlpacaDB/sqs_ws

RUN CGO_ENABLED=0 go build -o /tmp/sqs_ws .


### docker image
FROM alpine

COPY --from=build-env /tmp/sqs_ws /sqs_ws

ADD ca-certificates.crt /etc/ssl/certs/
ADD config.yml /config.yml


ENV AWS_REGION ap-northeast-1

CMD ["/sqs_ws", "-c", "/config.yml"]