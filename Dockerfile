FROM golang:alpine
RUN apk add --update git gcc musl-dev
RUN mkdir /app
RUN addgroup -g 1024 appgroup
RUN adduser -S -D -H -h /app --uid 1024 --ingroup appgroup appuser
ADD . /app/
WORKDIR /app
RUN go mod download
WORKDIR /app/src/cmd/AvWxTelegramBot/
RUN go build -o AvWxTelegramBot .
RUN chown -R 1024:1024 /app/src/cmd/AvWxTelegramBot/
USER appuser
ENV dbfile=airports.db3
ENV logfile=error.log
ENV noaa_hrs=12
ENV telegram_token="{INSERT_TELEGRAM_TOKEN}"
CMD ["./AvWxTelegramBot"]