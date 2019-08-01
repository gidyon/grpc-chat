service:
	protoc -Iapi/ --go_out=plugins=grpc:api/ chat.proto

build:
	go build -o server ./cmd/server/main.go && \
	go build -o client ./cmd/client/main.go

# help: 