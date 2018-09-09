package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

func init() {
	initLogger()
	initMetrics()
}

func initLogger() {
	level := zap.NewAtomicLevel()
	level.SetLevel(zapcore.DebugLevel)

	zapConfig := zap.Config{
		Level:    level,
		Encoding: "json",
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:  "msg",
			TimeKey:     "time",
			EncodeTime:  zapcore.ISO8601TimeEncoder,
			LevelKey:    "level",
			EncodeLevel: zapcore.CapitalLevelEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	l, err := zapConfig.Build()
	if err != nil {
		panic(err)
	}
	logger = l
}

var (
	msgCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "sqs_ws_message_counter",
		Help: "Numer of messages to ws",
	})
)

func initMetrics() {
	prometheus.MustRegister(msgCounter)
}

func main() {
	var confFileName string
	flag.StringVar(&confFileName, "c", "", "config file path")

	flag.Parse()
	if confFileName == "" {
		flag.PrintDefaults()
		return
	}

	logger.Info("read config from", zap.String("path", confFileName))
	config, err := NewConfig(confFileName)
	if err != nil {
		logger.Sync()
		logger.Fatal("parseConfig error", zap.NamedError("error", err))
	}

	exitch := make(chan bool, 1)

	receiver := newSQSReceiver(exitch, config)
	go receiver.run()

	hub := newHub(receiver)
	go hub.run()

	m := http.NewServeMux()
	m.Handle("/metrics", promhttp.Handler())
	m.HandleFunc("/", serveHome) // TODO: disable on production
	m.HandleFunc(config.WSEndpoint, func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	port := fmt.Sprintf(":%d", config.WSPort)
	srv := &http.Server{
		Addr:    port,
		Handler: m,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			logger.Fatal("ListenAndServe: ", zap.NamedError("error", err))
			logger.Sync()
			os.Exit(1)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh // block until signal comes

	// close SQS handler and WS clients
	exitch <- true

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Shutdown failed", zap.NamedError("error", err))
		logger.Sync()
		os.Exit(1)
	}

	logger.Info("sqs_ws finished")
}
