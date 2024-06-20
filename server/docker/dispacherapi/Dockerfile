FROM golang:1.22

WORKDIR /source

COPY . .

RUN make build-dispatcher-api

CMD [ "./dispatcherapi" ]
