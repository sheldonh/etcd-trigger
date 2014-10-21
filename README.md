# etcd-trigger

`etcd-trigger` is a long-lived process that sends values from etcd to an HTTP end point on change.

It can watch one key and send the value of another, or it can send the value of the key it watches.

## Usage

Configuration is performed through environment variables.

* `ETCD_WATCH_KEY` - The etcd key to watch (mandatory, no default).
* `ETCD_NOTIFY_KEY` - The etcd key whose value to send to `NOTIFY_URL` (default: `ETCD_WATCH_KEY`).
* `ETCD_RETRIGGER_KEY` - If specified, an etcd key to write after notification, in support of chaining (default: unset).
* `ETCD_PEERS` - A whitespace-delimited list of one or more etcd peer URLs (default: `http://127.0.0.1:4001`).
* `ETCD_PORT_4001_TCP_ADDR` - The address of an etcd peer if `ETCD_PEERS` is not given (default: `127.0.0.1`).
* `ETCD_PORT_4001_TCP_PORT` - The port of an etcd peer if `ETCD_PEERS` is not given (default: `4001`).
* `NOTIFY_URLS` - A space-delimited list of HTTP end points to notify on change (default: `http://127.0.0.1:8080/`).
* `NOTIFY_URL` - _Deprecated_.
* `NOTIFY_PORT_8080_TCP_ADDR` - _Deprecated_.
* `NOTIFY_PORT_8080_TCP_PORT` - _Deprecated_.
* `NOTIFY_PATH` - _Deprecated_.

The value of the etcd key named by `ETCD_NOTIFY_KEY` (or by `ETCD_WATCH_KEY` if not given)
is submitted as the body of an HTTP PUT to each URL in `NOTIFY_URLS` when the value of etcd modifiedIndex of `ETCD_WATCH_KEY` changes.

If `ETCD_RETRIGGER_KEY` is specified, `etcd-trigger` will write the value "1" to that key on success. This allows multiple
triggers to be chained.

If any notification is unsuccessful (either because of a network failure or an HTTP response that does not start with `2`),
and subsequent notifications and retrigger are not attempted.

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
