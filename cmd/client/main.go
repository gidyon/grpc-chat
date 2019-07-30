package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/Sirupsen/logrus"
	chat "github.com/gidyon/grpc/chat/api"
	"google.golang.org/grpc"
	"os"
	"strings"
	"sync"
)

func main() {
	cc, err := grpc.Dial(":9090", grpc.WithInsecure())
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
		logrus.Warn(err.Error())
		count++
		goto prompt
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)

	startReceiving := make(chan struct{}, 0)

	// Sending messages
	go func() {
		defer func() {
			cancel()
			wg.Done()
		}()

		// Send signal to receiver
		close(startReceiving)

		// Start receiving and sending chats
		for s.Scan() {

			msg := s.Text()
			if strings.ToLower(msg) == "quit" {
				return
			}

			select {
			case <-ctx.Done():
				return
			default:
				err = chatRoom.Send(&chat.ChatMessage{
					Message:  msg,
					UserName: name,
				})
				if err != nil {
					logrus.Errorf("error sending message: %v", err)
					return
				}
			}
		}
	}()

	// Receiving messages
	go func() {
		defer func() {
			cancel()
			wg.Done()
		}()

		// Proceed only when signaled
		<-startReceiving

		for {
			msg, err := chatRoom.Recv()
			if err != nil {
				return
			}

			select {
			case <-ctx.Done():
				return
			default:
				if strings.HasPrefix(msg.Message, name) {
					msg.Message = fmt.Sprintf("you%s", strings.TrimPrefix(msg.Message, name))
				}
				fmt.Fprint(os.Stdout, msg.Message)
			}
		}
	}()

	// Wait
	wg.Wait()
	logrus.Infoln("exiting chat...")
}
