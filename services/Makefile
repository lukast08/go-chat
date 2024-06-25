RECEPTION_API_BINARY_NAME=receptionapi
DISPATCHER_API_BINARY_NAME=dispatcherapi

build-reception-api:
	go build -o ${RECEPTION_API_BINARY_NAME} cmd/receptionapid/main.go

build-dispatcher-api:
	go build -o ${DISPATCHER_API_BINARY_NAME} cmd/dispatcherapid/main.go

clean:
	go clean
	rm ${RECEPTION_API_BINARY_NAME} ${DISPATCHER_API_BINARY_NAME}


