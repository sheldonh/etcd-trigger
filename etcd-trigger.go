package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/codegangsta/cli"
	"github.com/coreos/go-etcd/etcd"
)

func watch(c *etcd.Client, watch string, read string) (trigger string, value string, err error) {
	var r *etcd.Response

	log.Print("watching ", watch)
	r, err = c.Watch(watch, 0, false, nil, nil)
	if err != nil {
		return
	}
	trigger = r.Node.Value

	if read == watch {
		r, err = c.Get(read, false, false)
		if err != nil {
			return
		}
		value = r.Node.Value
	} else {
		value = trigger
	}

	return
}

func notify(url, value string) (err error) {
	var (
		body []byte
		res  *http.Response
		req  *http.Request
	)

	c := &http.Client{}
	req, err = http.NewRequest("PUT", url, strings.NewReader(value))
	if err != nil {
		return
	}
	res, err = c.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}
	log.Print("notified: ", url, " (", res.StatusCode, " ", string(body), ")")
	return
}

func retrigger(c *etcd.Client, key string, trigger string) (err error) {
	_, err = c.Set(key, trigger, 0)
	if err == nil {
		log.Print("retriggered: ", key, " value: ", trigger)
	}
	return
}

func main() {
	app := cli.NewApp()
	app.Name = "etcd-trigger"
	app.Usage = "sends values from etcd to HTTP end points on change"
	app.Version = "0.0.1"
	app.Flags = []cli.Flag{
		cli.StringFlag{"machines", "http://127.0.0.1:4001", "comma-separated list of etcd machines", "ETCD_MACHINES"},
		cli.StringFlag{"notifies", "http://127.0.0.1:8080/", "comma-separated list of URLs to notify", "NOTIFIES"},
		cli.StringFlag{"read", "", "etcd key whose value to send to notify URLs (default: same as --trigger)", "READ"},
		cli.StringFlag{"retrigger", "", "etcd key to write after notifications (default: no retrigger)", "RETRIGGER"},
		cli.StringFlag{"trigger", "", "etcd key to watch (required)", "TRIGGER"},
	}
	app.Action = func(c *cli.Context) {
		var (
			client         *etcd.Client
			err            error
			trigger, value string
			url            string
		)

		machines := strings.Split(c.String("machines"), ",")
		notifies := strings.Split(c.String("notifies"), ",")
		triggerKey := c.String("trigger")
		readKey := c.String("read")
		retriggerKey := c.String("retrigger")

		if triggerKey == "" {
			log.Fatal("-trigger flag required")
		}
		if readKey == "" {
			readKey = triggerKey
		}

		client = etcd.NewClient(machines)
		for {
			trigger, value, err = watch(client, triggerKey, readKey)
			if err != nil {
				goto Error
			}

			for _, url = range notifies {
				err = notify(url, value)
				if err != nil {
					goto Error
				}
			}

			if retriggerKey != "" {
				err = retrigger(client, triggerKey, trigger)
				if err != nil {
					goto Error
				}
			}
			continue
		Error:
			log.Print("error: ", err)
			time.Sleep(time.Second)
		}
	}
	app.Run(os.Args)
}
