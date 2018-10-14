package main

import (
	"github.com/Shopify/sarama"
	"strings"
	"time"

	"github.com/yjp211/go-replica/log"
	pb "github.com/yjp211/go-replica/rpc"
)

var (
	Producer  sarama.AsyncProducer
	DestTopic string
)

func InitProducer(kafkaVer string, brokers string, destTopic string) {
	kv, err := sarama.ParseKafkaVersion(kafkaVer)
	if err != nil {
		log.Fatal("parse kafka version:%s failed, %v", kafkaVer, err)
		return
	}

	DestTopic = destTopic
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForLocal       // Only wait for the leader to ack
	config.Producer.Flush.Frequency = 500 * time.Millisecond // Flush batches every 500ms
	config.Version = kv

	brokerList := strings.Split(brokers, ",")
	producer, err := sarama.NewAsyncProducer(brokerList, config)
	if err != nil {
		log.Fatal("Failed to start Sarama producer:", err)
	}

	// We will just log to STDOUT if we're not able to produce messages.
	// Note: messages will only be returned here after all retry attempts are exhausted.
	go func() {
		for err := range producer.Errors() {
			log.Error("Failed to write access log entry: %+v", err)
		}
	}()
	Producer = producer
}

func ProducerRecord(wrap *pb.RecordWrap) {
	cur := uint64(makeTimestamp())
	transTime := cur - wrap.ProcessTime

	log.Info("%-4s] <%-10d> with <%3d> record, transport time: %dms", wrap.Flag, wrap.BatchId, len(wrap.Items), transTime)

	for _, record := range wrap.Items {
		msg := &sarama.ProducerMessage{
			Topic: DestTopic,
			Key:   sarama.StringEncoder(record.Key),
			Value: sarama.StringEncoder(record.Body),
			//Timestamp: time.Now(),
		}

		Producer.Input() <- msg
	}
}


func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}