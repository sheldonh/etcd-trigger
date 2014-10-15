# etcd-trigger

`etcd-trigger` is a long-lived process that sends values from etcd to an HTTP end point on change.

It can watch one key and send the value of another, or it can send the value of the key it watches.

## Usage

Configuration is performed through environment variables.

* `ETCD_WATCH_KEY` - The etcd key to watch (mandatory, no default).
* `ETCD_NOTIFY_KEY` - The etcd key whose value to send to `NOTIFY_URL` (default: `ETCD_WATCH_KEY`).
* `ETCD_PEERS` - A whitespace-delimited list of one or more etcd peer URLs (default: `http://127.0.0.1:4001`).
* `ETCD_PORT_4001_TCP_ADDR` - The address of an etcd peer if `ETCD_PEERS` is not given (default: `127.0.0.1`).
* `ETCD_PORT_4001_TCP_PORT` - The port of an etcd peer if `ETCD_PEERS` is not given (default: `4001`).
* `NOTIFY_URL` - The HTTP end point to notify on change (default: `http://127.0.0.1:8080/`).
* `NOTIFY_PORT_8080_TCP_ADDR` - The address of the HTTP end point if `NOTIFY_URL` is not given (default: `127.0.0.1`).
* `NOTIFY_PORT_8080_TCP_PORT` - The port of the HTTP end point if `NOTIFY_URL` is not given (default: `8080`).
* `NOTIFY_PATH` - The path of the HTTP end point if `NOTIFY_URL` is not given (default: `/`).

The value of the etcd key named by `ETCD_NOTIFY_KEY` (or by `ETCD_WATCH_KEY` if not given)
is submitted as the body of an HTTP PUT to the `NOTIFY_URL` when the value of etcd modifiedIndex of `ETCD_WATCH_KEY` changes.

## Examples

The following docker process is used to apply redis master/slave topology changes to a linked redis dictator:

```
docker run -d \
	--link etcd:etcd \
	--link redis-1-dictator:notify \
	-e NOTIFY_PATH=/master
	-e ETCD_WATCH_KEY=/config/redis-1/topology-trigger \
	-e ETCD_NOTIFY_KEY=/config/redis-1/topology \
	sheldonh/etcd-trigger
```

In a less dynamic environment, the same might be accomplished as follows:

```
docker run -d \
	-e ETCD_PEERS="http://10.0.0.2:4001 http://10.0.0.3:4001 http://10.0.0.3:4001" \
	-e ETCD_WATCH_KEY=/config/redis-1/topology-trigger \
	-e ETCD_NOTIFY_KEY=/config/redis-1/topology \
	-e NOTIFY_URL="http://10.0.0.10:8080/master" \
	sheldonh/etcd-trigger
```
