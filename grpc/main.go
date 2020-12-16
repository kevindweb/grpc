package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"grpc/client"
)

// Request is used to hold starting data for server
type Request struct {
	Data    string
	Dist    int
	Systems bool
}

// Response is what server fills in
type Response struct {
	Data    string
	Updated int
}

type NotRequest struct {
	AnotherOne []string
}

type NotResponse struct {
	NotData []string
}

type test func(*client.Grpc, int) time.Duration

var args = Request{
	Data:    "world",
	Dist:    65,
	Systems: true,
}

func perfTest(grpc *client.Grpc, numReqs int) time.Duration {
	waitGroup := new(sync.WaitGroup)

	waitGroup.Add(numReqs)

	start := time.Now()
	for i := 0; i < numReqs; i++ {
		go func() {
			var res Response
			grpc.Call("ioBound", args, &res)

			waitGroup.Done()
		}()
	}

	waitGroup.Wait()
	return time.Since(start)
}

func averagePerformance(grpc *client.Grpc, fn test, numTests int, parallel int) {
	var average int64
	for i := 0; i < numTests; i++ {
		elapsed := fn(grpc, parallel)
		ns := int64(elapsed)
		average += ns
	}

	average /= int64(numTests)
	averageTime := time.Duration(average) * time.Nanosecond
	fmt.Printf("Average time for %d parallel tests was %s\n", numTests, averageTime)
}

func main() {
	grpc, err := client.Dial("tcp", "127.0.0.1", 4000)
	startTime := time.Now()
	if err != nil {
		os.Exit(1)
	}

	if len(os.Args) > 1 {
		// run with CLI arguments
		switch program := os.Args[1]; program {
		case "perfTest":
			averagePerformance(grpc, perfTest, 10, 100)
			// perfTest(grpc)
		default:
			fmt.Printf("Argument %s not recognized\n", program)
		}

		// don't run normal tests
		return
	}

	args := Request{
		Data:    "world",
		Dist:    65,
		Systems: true,
	}
	var res Response

	err = grpc.Call("modifyArg", args, &res)
	if err != nil {
		fmt.Println("Error here", err)
	} else {
		fmt.Printf("Response: %v\n", res)
	}

	second := NotRequest{
		AnotherOne: []string{"he makes distributed", "code"},
	}
	var secondRes NotResponse

	err = grpc.Call("differentFunc", second, &secondRes)
	if err != nil {
		fmt.Println("Error here", err)
	} else {
		fmt.Printf("Response: %v\n", secondRes)
	}
	fmt.Println("Time elapsed: ", time.Since(startTime))
}
