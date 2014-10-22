# etcd-trigger

`etcd-trigger` is a long-lived process that sends values from etcd to HTTP end points on change.

It can watch one key and send the value of another, or it can send the value of the key it watches.

## Usage

```
NAME:
   etcd-trigger - sends values from etcd to HTTP end points on change

USAGE:
   etcd-trigger [global options] command [command options] [arguments...]

VERSION:
   0.0.1

COMMANDS:
   help, h      Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --dns                                DNS server for SRV lookups (default: no SRV lookups) [$DNS]
   --dns-port '53'                      DNS server for SRV lookups (default: 53) [$DNS_PORT]
   --machines 'http://127.0.0.1:4001'   comma-separated list of etcd machines [$ETCD_MACHINES]
   --notifies 'http://127.0.0.1:8080/'  comma-separated list of URLs to notify [$NOTIFIES]
   --read                               etcd key whose value to send to notify URLs (default: same as --trigger) [$READ]
   --retrigger                          etcd key to write after notifications (default: no retrigger) [$RETRIGGER]
   --trigger                            etcd key to watch (required) [$TRIGGER]
   --help, -h                           show help
   --version, -v                        print the version
```

The value of the etcd key named by `$READ` (or by `$TRIGGER` if not given)
is submitted as the body of an HTTP PUT to each URL in `$NOTIFIES` when the etcd modifiedIndex of `$TRIGGER` changes.

If `$DNS` is specified, it is used to perform SRV lookups for hostnames in URLs that have no port specifier.

If `$RETRIGGER` is specified, `etcd-trigger` will write the value received from `$TRIGGER` to the key named by `$RETRIGGER` on success.
This allows multiple triggers to be chained.

If any notification is unsuccessful (either because of a network failure or an HTTP response that does not start with `2`),
any subsequent notifications and retrigger are not attempted for the current change. This does not prevent `etcd-trigger` from
continuing to watch the trigger and try again on next change.

## Examples

The following docker process is used to apply redis master/slave topology changes, looking up notify services using SRV records
from SkyDNS on a CoreOS cluster:

```
docker run -d \
	-e DNS=172.17.8.101 \
	-e MACHINES=http://172.17.8.101:4001,http://172.17.8.102:4001,http://172.17.8.103:4001 \
	-e NOTIFIES=http://redis-1-dictator.docker/master,http://redis-1-dnsd.docker/dns \
	-e TRIGGER=/config/redis-1/topology-trigger \
	-e READ=/config/redis-1/topology \
	sheldonh/etcd-trigger
```
