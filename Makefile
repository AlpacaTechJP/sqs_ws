install_tools:
	go get github.com/tools/godep

build:
	go build

build_image:
	godep restore
	docker build -t alpacadb/afp.input.fanout.pon:latest .

test_elasticmq:
	docker rm -f elasticmq || echo
	docker run -d -p 9324:9324 --name elasticmq s12v/elasticmq
	sleep 3

	go test -run TestMQ
	docker stop elasticmq
	docker rm -f elasticmq


test_send:
	aws sqs send-message --queue-url http://localhost:9324/queue/queue1 --message-body '{"test": "foo"}' --endpoint-url http://localhost:9324
