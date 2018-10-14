package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"os/signal"
	

	"github.com/yjp211/go-replica/log"
)

var (
	ConfigPath    = flag.String("configPath", "./config.json", "config path")
	PRINT_VERSION = flag.Bool("v", false, "version info")

	Version   string
	GitBranch string
	GitCommit string
	BuildTime string

	config *Config


	Stoper = make(chan struct{})
)

func Fatalf(err error) {
	fmt.Println(err)
	os.Exit(-1)
}

func main() {
	flag.Parse()
	if *PRINT_VERSION {
		fmt.Printf("Version:<%v> GitCommit:<%s-%s> BuildTime:%s\n",
			Version, GitBranch, GitCommit, BuildTime)
		os.Exit(1)
	}
	runtime.GOMAXPROCS(runtime.NumCPU())
	var err error
	config, err = ReadConfig(*ConfigPath)
	if err != nil {
		Fatalf(err)
	}
	fmt.Printf("%+v\n", config)

	log.CreateMyLog("monitor", config.LogPath, config.LogLevel)

	log.Info("----------begin---------")

	if config.ReplicaEnable{
		go StartReplica(config.ReplicaProtocol, config.ReplicaServer, config.ReplicaBatchSize, config.ReplicaSharding,
			config.KafkaVersion, config.KafkaZookeepers,
			config.KafkaGroup,config.KafkaOffsetInitial, config.KafkaOffsetReset,
			config.KafkaTopic, config.KafkaPartitions)

		log.Info("----------start replica: %s---------", config.ReplicaProtocol)
	}


	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	<-signals
	close(Stoper)
	if config.ReplicaEnable{
		StopReplica()
	}


	log.Info("----------end---------")

}
