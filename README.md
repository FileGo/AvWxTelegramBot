# AvWxTelegramBot

![Build](https://github.com/FileGo/avwxtelegrambot/actions/workflows/build.yaml/badge.svg) ![Publish Docker Image](https://github.com/FileGo/avwxtelegrambot/actions/workflows/dockerhub.yaml/badge.svg) ![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/FileGo/avwxtelegrambot) [![Go Report Card](https://goreportcard.com/badge/github.com/FileGo/avwxtelegrambot)](https://goreportcard.com/report/github.com/FileGo/avwxtelegrambot)

This projects creats a [Telegram](https://telegram.org/) bot, written in Go that receives one or multiple airport identifiers (4-letter ICAO or 3-letter IATA), separated by comma or space and returns current (METAR) and forecasted weather (TAF) for the requested airports.

It uses NOAA Aviation Weather Center's [Text Data Server](https://aviationweather.gov/data/api/) to retrieve data. Requests require ICAO airport code. Default interval for METAR/TAF is 12 hours, but can be overriden by providing NOAA_INTERVAL environmental variable.

It also provides a JSON file (airports.json), which has been created with data from [OpenFlights](https://openflights.org/data.html#airport). It stores a large majority of worldwide airports and it is used to lookup ICAO and IATA codes as required.

In order to make use of Telegram, a bot needs to be [created](https://core.telegram.org/bots#6-botfather) and token passed as an environmental variable. By default it uses the *updates* method of retrieving requests, however if both WEBHOOK_URL and WEBHOOK_PORT environmental variables are set, it will utilise the webhook.

## CLI
Can be run as a CLI program:
```
$ TELEGRAM_TOKEN={insert Telegram bot token} ./AvWxTelegramBot
```

## Docker
Easier way to run it is as a Docker container through docker-compose. An example of `docker-compose.yml` file:

```
services:
    avwxtelegrambot:
        container_name: avwxtelegrambot
        image: filego/avwxtelegrambot:latest
        environment: 
            - TELEGRAM_TOKEN=insert_telegram_bot_token
            - NOAA_INTERVAL=12                  # optional
            - WEBHOOK_URL=https://my.server.com # optional
            - WEBHOOK_PORT=443                  # optional
        restart: unless-stopped
```

Project makes use of the following libraries:

* https://github.com/yanzay/tbot (Go library for Telegram Bot)
