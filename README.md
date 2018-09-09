# SQS WS

This software receives a message from SQS and serve to WebSocket clients.


# How to use

```
$ dep ensure
$ go build
$ ./sqs_ws -c config.yml
```


# Config file

```
---
source_queue: queue1
ws_endpoint: "/ws"
ws_port: 8080
sampling_rate: 1  # log sampling rate
region: ap-northeast-1
endpoint: http://localhost:9324
```


# License

Apache 2
