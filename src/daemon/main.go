package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/suvarshachennareddy/minicni/common/constants"
	"github.com/suvarshachennareddy/minicni/daemon/internal/manager"
	"github.com/suvarshachennareddy/minicni/daemon/internal/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	netConfPath := os.Getenv(constants.NetConfPathEnv)
	pluginBinPath := os.Getenv(constants.PluginBinPathEnv)

	mgr, err := manager.New(netConfPath, pluginBinPath)
	if err != nil {
		log.Fatal(err)
	}

	srv, err := server.New(constants.SockPath, mgr)
	if err != nil {
		log.Fatal(err)
	}

	srv.Serve(ctx)
}
