FROM golang:1.22

WORKDIR /app

COPY . /app

RUN make build

CMD [ "./gochat" ]
