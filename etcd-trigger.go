package main

import (
	"flag"
	"log"
	"strings"

	"github.com/coreos/go-etcd/etcd"
)

var progname = "etcd-trigger"

var machinesFlag = flag.String("machines", "http://127.0.0.1:4001", "comma-separated list of etcd machines")
var watchKey = flag.String("watch", "", "etcd key to watch (required)")
var readKey = flag.String("notify", "", "etcd key whose value to send to notify URLs (default watch key)")

func assert(err error) {
	if err != nil {
		log.Fatal(progname, ": ", err)
	}
}

func main() {
	flag.Parse()
	machines := strings.Split(*machinesFlag, ",")
	if *watchKey == "" {
		log.Fatal("-watch flag required")
	}
	if *readKey == "" {
		readKey = watchKey
	}
	log.Print("watch: ", *watchKey, " notify: ", *readKey)

	client := etcd.NewClient(machines)
	response, err := client.Watch(*watchKey, 0, false, nil, nil)
	assert(err)

	if readKey != watchKey {
		response, err = client.Get(*readKey, false, false)
		assert(err)
	}

	value := response.Node.Value
}
