package main

import (
	"context"
	"strings"
	"sync"
	"time"
	"sync/atomic"
	"fmt"
	"hash/fnv"

	"github.com/Shopify/sarama"
	"github.com/wvanbergen/kafka/consumergroup"
	"github.com/wvanbergen/kazoo-go"

	"github.com/yjp211/go-replica/log"
	pb "github.com/yjp211/go-replica/rpc"

)

var (
	Consumers []*Consumer
)

const (
	TCP  = "tcp"
 	QUIC  = "quic"

 )


//分片算法
type Sharding func(key string, mod int) int

type Client struct {
	id       int
	flag 	  string
	batchSize int
	client   pb.RpcGreeterClient
	recordChans chan *pb.Record

}

type Consumer struct {
	server  string
	flag 	  string
	id       int
	consumer *consumergroup.ConsumerGroup
	sharding int
	fn Sharding
	clients []*Client

}

func createConsumers(rpcFactory CreateRpcClient, replicaServer string, replicaBatchSize int, replicaSharding int,
	kafkaVer string, zookeepers string, group string,
	offsetInitial string, offsetReset bool,	topic string, partitions int) {

	kv, err := sarama.ParseKafkaVersion(kafkaVer)
	if err != nil {
		log.Fatal("parse kafka version:%s failed, %v", kafkaVer, err)
		return
	}


	offsetInit := sarama.OffsetNewest
	if offsetInitial != "latest" {
		offsetInit = sarama.OffsetOldest
	}


	Consumers = make([]*Consumer, partitions)
	for i := 0; i < partitions; i++ {

		config := consumergroup.NewConfig()
		config.Offsets.Initial = offsetInit
		config.Offsets.ProcessingTimeout = 10 * time.Second
		config.Offsets.ResetOffsets = offsetReset
		config.Version = kv

		var zookeeperNodes []string
		zookeeperNodes, config.Zookeeper.Chroot = kazoo.ParseConnectionString(zookeepers)

		kafkaTopics := strings.Split(topic, ",")

		consumer, err := consumergroup.JoinConsumerGroup(group, kafkaTopics, zookeeperNodes, config)
		if err != nil {
			log.Fatal("construct kafka consumer to %s failed: %+v", zookeepers, err)
		}
		go func() {
			for err := range consumer.Errors() {
				log.Error("consumer error:%+v", err)
			}
		}()

		consumerFlag :=   fmt.Sprintf("c%0.2d", i)
		clients := make([]*Client, replicaSharding)
		for j:= 0; j< replicaSharding; j++ {
			rpc, err := rpcFactory(replicaServer)
			if err != nil {
				log.Fatal("construct connect to %s failed: %+v", replicaServer, err)
			}
			clients[j] = &Client{
				id: j,
				flag: fmt.Sprintf("%s#%0.2d", consumerFlag, j),
				batchSize: replicaBatchSize,
				client: rpc,
				recordChans: make(chan *pb.Record, replicaBatchSize*4),
			}
		}

		Consumers[i] = &Consumer{
			server: replicaServer,
			flag: consumerFlag,
			id:          i,
			consumer:    consumer,
			sharding: replicaSharding,
			fn: hashSharding,
			clients:      clients,
		}
	}

}


func startConsumers() {
	stoper := Stoper

	var wg sync.WaitGroup
	gcount := 0
	for _, c := range Consumers {
		gcount += 1
		gcount += c.sharding

	}
	wg.Add(gcount)

	for _, c := range Consumers {
		go func(consumer *Consumer) {

			go func() {
				//生产消息并进行投递
				defer wg.Done()
				var counter uint64 = 0
				for {
					select {
					case err := <-consumer.consumer.Errors():
						log.Error("%-4s] consumer error: %+v", consumer.flag, err)
					case msg := <-consumer.consumer.Messages():
						c := atomic.AddUint64(&counter, 1)
						record := &pb.Record{
							Id:        c,
							Partition: uint32(msg.Partition),
							Key:       string(msg.Key),
							Body:      string(msg.Value),
						}
						//按照key进行sharding
						sd := consumer.fn(record.Key, consumer.sharding)
						consumer.clients[sd].recordChans <- record

						consumer.consumer.CommitUpto(msg)
					case <-stoper:
						log.Warn("%-4s] consumer:Interrupt is detected", consumer.flag)
						goto FOREND
					}
				}
			FOREND:
			}()

			for j:= 0 ; j < consumer.sharding; j ++ {
				c2 := consumer.clients[j]
				go func(client *Client) {
					defer wg.Done()

					var batchCounter uint64 = 0

					for {
					RECONN:
						var stream pb.RpcGreeter_ReplicaClient
						streamChain := make(chan pb.RpcGreeter_ReplicaClient)
						go func () {
							stream, err := client.client.Replica(context.Background())
							if err != nil {
								log.Error("%s] prepare replica stream  %s failed: %+v",
									consumer.flag, consumer.server, err)

								//睡眠一秒钟
								<-time.After(time.Second)
								close(streamChain)
							}else {
								streamChain <- stream
							}
						}()

						select {
						case s, ok :=<-streamChain:
							if ok {
								stream = s
								go func(){
									<- stoper
									stream.CloseSend()
								}()
							}else{
								goto RECONN
							}
						case <-stoper:
							log.Warn("%s] rpc:Interrupt is detected", consumer.flag)
							goto FOREND
						}

						for {
							select {
							//阻塞式读取
							case record := <-client.recordChans:
								//先读出一个数据包
								records := []*pb.Record{record}
								//接着循环读取，尝试读满一个bufferSize大小
								for {
									select {
									case recordAfter := <-client.recordChans:
										records = append(records, recordAfter)
										//包裹已经达到了一个批次大小跳出本次读取
										if len(records) == client.batchSize {
											goto FORWRAPER
										}
									case <-time.After(time.Millisecond):
										//缓冲区没有数据 跳出循环
										goto FORWRAPER
									case <-stoper:
										log.Warn("%s] rpc:Interrupt is detected", consumer.flag)
										goto FOREND
									}
								}
							FORWRAPER:
							//开始打包发送
								c := atomic.AddUint64(&batchCounter, 1)
								wraper := &pb.RecordWrap{
									Flag:        client.flag,
									BatchId:     c,
									ProcessTime: uint64(makeTimestamp()),
									Items:       records,
								}
								if err := stream.Send(wraper); err != nil {
									log.Error("%s] <%-10d> send replica stream  %s failed",
										client.flag, c, err)
									goto RECONN

								} else {
									log.Info("%s] <%-10d> have send <%3d> record.",
										client.flag,  c, len(records))
								}

							case <-stoper:
								log.Warn("%s] rpc:Interrupt is detected", consumer.flag)
								goto FOREND
							}
						}
					}
				FOREND:
				}(c2)
			}

		}(c)
	}

	wg.Wait()

}

func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func hashSharding(key string, mod int)int{
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32()) % mod
}

func StartReplica(replicaProtocol string, replicaServer string, replicaBatchSize int, replicaSharding int,
	kafkaVer string, zookeepers string, group string,
	offsetInitial string, offsetReset bool,	topic string, partitions int) {
	var rpcFactory	CreateRpcClient
	if TCP == replicaProtocol{
		rpcFactory = CreateTcpRpcClient
	}else if QUIC == replicaProtocol{
		rpcFactory = CreateQuicRpcClient
	}else {
		log.Fatal("not support replica protocol:%s", replicaProtocol)
	}

	createConsumers(rpcFactory, replicaServer, replicaBatchSize, replicaSharding, kafkaVer, zookeepers,
			group, offsetInitial, offsetReset, topic, partitions)
	startConsumers()
}

func StopReplica() {
	for _, c := range Consumers {
		c.consumer.Close()
	}
}


