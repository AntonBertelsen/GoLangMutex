package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"io"
	"log"
	pb "mini_project_dis_sys"
	"net"
	"os"
)

const (
	port = ":50000"
)

type CriticalSectionServiceServer struct {
	pb.UnimplementedCriticalSectionServiceServer
	nodes []ClientNode
	queue chan *ClientNode
	done chan *pb.DoneMessage
}

type ClientNode struct {
	id int64
	channel chan *pb.AccessGranted
}

func (s *CriticalSectionServiceServer) RequestAccess(_ context.Context, msg *pb.RequestMessage) (*pb.MessageAcknowledgement, error) {

	for _, node := range s.nodes {
		//fmt.Printf("------------------------------------------------------------------- Nodeid: %v, %v \n", node.id, len(node.channel))

		if node.id == msg.Id {
			fmt.Printf("------------------------------------------------------------------------------------WRITING INTO QUEUE NODE WITH THE ID %v \n", msg.Id)
			s.queue <- &node
		}
		//fmt.Printf("------------------------------------------------------------------- Nodeid: %v, %v \n", node.id, len(node.channel))
	}
	log.Printf("----(debug: RECEIVED A REQUEST ACCESS MESSAGE FROM CLIENT %v) \n", msg.Id)
	return &pb.MessageAcknowledgement{}, nil
}

func (s *CriticalSectionServiceServer) Done(_ context.Context, msg *pb.DoneMessage) (*pb.MessageAcknowledgement, error) {

	s.done <- msg
	log.Printf("----(debug: RECEIVED A DONE MESSAGE FROM CLIENT %v) \n", msg.Id)
	return &pb.MessageAcknowledgement{}, nil
}

func (s *CriticalSectionServiceServer) Subscribe(joinMessage *pb.JoinMessage, stream pb.CriticalSectionService_SubscribeServer) error {
	log.Printf("----(debug: CLIENT NODE CONNECTED %v) \n", joinMessage.Id)

	node := ClientNode{
		id: joinMessage.Id,
		channel:    make(chan *pb.AccessGranted,10),
	}

	s.nodes = append(s.nodes, node)

	waitChannel := make(chan struct{})

	go func() {
		for {
			msg := <-node.channel
			log.Printf("----(debug: SENDING ACCESS GRANTED MESSAGE TO CLIENT %v) \n", node.id)
			stream.Send(msg)
		}
	}()
	<-waitChannel
	return nil
}

func newServer() *CriticalSectionServiceServer {
	s := &CriticalSectionServiceServer{
		nodes: make([]ClientNode, 0),
		queue: make(chan *ClientNode, 100),
		done: make(chan *pb.DoneMessage, 10),
	}
	return s
}

func main() {
	file, err := os.OpenFile("logFile.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	mw := io.MultiWriter(os.Stdout, file)
	log.SetOutput(mw)

	log.Println("--- SERVER ---")
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	server := newServer()

	pb.RegisterCriticalSectionServiceServer(s, server)
	log.Printf("server listening at %v", lis.Addr())
	go func() {
		server.done <- &pb.DoneMessage{}
		for{
			<- server.done
			next :=<- server.queue
			fmt.Printf("------------------------------------------------------------------------------------THE NEXT NODE HAS THE ID %v \n", next.id)
			next.channel <- &pb.AccessGranted{}
		}
	}()
	s.Serve(lis)
}
