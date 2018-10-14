package main

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	LogPath  string
	LogLevel string

	ReplicaEnable	bool
	ReplicaProtocol string
	ReplicaServer string
	ReplicaBatchSize int
	ReplicaSharding int

	KafkaVersion	string
	KafkaZookeepers string
	KafkaGroup      string
	KafkaOffsetInitial string
	KafkaOffsetReset bool
	KafkaTopic      string
	KafkaPartitions    int
}

func ReadConfig(path string) (*Config, error) {
	contents, err := ioutil.ReadFile(path)
	if nil != err {
		return nil, err
	}
	c := &Config{}
	err = json.Unmarshal(contents, c)
	if nil != err {
		return nil, err
	}
	return c, nil
}
