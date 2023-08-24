package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"time"

	"github.com/gogo/protobuf/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	jsgrpc "github.com/armadaproject/armada/pkg/api/jobservice"
)

var (
	rng        *rand.Rand
	numJobs    = flag.Int("jobs", 100, "number of jobs per jobset")
	numJobSets = flag.Int("jobsets", 50, "number of jobsets")
)

type JSQuery struct {
	JobSetId int
	JobId    int
	Prefix   int
}

const workerPoolSize = 50

func init() {
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	flag.Parse()
}

func main() {
	ctx := context.Background()

	// Launch a jobservice client to query jobservice about a jobset
	conn, err := grpc.Dial("localhost:2000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	client := jsgrpc.NewJobServiceClient(conn)
	healthResp, err := client.Health(ctx, &types.Empty{})
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(healthResp.Status.String())

	prefix := rng.Intn(10000)

	numTasks := *numJobSets * *numJobs
	queries := make(chan JSQuery, numTasks)
	results := make(chan string, numTasks)

	for jobSetId := 0; jobSetId < *numJobSets; jobSetId++ {
		for jobId := 0; jobId < *numJobs; jobId++ {
			queries <- JSQuery{JobSetId: jobSetId, JobId: jobId, Prefix: prefix}
		}
	}

	// Start up the pool of workers. They will block until they have
	// something to do, which arrives to them via the jobs channel.
	for w := 1; w <= workerPoolSize; w++ {
		go queryJobStatus(ctx, conn, queries, results)
	}

	for n := 0; n < numTasks; n++ {
		fmt.Println(<-results)
	}
}

func queryJobStatus(ctx context.Context, conn *grpc.ClientConn, queries <-chan JSQuery, results chan<- string) {
	client := jsgrpc.NewJobServiceClient(conn)

	for query := range queries {
		// fmt.Printf("querying  Job Set %d, Job %d\n", query.JobSetId, query.JobId)
		resp, err := client.GetJobStatus(ctx, &jsgrpc.JobServiceRequest{
			JobId:    fmt.Sprintf("%d", query.JobId),
			JobSetId: fmt.Sprintf("%d", query.JobSetId),
			Queue:    "fake_queue",
		})
		if err != nil {
			results <- err.Error()
		} else {
			results <- fmt.Sprintf("%s - Job Set %d, Job %d", resp.State.String(), query.JobSetId, query.JobId)
		}
	}
}
