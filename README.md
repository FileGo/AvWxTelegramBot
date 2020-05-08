# AvWxTelegramBot

This projects creats a [Telegram](https://telegram.org/) bot, written in Go that receives one or multiple airport identifiers (4-letter ICAO or 3-letter IATA), separated by comma or space and returns current (METAR) and forecasted weather (TAF) for the requested airports.

It uses NOAA Aviation Weather Center's [Text Data Server](https://www.aviationweather.gov/dataserver) to retrieve data.
