#!/usr/bin/env ruby

require 'logger'
require 'net/http'

require 'rubygems'
require 'etcd'
require 'json'

LOG_LEVELS = {
  "fatal" => Logger::FATAL,
  "error" => Logger::ERROR,
  "warn"  => Logger::WARN,
  "info"  => Logger::INFO,
  "debug" => Logger::DEBUG
}

def get_peers_from_env(env)
  if env['ETCD_PEERS']
    env['ETCD_PEERS'].split.map do |peer|
      p = peer.dup
      peer.gsub!(/http(s?):\/\//, '')
      if $1 == "s"
        $logger.fatal "etcd SSL not currently supported"
        exit(1)
      end
      peer.gsub!(/\/.*/, '')
      host, port = peer.split(':')
      {host: host, port: port}
    end
  else
    [ {host: env['ETCD_PORT_4001_TCP_ADDR'], port: env['ETCD_PORT_4001_TCP_PORT']} ]
  end
end

def get_notify_url_from_env(env)
  if env['NOTIFY_URL']
    env['NOTIFY_URL']
  else
    address = env['NOTIFY_PORT_8080_TCP_ADDR'] || '127.0.0.1'
    port = env['NOTIFY_PORT_8080_TCP_PORT'] || '8080'
    path = env['NOTIFY_PATH'] || '/'
    'http://' + address + ':' + port + path
  end
end

def get_etcd
  Etcd.client(get_peers_from_env(ENV).sample)
end

watch_key = ENV['ETCD_WATCH_KEY'] or raise "no ETCD_WATCH_KEY given"
notify_key = ENV['ETCD_NOTIFY_KEY'] || ENV['ETCD_WATCH_KEY']
notify_url = URI(get_notify_url_from_env(ENV))

$logger = Logger.new($stderr)
$logger.level = LOG_LEVELS[ENV['LOG_LEVEL'] || "info"]

begin
  etcd = get_etcd
  $logger.info "watching #{watch_key}"

  loop do
    $logger.debug "watching #{watch_key}"
    watch_node = etcd.watch(watch_key).node
    if notify_key != watch_key
      notify_node = etcd.get(notify_key).node
    else
      notify_node = watch_node
    end
    $logger.debug "notifying #{notify_url}:\n#{notify_node.value.chomp}"
    Net::HTTP.start(notify_url.host, notify_url.port) do |http|
      req = Net::HTTP::Put.new(notify_url)
      req.body = notify_node.value
      res = http.request(req)
      if res.code[0] == '2'
        $logger.info "notification receiver success: #{res.body.chomp}"
        watch = true
      else
        $logger.error "error from #{notify_url}: #{res.body.chomp}"
        watch = false
      end
    end
  end
rescue Exception => e
  if e.is_a?(SignalException) or e.is_a?(SystemExit)
    raise
  else
    $logger.error "#{e.class}: #{e.message}\n\t#{e.backtrace.join("\n\t")}"
    sleep 1
  end
end
