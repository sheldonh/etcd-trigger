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

If `$RETRIGGER` is specified, `etcd-trigger` will write the value received from `$TRIGGER` to the key named by `$RETRIGGER` on success.
This allows multiple triggers to be chained.

If any notification is unsuccessful (either because of a network failure or an HTTP response that does not start with `2`),
any subsequent notifications and retrigger are not attempted for the current change. This does not prevent `etcd-trigger` from
continuing to watch the trigger and try again on next change.

## TODO

* Add back support for `ETCD_PORT_4001_TCP_ADDR` and `ETCD_PORT_4001_TCP_PORT`.
* Add back support for environment variable interpolation in NOTIFY URLs.

## Examples

The following docker process is used to apply redis master/slave topology changes to a linked redis dictator:

```
docker run -d \
	--link etcd:etcd \
	--link redis-1-dictator:dictator \
	--link redis-1-dnsd:dnsd \
	-e `NOTIFY_URLS=http://${DICTATOR_PORT_8080_TCP_ADDR}:${DICTATOR_PORT_8080_TCP_PORT}/master http://${DNSD_PORT_8080_TCP_ADDR}:${DNSD_PORT_8080_TCP_PORT}/dns'
	-e ETCD_WATCH_KEY=/config/redis-1/topology-trigger \
	-e ETCD_NOTIFY_KEY=/config/redis-1/topology \
	sheldonh/etcd-trigger
```

Note that shell variable syntax was used to inform `etcd-trigger` that it should interpolate values from the environment.

In a less dynamic environment, the same might be accomplished as follows:

```
docker run -d \
	-e ETCD_PEERS="http://10.0.0.2:4001 http://10.0.0.3:4001 http://10.0.0.3:4001" \
	-e ETCD_WATCH_KEY=/config/redis-1/topology-trigger \
	-e ETCD_NOTIFY_KEY=/config/redis-1/topology \
	-e NOTIFY_URLS="http://10.0.0.10:8080/master http://10.0.0.43/dns" \
	sheldonh/etcd-trigger
```
