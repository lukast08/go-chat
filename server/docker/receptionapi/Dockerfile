FROM golang:1.22

WORKDIR /source

COPY . .

RUN make build-reception-api

CMD [ "./receptionapi" ]
