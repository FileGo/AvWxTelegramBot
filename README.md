# AvWxTelegramBot

This projects creats a [Telegram](https://telegram.org/) bot, written in Go that receives one or multiple airport identifiers (4-letter ICAO or 3-letter IATA), separated by comma or space and returns current (METAR) and forecasted weather (TAF) for the requested airports.

It uses NOAA Aviation Weather Center's [Text Data Server](https://www.aviationweather.gov/dataserver) to retrieve data. Requests require ICAO airport code.

It also provides a SQLite3 database (airports.db3), which has been created with data from [OpenFlights](https://openflights.org/data.html#airport). It stores a large majority of worldwide airports and it is used to lookup ICAO and IATA codes as required.

In order to make use of Telegram, a bot needs to be [created](https://core.telegram.org/bots#6-botfather) and token inserted into Dockerfile.

Usage:

`$ docker build -t avwxtelegrambot .`

`$ docker start --name avwxtelegrambot avwxtelegrambot`

Project makes use of the following libraries:

* https://github.com/yanzay/tbot (Go library for Telegram Bot)
* https://github.com/mattn/go-sqlite3 (SQLite3 driver)
