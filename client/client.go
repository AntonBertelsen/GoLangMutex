package main

import (
	"context"
	"google.golang.org/grpc"
	"io"
	"log"
	"math"
	"math/rand"
	pb "mini_project_dis_sys"
	"os"
	"time"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

var accessGranted = make(chan int, 10)

var id int64
var joined = make(chan bool)

func join(ctx context.Context, client pb.CriticalSectionServiceClient) {
	rand.Seed(time.Now().UnixNano())
	id = int64(rand.Intn(math.MaxInt64))

	joinMessage := pb.JoinMessage{
		Id: id,
	}

	stream, err := client.Subscribe(ctx, &joinMessage)
	check(err)
	log.Printf("----(debug: NODE JOINED) \n")
	joined <- true

	waitChannel := make(chan struct{})
	go func() {
		for {
			_, err := stream.Recv()

			if err == io.EOF {
				close(waitChannel)
				return
			}
			check(err)

			log.Printf("----(debug: GRANTED ACCESS TO CRITICAL SECTION) \n")
			accessGranted <- 1
			//fmt.Println("!!!! - wrote in to access granted channel")
		}
	}()
	<-waitChannel
}

func requestAccess(ctx context.Context, client pb.CriticalSectionServiceClient) {

	requestMessage := pb.RequestMessage{Id: id}

	_, err := client.RequestAccess(ctx, &requestMessage)
	check(err)

	log.Printf("----(debug: I SUCCESFULLY REQUESTED ACCESS TO THE CRITICAL SECTION) \n")
}

func doneWithAccess(ctx context.Context, client pb.CriticalSectionServiceClient) {

	doneMessage := pb.DoneMessage{Id: id}

	_, err := client.Done(ctx, &doneMessage)
	check(err)

	log.Printf("----(debug: I SUCCESFULLY INFORMED THE SERVER THAT I AM DONE WITH THE CRITICAL SECTION) \n")
}

func main() {
	dat, err := os.ReadFile("address.txt")
	check(err)
	address := string(dat)

	// Connect
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("Did not connect: %v", err)
	}
	defer conn.Close()

	log.Println("--- CLIENT NODE ---")

	// Create stream
	client := pb.NewCriticalSectionServiceClient(conn)
	go join(context.Background(), client)
	<-joined

	for {
		r := rand.Intn(4000) + 1000
		time.Sleep(time.Duration(r) * time.Millisecond)
		requestAccess(context.Background(), client)
		<-accessGranted
		r = rand.Intn(4000) + 1000
		time.Sleep(time.Duration(r) * time.Millisecond)
		doneWithAccess(context.Background(), client)
	}
}
