package main

import (
	"errors"
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	yaml "gopkg.in/yaml.v2"
)

const DefaultSamplingRate = 1
const DefaultQueueConnectionNum = 5

type Config struct {
	SourceQueue        string `yaml:"source_queue"`
	QueueConnectionNum int    `yaml:"connection_num"`

	WSEndpoint string `yaml:"ws_endpoint"`
	WSPort     int    `yaml:"ws_port"`

	AWSRegion   string `yaml:"region,omitempty"`
	AWSEndpoint string `yaml:"endpoint,omitempty"`
	AWSConfig   aws.Config

	SamplingRate int `yaml:"sampling_rate"`
}

func parseConfig(path string) (Config, error) {
	var conf Config

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return conf, err
	}

	err = yaml.Unmarshal(data, &conf)
	return conf, err
}

func NewConfig(path string) (Config, error) {
	conf, err := parseConfig(path)
	if err != nil {
		return conf, err
	}

	if conf.WSEndpoint == "" {
		return conf, errors.New("ws endpoint is not set")
	}

	if conf.QueueConnectionNum == 0 {
		conf.QueueConnectionNum = DefaultQueueConnectionNum
	}

	ac := aws.NewConfig()
	if conf.AWSRegion != "" {
		ac = ac.WithRegion(conf.AWSRegion)
	}
	if conf.AWSEndpoint != "" {
		ac = ac.WithEndpoint(conf.AWSEndpoint)
	}
	if conf.SamplingRate == 0 {
		conf.SamplingRate = DefaultSamplingRate
	}

	conf.AWSConfig = *ac

	return conf, nil
}
