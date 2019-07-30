service:
	protoc -Iapi/ --go_out=plugins=grpc:api/ chat.proto

build:
	go build -o client ./cmd/server/main.go && \
	go build -o server ./cmd/client/main.go

# help: 