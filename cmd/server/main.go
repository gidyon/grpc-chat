package main

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/gidyon/grpc/chat/api"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"net"
	"strings"
	"sync"
)

func main() {
	s := grpc.NewServer()

	chatSRV := &chatRoomDS{
		members: make(map[string]chat.ChatRoom_ChatServer, 0),
		chats:   make(chan string, 0),
	}
	go chatSRV.worker()

	chat.RegisterChatRoomServer(s, chatSRV)

	lis, err := net.Listen("tcp", ":9090")
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
	// First message is used for registration
	msg, err := chatRoom.Recv()
	if err != nil {
		return err
	}
	if strings.Trim(msg.UserName, " ") == "" {
		return errors.New("empty username")
	}

	chatSRV.mu.Lock()

	// Check username is not taken
	if _, ok := chatSRV.members[msg.UserName]; ok {
		return errors.New("username is taken")
	}

	// Add to members
	chatSRV.members[msg.UserName] = chatRoom

	chatSRV.mu.Unlock()

	// Send join message
	select {
	case <-chatRoom.Context().Done():
		return chatRoom.Context().Err()
	case chatSRV.chats <- fmt.Sprintf("%s joined chat\n", msg.UserName):
	}

	// Subsequent chats messages they sent
	for {
		msg, err = chatRoom.Recv()
		if err != nil {
			return err
		}

		select {
		case <-chatRoom.Context().Done():
			return chatRoom.Context().Err()
		case chatSRV.chats <- fmt.Sprintf("%s: %s\n", msg.UserName, msg.Message):
		}
	}
}

func (chatSRV *chatRoomDS) worker() {
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
