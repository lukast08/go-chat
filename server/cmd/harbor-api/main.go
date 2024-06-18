package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/kelseyhightower/envconfig"

	"server/cmd/reception-api/server"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	conf := server.Config{}
	err := envconfig.Process("harborapi", &conf)
	if err != nil {
		panic(err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

}
