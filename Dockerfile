FROM golang:alpine
RUN apk update && apk upgrade && \
    apk add --no-cache gcc libc-dev git bash
WORKDIR /app
COPY go.mod go.sum ./
COPY . .
RUN go get -u
RUN go build .
CMD ["./AvWxTelegramBot"] 
