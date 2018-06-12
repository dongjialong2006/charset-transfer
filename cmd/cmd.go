// project main.go
package main

import (
	"charset-transfer/config"
	"charset-transfer/log"
	"charset-transfer/transfer"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/namsral/flag"
)

func main() {
	var max, min int = 0, 0
	var path string = ""
	fs := flag.NewFlagSetWithEnvPrefix("charset-transfer", "XHARSET_TRANSFER_", flag.ContinueOnError)
	fs.IntVar(&max, "max", 16, "max thread num")
	fs.IntVar(&min, "min", 2, "min thread num")
	fs.StringVar(&path, "path", "", "config file path.")
	fs.String(flag.DefaultConfigFlagname, "", "config location")

	if err := fs.Parse(os.Args[1:]); nil != err {
		fmt.Println(err)
		return
	}

	if path == "" {
		fmt.Println("config file is empty.")
		return
	}
	logger := log.New("charset-transfer")

	ctx := context.Background()
	err := config.Watch(ctx, path)
	if nil != err {
		logger.Fatal(err)
	}

	cfg := config.GetConfig()
	if nil == cfg {
		logger.Fatal("parse config xml file error.")
	}

	var srv transfer.Server = nil
	switch strings.ToUpper(cfg.Remote.Protocol) {
	case "TCP":
		srv = transfer.NewTCP(ctx, cfg, max, min)
	case "UDP":
		srv = transfer.NewUDP(ctx, cfg, max, min)
	default:
		logger.Fatalf("unknown protocol type:%s.", cfg.Remote.Protocol)
	}

	if err = signalNotify(srv); nil != err {
		logger.Fatal(err)
	}

	if err = srv.Start(); nil != err {
		logger.Fatal(err)
	}

	return
}

func signalNotify(srv transfer.Server) error {
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigChan
		signal.Stop(sigChan)
		fmt.Println(fmt.Sprintf("receive stop signal:%v, the programm will be quit.", sig))
		srv.Stop()
	}()
	return nil
}
