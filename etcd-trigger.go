package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/coreos/go-etcd/etcd"
)

var machinesFlag = flag.String("machines", "http://127.0.0.1:4001", "comma-separated list of etcd machines")
var notifiesFlag = flag.String("notifies", "http://127.0.0.1:8080/", "comma-separated list of URLs to notify")
var triggerKey = flag.String("trigger", "", "etcd key to watch (required)")
var retriggerKey = flag.String("retrigger", "", "etcd key to write after notifications (default no retrigger)")
var readKey = flag.String("read", "", "etcd key whose value to send to notify URLs (default trigger key)")

func assert(err error) {
	if err != nil {
		log.Fatal("etcd-trigger: ", err)
	}
}

func main() {
	flag.Parse()
	machines := strings.Split(*machinesFlag, ",")
	notifies := strings.Split(*notifiesFlag, ",")
	if *triggerKey == "" {
		log.Fatal("-trigger flag required")
	}
	if *readKey == "" {
		readKey = triggerKey
	}
	log.Print("trigger: ", *triggerKey)
	log.Print("read: ", *readKey)
	log.Print("notifies: ", notifies)

	for {
		client := etcd.NewClient(machines)
		response, err := client.Watch(*triggerKey, 0, false, nil, nil)
		assert(err)
		triggerValue := response.Node.Value
		var value string

		if readKey != triggerKey {
			response, err := client.Get(*readKey, false, false)
			assert(err)
			value = response.Node.Value
		} else {
			value = triggerValue
		}

		log.Print("trigger value: ", triggerValue)
		log.Print("value: ", value)

		for _, url := range notifies {
			client := &http.Client{}
			request, err := http.NewRequest("PUT", url, strings.NewReader(value))
			assert(err)
			response, err := client.Do(request)
			assert(err)
			defer response.Body.Close()
			bytes, err := ioutil.ReadAll(response.Body)
			assert(err)
			body := string(bytes)
			log.Print("notify: ", url, "response: ", response.StatusCode, " ", body)
		}

		if *retriggerKey != "" {
			_, err := client.Set(*retriggerKey, triggerValue, 0)
			assert(err)
			log.Print("retriggered: ", *retriggerKey, " value: ", triggerValue)
		}
	}
}
