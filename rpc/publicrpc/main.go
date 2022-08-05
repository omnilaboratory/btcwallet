package main

import (
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcwallet/rpc/legacyrpc"
	"log"
	"net/http"
)

func main() {
	connCfg := &rpcclient.ConnConfig{
		Host:         "localhost:8332",
		User:         "omniwallet",
		Pass:         "cB3]iL2@eZ1?cB2?",
		HTTPPostMode: true, // Bitcoin core only supports HTTP POST mode
		DisableTLS:   true, // Bitcoin core does not provide TLS by default
	}
	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Shutdown()


	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Connection", "close")
			w.Header().Set("Content-Type", "application/json")
			r.Close = true
			legacyrpc.PublicHttpHandler(w,r,client)
		})

	log.Println("server start at :18332")
	err=http.ListenAndServe(":18332", serveMux)
	if err != nil {
		log.Println(err)
	}
}