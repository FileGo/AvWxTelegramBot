FROM golang AS build-env
WORKDIR /app
ADD . /app/
RUN go get -d -v ./...
ENV CGO_ENABLED=0
RUN go build -o /go/bin/app

FROM gcr.io/distroless/base
COPY --from=build-env /go/bin/app /
COPY --from=build-env /app/airports.json /
CMD ["/app"] 
