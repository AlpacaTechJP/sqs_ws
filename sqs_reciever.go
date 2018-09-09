package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"go.uber.org/zap"
)

const ChanelBufferSize = 10000
const MaxNumberOfMessages = int64(10)
const WaitTimeSeconds = int64(10)

type SQSReceiver struct {
	DeleteCh chan *sqs.ReceiveMessageOutput // SQS delete channel
	RecvCh   chan *sqs.ReceiveMessageOutput // send channel

	connectionNum int
	svcs          []*sqs.SQS
	qURL          *string
	exitch        chan bool
}

func newSQSReceiver(exitch chan bool, config Config) *SQSReceiver {
	awsConfig := config.AWSConfig

	num := config.QueueConnectionNum

	logger.Info(fmt.Sprintf("recv queue: %s", config.SourceQueue),
		zap.Int("queue_connection_num", num),
	)

	svcs := make([]*sqs.SQS, num)

	for i := 0; i < num; i++ {
		s := session.Must(session.NewSessionWithOptions(session.Options{
			Config: awsConfig,
		}))

		svcs[i] = sqs.New(s)
	}

	// assume qURL is same
	qURL, err := svcs[0].GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(config.SourceQueue),
	})
	if err != nil {
		logger.Fatal("NewSQSReceiver failed", zap.NamedError("error", err))
	}

	return &SQSReceiver{
		svcs:          svcs,
		qURL:          qURL.QueueUrl,
		connectionNum: num,
		DeleteCh:      make(chan *sqs.ReceiveMessageOutput, ChanelBufferSize),
		RecvCh:        make(chan *sqs.ReceiveMessageOutput, ChanelBufferSize),
		exitch:        exitch,
	}
}

func (s *SQSReceiver) startRecvQueue(svc *sqs.SQS) {
	param := &sqs.ReceiveMessageInput{
		QueueUrl:            s.qURL,
		MaxNumberOfMessages: aws.Int64(MaxNumberOfMessages),
		WaitTimeSeconds:     aws.Int64(WaitTimeSeconds),
		AttributeNames: aws.StringSlice([]string{
			"SentTimestamp",
		}),
		MessageAttributeNames: aws.StringSlice([]string{
			"All",
		}),
	}
	for {
		ret, err := svc.ReceiveMessage(param)
		if err != nil {
			logger.Warn("sqs recv error", zap.NamedError("error", err))
			continue
		}
		if len(ret.Messages) > 0 {
			s.RecvCh <- ret
		}
	}
}

func (s *SQSReceiver) startDeleteQueue(svc *sqs.SQS) {
	for {
		select {
		case msg, ok := <-s.DeleteCh:
			if !ok {
				logger.Warn("msgCh closed")
				return
			}
			if err := s.Delete(msg, svc); err != nil {
				logger.Warn("sqs delete error", zap.NamedError("error", err))
			}
		}
	}
}

func (s *SQSReceiver) Delete(msgs *sqs.ReceiveMessageOutput, svc *sqs.SQS) error {
	param := &sqs.DeleteMessageBatchInput{
		QueueUrl: s.qURL,
	}

	entries := make([]*sqs.DeleteMessageBatchRequestEntry, len(msgs.Messages))
	for i, msg := range msgs.Messages {
		entries[i] = &sqs.DeleteMessageBatchRequestEntry{
			Id:            msg.MessageId,
			ReceiptHandle: msg.ReceiptHandle,
		}
	}
	param.SetEntries(entries)

	_, err := svc.DeleteMessageBatch(param)
	return err
}

func (s *SQSReceiver) run() {
	for i := 0; i < s.connectionNum; i++ {
		logger.Debug(fmt.Sprintf("recv goroutine started: %d", i))
		go s.startRecvQueue(s.svcs[i])
		go s.startDeleteQueue(s.svcs[i])
	}

	for {
		select {
		case <-s.exitch:
			logger.Warn("exit")
			close(s.RecvCh)
			close(s.DeleteCh)

			return
		}
	}
}
