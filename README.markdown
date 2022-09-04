# MQTT Grafana Event Publisher

Publishes MQTT messages as annotations (events) to Grafana.

# Example

1. Subscribe to `home/frontdoor` and  `home/backdoor` at `mqtt.example.com` (authenticated as `alice` with the password `s3cret`).
1. Publish each message received on one of these topics as Grafana annotation to `grafana.example.com` (authenticated as `bob` with the password `geh3im`). The message payload is prefixed with `$topic: `.
1. Tag each annotation with `mqtt` and `home`.

```command
$ mqtt-grafana-event-publisher \
    --mqtt-url mqtts://alice:s3cret@mqtt.example.com \
    --grafana-url https://bob:geh3im@grafana.example.com \
    --verbose \
    --topic home/frontdoor \
    --topic home/backdoor \
    --tag mqtt \
    --tag home
```

As the URLs contain secrets, it is recommended to set them as environment variables  `MQTT_URL` `GRAFANA_URL` instead of passing them as command line parameters.

# Development

```command
$ find . -name '*.go' -type f | entr -r \
    go run cmd/mqtt-grafana-event-publisher \
      --verbose \
      --topic home/door \
      --tag mqtt \
      --tag home
```
