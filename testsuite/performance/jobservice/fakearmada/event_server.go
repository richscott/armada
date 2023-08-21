package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gogo/protobuf/types"

	"github.com/armadaproject/armada/pkg/api"
)

type PerformanceTestEventServer struct{}

func NewPerformanceTestEventServer() *PerformanceTestEventServer {
	return &PerformanceTestEventServer{}
}

func (s *PerformanceTestEventServer) Report(ctx context.Context, message *api.EventMessage) (*types.Empty, error) {
	return &types.Empty{}, nil
}

func (s *PerformanceTestEventServer) ReportMultiple(ctx context.Context, message *api.EventList) (*types.Empty, error) {
	return &types.Empty{}, nil
}

// GetJobSetEvents streams back all events associated with a particular job set.
func (s *PerformanceTestEventServer) GetJobSetEvents(request *api.JobSetRequest, stream api.Event_GetJobSetEventsServer) error {
	// FIXME: Handle case where watch is not True.
	return s.serveSimulatedEvents(request, stream)
}

func (s *PerformanceTestEventServer) Health(ctx context.Context, cont_ *types.Empty) (*api.HealthCheckResponse, error) {
	return &api.HealthCheckResponse{Status: api.HealthCheckResponse_SERVING}, nil
}

func (s *PerformanceTestEventServer) Watch(req *api.WatchRequest, stream api.Event_WatchServer) error {
	request := &api.JobSetRequest{
		Id:             req.JobSetId,
		Watch:          true,
		FromMessageId:  req.FromId,
		Queue:          req.Queue,
		ErrorIfMissing: true,
		ForceLegacy:    req.ForceLegacy,
		ForceNew:       req.ForceNew,
	}
	return s.GetJobSetEvents(request, stream)
}

type scriptedMessage struct {
	Delay       time.Duration
	MessageFunc func(req *api.JobSetRequest, jobSetId, jobId int) *api.EventMessage
}

var messageScript = []*scriptedMessage{
	{ // Submitted
		Delay: time.Duration(1),
		MessageFunc: func(request *api.JobSetRequest, jobSetId, jobId int) *api.EventMessage {
			jobIdStr := fmt.Sprintf("%d", jobId)
			jobSetIdStr := fmt.Sprintf("%d", jobSetId)
			now := time.Now()

			return &api.EventMessage{
				Events: &api.EventMessage_Submitted{
					Submitted: &api.JobSubmittedEvent{
						JobId:    jobIdStr,
						JobSetId: jobSetIdStr,
						Queue:    request.Queue,
						Created:  now,
						Job: api.Job{
							Id:        jobIdStr,
							ClientId:  "",
							Queue:     request.Queue,
							JobSetId:  jobSetIdStr,
							Namespace: "fakeNamespace",
							Created:   now,
						},
					},
				},
			}
		},
	},
	{ // Queued
		Delay: time.Duration(time.Millisecond * 200),
		MessageFunc: func(request *api.JobSetRequest, jobSetId, jobId int) *api.EventMessage {
			jobIdStr := fmt.Sprintf("%d", jobId)
			jobSetIdStr := fmt.Sprintf("%d", jobSetId)
			now := time.Now()

			return &api.EventMessage{
				Events: &api.EventMessage_Queued{
					Queued: &api.JobQueuedEvent{
						JobId:    jobIdStr,
						JobSetId: jobSetIdStr,
						Queue:    request.Queue,
						Created:  now,
					},
				},
			}
		},
	},
	{ // Running
		Delay: time.Duration(time.Millisecond * 500),
		MessageFunc: func(request *api.JobSetRequest, jobSetId, jobId int) *api.EventMessage {
			jobIdStr := fmt.Sprintf("%d", jobId)
			jobSetIdStr := fmt.Sprintf("%d", jobSetId)
			now := time.Now()

			return &api.EventMessage{
				Events: &api.EventMessage_Running{
					Running: &api.JobRunningEvent{
						JobId:        jobIdStr,
						JobSetId:     jobSetIdStr,
						Queue:        request.Queue,
						Created:      now,
						ClusterId:    "fakeCluster",
						KubernetesId: "fakeK8s",
						NodeName:     "fakeNode",
						PodNumber:    1,
						PodName:      "fakePod",
						PodNamespace: "fakeNamespace",
					},
				},
			}
		},
	},
	{ // Success
		Delay: time.Duration(time.Second * 5),
		MessageFunc: func(request *api.JobSetRequest, jobSetId, jobId int) *api.EventMessage {
			jobIdStr := fmt.Sprintf("%d", jobId)
			jobSetIdStr := fmt.Sprintf("%d", jobSetId)
			now := time.Now()

			return &api.EventMessage{
				Events: &api.EventMessage_Succeeded{
					Succeeded: &api.JobSucceededEvent{
						JobId:        jobIdStr,
						JobSetId:     jobSetIdStr,
						Queue:        request.Queue,
						Created:      now,
						ClusterId:    "fakeCluster",
						KubernetesId: "fakeK8s",
						NodeName:     "fakeNode",
						PodNumber:    1,
						PodName:      "fakePod",
						PodNamespace: "fakeNamespace",
					},
				},
			}
		},
	},
}

func (s *PerformanceTestEventServer) serveSimulatedEvents(request *api.JobSetRequest,
	stream api.Event_GetJobSetEventsServer,
) error {
	jobSetId, err := strconv.Atoi(request.Id)
	if err != nil {
		return err
	}

	for jobId := 0; jobId < *NumJobs; jobId++ {
		for _, message := range messageScript {
			time.Sleep(message.Delay)
			err := stream.Send(&api.EventStreamMessage{
				Id:      fmt.Sprintf("%d-%d", jobSetId, jobId),
				Message: message.MessageFunc(request, jobSetId, jobId),
			})
			if err != nil {
				return err
			}
		}
	}

	// Keep the stream active but don't send anything
	time.Sleep(time.Minute * 10)

	return nil
}
