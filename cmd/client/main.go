package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/gidyon/grpc/chat/api"
	"google.golang.org/grpc"
	"os"
	"strings"
)

func main() {
	url := flag.String("url", "passthrough:///localhost:9090", "url of the chat server")
	flag.Parse()

	cc, err := grpc.Dial(*url, grpc.WithInsecure())
	if err != nil {
		logrus.Fatal(err)
	}

	chatClient := chat.NewChatRoomClient(cc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chatRoom, err := chatClient.Chat(ctx)
	if err != nil {
		logrus.Fatal(err)
	}

	s := bufio.NewScanner(os.Stdin)

	count := 0
prompt:
	if count > 3 {
		return
	}
	fmt.Print("Enter Username: ")
	s.Scan()
	name := s.Text()

	// Registration
	if err = chatRoom.Send(&chat.ChatMessage{
		UserName: name,
	}); err != nil {
		logrus.Warnf("%s\n Please try again", err.Error())
		count++
		goto prompt
	}

	startReceiving := make(chan struct{}, 0)

	// Sending messages
	go func() {
		// Send signal to receiver
		close(startReceiving)

		// Get os input and send to chats
		for s.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
				err = chatRoom.Send(&chat.ChatMessage{
					Message:  s.Text(),
					UserName: name,
				})
				if err != nil {
					logrus.Errorf("error sending message: %v", err)
					return
				}
			}
		}
	}()

	// Proceed only when signaled
	<-startReceiving

	for {
		msg, err := chatRoom.Recv()
		if err != nil {
			logrus.Errorf("exiting chat: %v", err)
			return
		}

		select {
		case <-ctx.Done():
			logrus.Errorf("exiting chat: %v", err)
			return
		default:
			if strings.HasPrefix(msg.Message, name) {
				msg.Message = fmt.Sprintf("you%s", strings.TrimPrefix(msg.Message, name))
			}
			fmt.Fprint(os.Stdout, msg.Message)
		}
	}
}
