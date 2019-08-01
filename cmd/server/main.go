package main

import (
	"flag"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/gidyon/grpc/chat/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net"
	"strings"
	"sync"
)

func main() {
	s := grpc.NewServer()

	// Create chat room
	chatSRV := &chatRoomDS{
		members: make(map[string]chat.ChatRoom_ChatServer, 0),
		chats:   make(chan string, 0),
	}
	// Handle streaming messages
	go chatSRV.run()

	chat.RegisterChatRoomServer(s, chatSRV)

	port := flag.String("port", ":9090", "port server is running")
	flag.Parse()

	lis, err := net.Listen("tcp", *port)
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Info("chat server started on port :9090")
	s.Serve(lis)
}

type chatRoomDS struct {
	mu      sync.Mutex // guards members
	members map[string]chat.ChatRoom_ChatServer
	chats   chan string
}

func (chatSRV *chatRoomDS) Chat(chatRoom chat.ChatRoom_ChatServer) error {
	// First message is used for registration and authentication
	msg, err := chatRoom.Recv()
	if err != nil {
		return status.Errorf(codes.OutOfRange, "couldn't receive message: %v", err)
	}

	if strings.TrimSpace(msg.UserName) == "" {
		return status.Error(codes.InvalidArgument, "missing user name")
	}

	chatSRV.mu.Lock()

	// Check username is not taken
	if _, ok := chatSRV.members[msg.UserName]; ok {
		return status.Error(codes.ResourceExhausted, "username is taken")
	}

	// Add to caht members
	chatSRV.members[msg.UserName] = chatRoom

	chatSRV.mu.Unlock()

	broadCastMsg := func(msg string) error {
		select {
		case <-chatRoom.Context().Done():
			return chatRoom.Context().Err()
		case chatSRV.chats <- fmt.Sprintf("%s joined chat\n", msg):
		}
		return nil
	}

	// Broadcast join message to all members
	broadCastMsg(fmt.Sprintf("%s joined chat\n", msg.UserName))
	if err != nil {
		return err
	}

	// Handle subsequent chats messages this member will send
	for {
		msg, err = chatRoom.Recv()
		if err != nil {
			return status.Errorf(codes.OutOfRange, "couldn't receive message: %v", err)
		}

		// Broadcast join message to all members
		broadCastMsg(fmt.Sprintf("%s: %s\n", msg.UserName, msg.Message))
		if err != nil {
			return err
		}
	}
}

func (chatSRV *chatRoomDS) run() {
	var err error
	for msg := range chatSRV.chats {
		chatMsg := &chat.ChatMessage{
			Message: msg,
		}
		// send message to all members
		for _, chatStream := range chatSRV.members {
			err = chatStream.Send(chatMsg)
			if err != nil {
				logrus.Errorf("error sending chat: %v", err)
				continue
			}
		}
	}
}
