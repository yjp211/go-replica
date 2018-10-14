package main

import (
	"io/ioutil"
	"encoding/json"
)

type Config struct {
	TcpPort     int
	QuicPort     int
	LogPath  string
	LogLevel string

	ReplicaEnable bool

	KafkaVersion string
	KafkaBrokers     string
	KafkaDestTopic     string
}


func ReadConfig(path string)(*Config, error){
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