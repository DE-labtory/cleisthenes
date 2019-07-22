package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/core"

	"github.com/DE-labtory/cleisthenes/config"
	"github.com/DE-labtory/cleisthenes/example/app"
	kitlog "github.com/go-kit/kit/log"
)

func main() {
	host := flag.String("address", "127.0.0.1", "Application address")
	port := flag.Int("port", 8000, "Application port")
	configPath := flag.String("config", "", "User defined config path")

	address := fmt.Sprintf("%s:%d", *host, *port)

	kitLogger := kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stderr))
	kitLogger = kitlog.With(kitLogger, "ts", kitlog.DefaultTimestampUTC)
	httpLogger := kitlog.With(kitLogger, "component", "http")

	config.Init(*configPath)

	txValidator := func(tx cleisthenes.Transaction) bool {
		// custom transaction validation logic
		return true
	}

	node, err := core.New(txValidator)
	if err != nil {
		panic(fmt.Sprintf("Cleisthenes instantiate failed with err: %s", err))
	}

	go func() {
		for {
			result := <-node.Result()
			fmt.Printf("[DONE]epoch : %d, batch tx count : %d\n", result.Epoch, len(result.Batch))
		}
	}()

	go func() {
		httpLogger.Log("message", "hbbft started")
		node.Run()
	}()

	httpLogger.Log("message", fmt.Sprintf("http server started: %s", address))
	if err := http.ListenAndServe(address, app.NewApiHandler(node, httpLogger)); err != nil {
		httpLogger.Log("message", fmt.Sprintf("http server closed: %s", err))
	}
}
