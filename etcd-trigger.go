package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/codegangsta/cli"
	"github.com/coreos/go-etcd/etcd"
	"github.com/jcomputing/dns-clb-go/clb"
)

func watch(c *etcd.Client, watch string, read string) (trigger string, value string, err error) {
	var r *etcd.Response

	log.Print("watching ", watch)
	r, err = c.Watch(watch, 0, false, nil, nil)
	if err != nil {
		return
	}
	trigger = r.Node.Value

	if read != watch {
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

func lookup(c clb.LoadBalancer, u string) (string, error) {
	if c == nil {
		return u, nil
	}

	x, err := url.Parse(u)
	if err != nil {
		return "", err
	}

	components := strings.Split(x.Host, ":")
	if len(components) == 1 {
		h := components[0]
		a, err := c.GetAddress(h)
		if err == nil {
			x.Host = fmt.Sprintf("%s:%d", a.Address, a.Port)
		}
	}

	return x.String(), nil
}

func notify(u, value string) (err error) {
	var (
		body []byte
		res  *http.Response
		req  *http.Request
	)

	c := &http.Client{}
	req, err = http.NewRequest("PUT", u, strings.NewReader(value))
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
	log.Print("notified: ", u, " (", res.StatusCode, " ", string(body), ")")
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
		cli.StringFlag{"dns", "", "DNS server for SRV lookups (default: no SRV lookups)", "DNS"},
		cli.StringFlag{"dns-port", "53", "DNS server for SRV lookups (default: 53)", "DNS_PORT"},
		cli.StringFlag{"machines", "http://127.0.0.1:4001", "comma-separated list of etcd machines", "ETCD_MACHINES"},
		cli.StringFlag{"notifies", "http://127.0.0.1:8080/", "comma-separated list of URLs to notify", "NOTIFIES"},
		cli.StringFlag{"read", "", "etcd key whose value to send to notify URLs (default: same as --trigger)", "READ"},
		cli.StringFlag{"retrigger", "", "etcd key to write after notifications (default: no retrigger)", "RETRIGGER"},
		cli.StringFlag{"trigger", "", "etcd key to watch (required)", "TRIGGER"},
	}
	app.Action = func(c *cli.Context) {
		var (
			client         *etcd.Client
			dnsClb         clb.LoadBalancer
			err            error
			trigger, value string
			u              string
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

		if c.String("dns") != "" {
			dnsClb = clb.NewClb(c.String("dns"), c.String("dns-port"), clb.RoundRobin)
		}

		client = etcd.NewClient(machines)
		for {
			trigger, value, err = watch(client, triggerKey, readKey)
			if err != nil {
				goto Error
			}

			for _, u = range notifies {
				u, err = lookup(dnsClb, u)
				if err != nil {
					goto Error
				}
				err = notify(u, value)
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
