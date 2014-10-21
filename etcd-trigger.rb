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

def get_notify_urls_from_env(env)
  if env['NOTIFY_URLS']
    urls = env['NOTIFY_URLS'].split.map { |x| interpolate(x) }
  else
    # Deprecated
    if env['NOTIFY_URL']
      [ env['NOTIFY_URL'] ]
    else
      address = env['NOTIFY_PORT_8080_TCP_ADDR'] || '127.0.0.1'
      port = env['NOTIFY_PORT_8080_TCP_PORT'] || '8080'
      path = env['NOTIFY_PATH'] || '/'
      [ 'http://' + address + ':' + port + path ]
    end
  end
end

def interpolate(x)
  x.gsub(/\$\{[A-Z_][A-Z0-9_]*\}/) do |match|
    envar = match[2, match.length - 3]
    ENV[envar] or raise "No such variable #{match}"
  end
end

def get_etcd
  Etcd.client(get_peers_from_env(ENV).sample)
end

watch_key = ENV['ETCD_WATCH_KEY'] or raise "no ETCD_WATCH_KEY given"
notify_key = ENV['ETCD_NOTIFY_KEY'] || ENV['ETCD_WATCH_KEY']
retrigger_key = ENV['ETCD_RETRIGGER_KEY']

notify_urls = get_notify_urls_from_env(ENV).map { |x| URI(x) }

$logger = Logger.new($stderr)
$logger.level = LOG_LEVELS[ENV['LOG_LEVEL'] || "info"]

loop do
  begin
    etcd = get_etcd
    $logger.info "watching #{watch_key}"

    $logger.debug "watching #{watch_key}"
    watch_node = etcd.watch(watch_key).node
    if notify_key != watch_key
      notify_node = etcd.get(notify_key).node
    else
      notify_node = watch_node
    end
    notify_urls.each do |notify_url|
      $logger.debug "notifying #{notify_url}:\n#{notify_node.value.chomp}"
      Net::HTTP.start(notify_url.host, notify_url.port) do |http|
        req = Net::HTTP::Put.new(notify_url)
        req.body = notify_node.value
        res = http.request(req)
        if res.code[0] == '2'
          $logger.info "notified #{notify_url}: #{res.body.chomp}"
        else
          $logger.error "error from #{notify_url}: #{res.body.chomp}"
        end
      end
    end
    if retrigger_key
      $logger.debug "retriggering #{retrigger_key}"
      etcd.set(retrigger_key, value: 1)
      $logger.info "retriggered #{retrigger_key}"
    end
  rescue Exception => e
    if e.is_a?(SignalException) or e.is_a?(SystemExit)
      raise
    else
      $logger.error "#{e.class}: #{e.message}\n\t#{e.backtrace.join("\n\t")}"
      sleep 1
    end
  end
end
