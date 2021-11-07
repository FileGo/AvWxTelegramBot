FROM golang AS build-env
WORKDIR /app
ADD . /app/
RUN go get -d -v ./...
RUN go build -o /go/bin/app

FROM gcr.io/distroless/base
COPY --from=build-env /go/bin/app /
COPY --from=build-env /app/airports.json /airports.json
CMD ["/app"] 
