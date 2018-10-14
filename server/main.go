package main

import (
	"flag"
	"fmt"
	"runtime"

	"os"
	"os/signal"
	"syscall"

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

	if config.ReplicaEnable {
		InitProducer(config.KafkaVersion, config.KafkaBrokers, config.KafkaDestTopic)
		log.Info("init kafka producer.")
	}

	if config.TcpPort > 0 {
		StartGRpcTcpServer(config.TcpPort)
	}

	if config.QuicPort > 0 {
		StartGRpcQuicServer(config.QuicPort)
	}


	// catchs system signal
	chSig := make(chan os.Signal)
	signal.Notify(chSig, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println("Signal: ", <-chSig)

	log.Info("----------end---------")

}
