package scheduleringester

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/maps"
	"k8s.io/utils/strings/slices"

	"github.com/armadaproject/armada/internal/common/armadacontext"
	"github.com/armadaproject/armada/internal/common/ingest/metrics"
	"github.com/armadaproject/armada/internal/common/ingest/testfixtures"
	armadaslices "github.com/armadaproject/armada/internal/common/slices"
	"github.com/armadaproject/armada/internal/common/util"
	schedulerdb "github.com/armadaproject/armada/internal/scheduler/database"
)

const testQueueName = "test"

func TestWriteOps(t *testing.T) {
	jobIds := make([]string, 10)
	for i := range jobIds {
		jobIds[i] = util.NewULID()
	}
	runIds := make([]string, 10)
	for i := range runIds {
		runIds[i] = uuid.NewString()
	}
	scheduledAtPriorities := []int32{5, 10}
	tests := map[string]struct {
		Ops             []DbOperation
		ExpectNoUpdates bool
	}{
		"InsertJobs": {Ops: []DbOperation{
			InsertJobs{
				jobIds[0]: &schedulerdb.Job{JobID: jobIds[0], JobSet: "set1"},
				jobIds[1]: &schedulerdb.Job{JobID: jobIds[1], JobSet: "set2"},
			},
			InsertJobs{
				jobIds[2]: &schedulerdb.Job{JobID: jobIds[2], JobSet: "set1"},
				jobIds[3]: &schedulerdb.Job{JobID: jobIds[3], JobSet: "set2"},
			},
		}},
		"Submit Check": {Ops: []DbOperation{
			InsertJobs{
				jobIds[0]: &schedulerdb.Job{JobID: jobIds[0], JobSet: "set1"},
				jobIds[1]: &schedulerdb.Job{JobID: jobIds[1], JobSet: "set2"},
			},
			MarkJobsValidated{
				jobIds[0]: []string{"cpu"},
				jobIds[1]: []string{"gpu", "cpu"},
			},
		}},
		"InsertRuns": {Ops: []DbOperation{
			InsertJobs{
				jobIds[0]: &schedulerdb.Job{JobID: jobIds[0], Queue: testQueueName, JobSet: "set1"},
				jobIds[1]: &schedulerdb.Job{JobID: jobIds[1], Queue: testQueueName, JobSet: "set2"},
				jobIds[2]: &schedulerdb.Job{JobID: jobIds[2], Queue: testQueueName, JobSet: "set1"},
				jobIds[3]: &schedulerdb.Job{JobID: jobIds[3], Queue: testQueueName, JobSet: "set2"},
			},
			InsertRuns{
				runIds[0]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[0], RunID: runIds[0]}},
				runIds[1]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[1], RunID: runIds[1]}},
			},
			InsertRuns{
				runIds[2]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[2], RunID: runIds[2], ScheduledAtPriority: &scheduledAtPriorities[0]}},
				runIds[3]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[3], RunID: runIds[3], ScheduledAtPriority: &scheduledAtPriorities[1]}},
			},
			UpdateJobQueuedState{
				jobIds[0]: &JobQueuedStateUpdate{Queued: false, QueuedStateVersion: 1},
				jobIds[1]: &JobQueuedStateUpdate{Queued: false, QueuedStateVersion: 1},
				jobIds[2]: &JobQueuedStateUpdate{Queued: false, QueuedStateVersion: 1},
				jobIds[3]: &JobQueuedStateUpdate{Queued: false, QueuedStateVersion: 1},
			},
		}},
		"UpdateJobSetPriorities": {Ops: []DbOperation{
			InsertJobs{
				jobIds[0]: &schedulerdb.Job{JobID: jobIds[0], Queue: testQueueName, JobSet: "set1"},
				jobIds[1]: &schedulerdb.Job{JobID: jobIds[1], Queue: testQueueName, JobSet: "set2"},
				jobIds[2]: &schedulerdb.Job{JobID: jobIds[2], Queue: testQueueName, JobSet: "set1"},
				jobIds[3]: &schedulerdb.Job{JobID: jobIds[3], Queue: testQueueName, JobSet: "set2"},
				jobIds[4]: &schedulerdb.Job{JobID: jobIds[4], Queue: "queue-2", JobSet: "set1"},
			},
			UpdateJobSetPriorities{JobSetKey{queue: testQueueName, jobSet: "set1"}: 1},
		}},
		"UpdateJobPriorities": {Ops: []DbOperation{
			InsertJobs{
				jobIds[0]: &schedulerdb.Job{JobID: jobIds[0], Queue: testQueueName, JobSet: "set1"},
				jobIds[1]: &schedulerdb.Job{JobID: jobIds[1], Queue: testQueueName, JobSet: "set2"},
				jobIds[2]: &schedulerdb.Job{JobID: jobIds[2], Queue: testQueueName, JobSet: "set1"},
				jobIds[3]: &schedulerdb.Job{JobID: jobIds[3], Queue: testQueueName, JobSet: "set2"},
			},
			&UpdateJobPriorities{
				key: JobReprioritiseKey{
					JobSetKey: JobSetKey{
						queue:  testQueueName,
						jobSet: "set1",
					},
					Priority: 3,
				},
				jobIds: []string{jobIds[0], jobIds[2]},
			},
		}},
		"MarkRunsForJobPreemptRequested": {Ops: []DbOperation{
			InsertJobs{
				jobIds[0]: &schedulerdb.Job{JobID: jobIds[0], Queue: testQueueName, JobSet: "set1"},
				jobIds[1]: &schedulerdb.Job{JobID: jobIds[1], Queue: testQueueName, JobSet: "set1"},
				jobIds[2]: &schedulerdb.Job{JobID: jobIds[2], Queue: testQueueName, JobSet: "set2"},
				jobIds[3]: &schedulerdb.Job{JobID: jobIds[3], Queue: "queue-2", JobSet: "set1"},
				jobIds[4]: &schedulerdb.Job{JobID: jobIds[4], Queue: "queue-2", JobSet: "set2"},
			},
			InsertRuns{
				runIds[0]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[0], RunID: runIds[0], Queue: testQueueName, JobSet: "set1"}},
				runIds[1]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[1], RunID: runIds[1], Queue: testQueueName, JobSet: "set1"}},
				runIds[2]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[2], RunID: runIds[2], Queue: testQueueName, JobSet: "set2"}},
				runIds[3]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[3], RunID: runIds[3], Queue: "queue-2", JobSet: "set1"}},
				runIds[4]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[4], RunID: runIds[4], Queue: "queue-2", JobSet: "set2"}},
			},
			MarkRunsForJobPreemptRequested{JobSetKey{queue: testQueueName, jobSet: "set1"}: []string{jobIds[0], jobIds[1]}},
		}},
		"MarkJobSetsCancelRequested": {Ops: []DbOperation{
			InsertJobs{
				jobIds[0]: &schedulerdb.Job{JobID: jobIds[0], Queue: testQueueName, JobSet: "set1"},
				jobIds[1]: &schedulerdb.Job{JobID: jobIds[1], Queue: testQueueName, JobSet: "set2"},
				jobIds[2]: &schedulerdb.Job{JobID: jobIds[2], Queue: testQueueName, JobSet: "set1"},
				jobIds[3]: &schedulerdb.Job{JobID: jobIds[3], Queue: testQueueName, JobSet: "set2"},
				jobIds[4]: &schedulerdb.Job{JobID: jobIds[4], Queue: "queue-2", JobSet: "set1"},
			},
			MarkJobSetsCancelRequested{
				cancelUser: testfixtures.CancelUser,
				jobSets: map[JobSetKey]*JobSetCancelAction{
					{queue: testQueueName, jobSet: "set1"}: {cancelLeased: true, cancelQueued: true},
				},
			},
		}},
		"MarkJobSetsCancelRequested - Queued only": {Ops: []DbOperation{
			InsertJobs{
				jobIds[0]: &schedulerdb.Job{JobID: jobIds[0], Queue: testQueueName, JobSet: "set1", Queued: true},
				jobIds[1]: &schedulerdb.Job{JobID: jobIds[1], Queue: testQueueName, JobSet: "set2", Queued: true},
				jobIds[2]: &schedulerdb.Job{JobID: jobIds[2], Queue: testQueueName, JobSet: "set1", Queued: false},
				jobIds[3]: &schedulerdb.Job{JobID: jobIds[3], Queue: testQueueName, JobSet: "set2", Queued: false},
			},
			MarkJobSetsCancelRequested{
				cancelUser: testfixtures.CancelUser,
				jobSets: map[JobSetKey]*JobSetCancelAction{
					{queue: testQueueName, jobSet: "set1"}: {cancelLeased: false, cancelQueued: true},
				},
			},
		}},
		"MarkJobSetsCancelRequested - Leased only": {Ops: []DbOperation{
			InsertJobs{
				jobIds[0]: &schedulerdb.Job{JobID: jobIds[0], Queue: testQueueName, JobSet: "set1", Queued: true},
				jobIds[1]: &schedulerdb.Job{JobID: jobIds[1], Queue: testQueueName, JobSet: "set2", Queued: true},
				jobIds[2]: &schedulerdb.Job{JobID: jobIds[2], Queue: testQueueName, JobSet: "set1", Queued: false},
				jobIds[3]: &schedulerdb.Job{JobID: jobIds[3], Queue: testQueueName, JobSet: "set2", Queued: false},
			},
			MarkJobSetsCancelRequested{
				cancelUser: testfixtures.CancelUser,
				jobSets: map[JobSetKey]*JobSetCancelAction{
					{queue: testQueueName, jobSet: "set1"}: {cancelLeased: true, cancelQueued: false},
				},
			},
		}},
		"MarkJobsCancelRequested": {Ops: []DbOperation{
			InsertJobs{
				jobIds[0]: &schedulerdb.Job{JobID: jobIds[0], Queue: testQueueName, JobSet: "set1"},
				jobIds[1]: &schedulerdb.Job{JobID: jobIds[1], Queue: testQueueName, JobSet: "set2"},
				jobIds[2]: &schedulerdb.Job{JobID: jobIds[2], Queue: testQueueName, JobSet: "set1"},
				jobIds[3]: &schedulerdb.Job{JobID: jobIds[3], Queue: testQueueName, JobSet: "set2"},
			},
			MarkJobsCancelRequested{
				cancelUser: testfixtures.CancelUser,
				jobIds: map[JobSetKey][]string{
					{queue: testQueueName, jobSet: "set1"}: {jobIds[0]},
					{queue: testQueueName, jobSet: "set2"}: {jobIds[1]},
				},
			},
		}},
		"MarkJobsCancelled": {Ops: []DbOperation{
			InsertJobs{
				jobIds[0]: &schedulerdb.Job{JobID: jobIds[0], JobSet: "set1"},
				jobIds[1]: &schedulerdb.Job{JobID: jobIds[1], JobSet: "set2"},
				jobIds[2]: &schedulerdb.Job{JobID: jobIds[2], JobSet: "set1"},
				jobIds[3]: &schedulerdb.Job{JobID: jobIds[3], JobSet: "set2"},
			},
			InsertRuns{
				runIds[0]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[0], RunID: runIds[0]}},
				runIds[1]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[1], RunID: runIds[1]}},
				runIds[2]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[2], RunID: runIds[2]}},
				runIds[3]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[3], RunID: runIds[3]}},
			},
			MarkJobsCancelled{
				jobIds[0]: testfixtures.BaseTime,
				jobIds[1]: testfixtures.BaseTime.Add(time.Hour),
			},
		}},
		"MarkJobsSucceeded": {Ops: []DbOperation{
			InsertJobs{
				jobIds[0]: &schedulerdb.Job{JobID: jobIds[0], JobSet: "set1"},
				jobIds[1]: &schedulerdb.Job{JobID: jobIds[1], JobSet: "set2"},
				jobIds[2]: &schedulerdb.Job{JobID: jobIds[2], JobSet: "set1"},
				jobIds[3]: &schedulerdb.Job{JobID: jobIds[3], JobSet: "set2"},
			},
			MarkJobsSucceeded{
				jobIds[0]: true,
				jobIds[1]: true,
			},
		}},
		"MarkRunsPending": {Ops: []DbOperation{
			InsertJobs{
				jobIds[0]: &schedulerdb.Job{JobID: jobIds[0]},
				jobIds[1]: &schedulerdb.Job{JobID: jobIds[1]},
			},
			InsertRuns{
				runIds[0]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[0], RunID: runIds[0]}},
				runIds[1]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[1], RunID: runIds[1]}},
			},
			MarkRunsPending{
				runIds[0]: testfixtures.BaseTime,
				runIds[1]: testfixtures.BaseTime.Add(time.Hour),
			},
		}},
		"MarkRunsPreempted": {Ops: []DbOperation{
			InsertJobs{
				jobIds[0]: &schedulerdb.Job{JobID: jobIds[0]},
				jobIds[1]: &schedulerdb.Job{JobID: jobIds[1]},
			},
			InsertRuns{
				runIds[0]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[0], RunID: runIds[0]}},
				runIds[1]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[1], RunID: runIds[1]}},
			},
			MarkRunsPreempted{
				runIds[0]: testfixtures.BaseTime,
			},
		}},
		"MarkJobsFailed": {Ops: []DbOperation{
			InsertJobs{
				jobIds[0]: &schedulerdb.Job{JobID: jobIds[0], JobSet: "set1"},
				jobIds[1]: &schedulerdb.Job{JobID: jobIds[1], JobSet: "set2"},
				jobIds[2]: &schedulerdb.Job{JobID: jobIds[2], JobSet: "set1"},
				jobIds[3]: &schedulerdb.Job{JobID: jobIds[3], JobSet: "set2"},
			},
			MarkJobsFailed{
				jobIds[0]: true,
				jobIds[1]: true,
			},
		}},
		"MarkRunsSucceeded": {Ops: []DbOperation{
			InsertJobs{
				jobIds[0]: &schedulerdb.Job{JobID: jobIds[0], JobSet: "set1"},
				jobIds[1]: &schedulerdb.Job{JobID: jobIds[1], JobSet: "set2"},
				jobIds[2]: &schedulerdb.Job{JobID: jobIds[2], JobSet: "set1"},
				jobIds[3]: &schedulerdb.Job{JobID: jobIds[3], JobSet: "set2"},
			},
			InsertRuns{
				runIds[0]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[0], RunID: runIds[0]}},
				runIds[1]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[1], RunID: runIds[1]}},
				runIds[2]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[2], RunID: runIds[2]}},
				runIds[3]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[3], RunID: runIds[3]}},
			},
			MarkRunsSucceeded{
				runIds[0]: testfixtures.BaseTime,
				runIds[1]: testfixtures.BaseTime.Add(time.Hour),
			},
		}},
		"UpdateJobSchedulingInfo": {Ops: []DbOperation{
			InsertJobs{
				jobIds[0]: &schedulerdb.Job{JobID: jobIds[0], JobSet: "set1"},
				jobIds[1]: &schedulerdb.Job{JobID: jobIds[1], JobSet: "set2"},
			},
			UpdateJobSchedulingInfo{
				jobIds[0]: &JobSchedulingInfoUpdate{
					JobSchedulingInfo:        []byte("job-0 info update"),
					JobSchedulingInfoVersion: 1,
				},
				jobIds[1]: &JobSchedulingInfoUpdate{
					JobSchedulingInfo:        []byte("job-1 info update"),
					JobSchedulingInfoVersion: 2,
				},
			},
		}},
		"Insert JobRunErrors": {Ops: []DbOperation{
			InsertJobRunErrors{
				runIds[0]: &schedulerdb.JobRunError{
					RunID: runIds[0],
					JobID: jobIds[0],
					Error: []byte{0x1},
				},
				runIds[1]: &schedulerdb.JobRunError{
					RunID: runIds[1],
					JobID: jobIds[1],
					Error: []byte{0x2},
				},
			},
		}},
		"MarkRunsFailed": {Ops: []DbOperation{
			InsertJobs{
				jobIds[0]: &schedulerdb.Job{JobID: jobIds[0], JobSet: "set1"},
				jobIds[1]: &schedulerdb.Job{JobID: jobIds[1], JobSet: "set2"},
				jobIds[2]: &schedulerdb.Job{JobID: jobIds[2], JobSet: "set1"},
				jobIds[3]: &schedulerdb.Job{JobID: jobIds[3], JobSet: "set2"},
			},
			InsertRuns{
				runIds[0]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[0], RunID: runIds[0]}},
				runIds[1]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[1], RunID: runIds[1]}},
				runIds[2]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[2], RunID: runIds[2]}},
				runIds[3]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[3], RunID: runIds[3]}},
			},
			MarkRunsFailed{
				runIds[0]: &JobRunFailed{LeaseReturned: true, FailureTime: testfixtures.BaseTime},
				runIds[1]: &JobRunFailed{LeaseReturned: true, RunAttempted: true, FailureTime: testfixtures.BaseTime.Add(time.Hour)},
				runIds[2]: &JobRunFailed{LeaseReturned: false, FailureTime: testfixtures.BaseTime},
			},
		}},
		"MarkRunsRunning": {Ops: []DbOperation{
			InsertJobs{
				jobIds[0]: &schedulerdb.Job{JobID: jobIds[0], JobSet: "set1"},
				jobIds[1]: &schedulerdb.Job{JobID: jobIds[1], JobSet: "set2"},
				jobIds[2]: &schedulerdb.Job{JobID: jobIds[2], JobSet: "set1"},
				jobIds[3]: &schedulerdb.Job{JobID: jobIds[3], JobSet: "set2"},
			},
			InsertRuns{
				runIds[0]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[0], RunID: runIds[0]}},
				runIds[1]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[1], RunID: runIds[1]}},
				runIds[2]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[2], RunID: runIds[2]}},
				runIds[3]: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobIds[3], RunID: runIds[3]}},
			},
			MarkRunsRunning{
				runIds[0]: testfixtures.BaseTime,
				runIds[1]: testfixtures.BaseTime.Add(time.Hour),
			},
		}},
		"Insert PositionMarkers": {Ops: []DbOperation{
			&InsertPartitionMarker{
				markers: []*schedulerdb.Marker{
					{
						GroupID:     uuid.New(),
						PartitionID: 1,
					},
					{
						GroupID:     uuid.New(),
						PartitionID: 3,
					},
					{
						GroupID:     uuid.New(),
						PartitionID: 2,
					},
				},
			},
		}},
		"Upsert Single Executor Setting": {Ops: []DbOperation{
			UpsertExecutorSettings{
				"executor-1": &ExecutorSettingsUpsert{
					ExecutorID:   "executor-1",
					Cordoned:     true,
					CordonReason: "Bad Executor",
				},
			},
		}},
		"Upsert Multiple Executor Settings": {Ops: []DbOperation{
			UpsertExecutorSettings{
				"executor-1": &ExecutorSettingsUpsert{
					ExecutorID:   "executor-1",
					Cordoned:     true,
					CordonReason: "Bad Executor",
				},
				"executor-2": &ExecutorSettingsUpsert{
					ExecutorID:   "executor-2",
					Cordoned:     true,
					CordonReason: "Bad Executor",
				},
				"executor-3": &ExecutorSettingsUpsert{
					ExecutorID:   "executor-3",
					Cordoned:     false,
					CordonReason: "",
				},
			},
		}},
		"Delete Single Executor Setting": {Ops: []DbOperation{
			DeleteExecutorSettings{
				"executor-1": &ExecutorSettingsDelete{
					ExecutorID: "executor-1",
				},
			},
		}},
		"Delete Multiple Executor Settings": {Ops: []DbOperation{
			UpsertExecutorSettings{
				"executor-1": &ExecutorSettingsUpsert{
					ExecutorID:   "executor-1",
					Cordoned:     true,
					CordonReason: "Bad Executor",
				},
				"executor-2": &ExecutorSettingsUpsert{
					ExecutorID:   "executor-2",
					Cordoned:     true,
					CordonReason: "Bad Executor",
				},
				"executor-3": &ExecutorSettingsUpsert{
					ExecutorID:   "executor-3",
					Cordoned:     false,
					CordonReason: "",
				},
			},
			DeleteExecutorSettings{
				"executor-1": &ExecutorSettingsDelete{
					ExecutorID: "executor-1",
				},
				"executor-2": &ExecutorSettingsDelete{
					ExecutorID: "executor-2",
				},
				"executor-3": &ExecutorSettingsDelete{
					ExecutorID: "executor-3",
				},
			},
		}},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := schedulerdb.WithTestDb(func(_ *schedulerdb.Queries, db *pgxpool.Pool) error {
				schedulerDb := &SchedulerDb{db: db}
				serials := make(map[string]int64)
				for _, op := range tc.Ops {
					err := assertOpSuccess(t, schedulerDb, serials, addDefaultValues(op))
					if err != nil {
						return err
					}
				}
				return nil
			})
			assert.NoError(t, err)
		})
	}
}

func TestScoping(t *testing.T) {
	jobIds := make([]string, 10)
	for i := range jobIds {
		jobIds[i] = util.NewULID()
	}
	executorIds := []string{testfixtures.ExecutorId, testfixtures.ExecutorId2, testfixtures.ExecutorId3}
	tests := map[string]struct {
		Ops   []DbOperation
		Scope int
		Error bool
	}{
		"Single DbOp jobSet scope": {
			Ops: []DbOperation{
				InsertJobs{jobIds[0]: &schedulerdb.Job{JobID: jobIds[0], Queue: testQueueName, JobSet: "set1"}},
			},
			Scope: JobSetEventsLockKey,
			Error: false,
		},
		"Multiple DbOp jobSet scope": {
			Ops: []DbOperation{
				InsertJobs{jobIds[0]: &schedulerdb.Job{JobID: jobIds[0], Queue: testQueueName, JobSet: "set1"}}, // 1
				MarkJobSetsCancelRequested{
					cancelUser: testfixtures.CancelUser,
					jobSets: map[JobSetKey]*JobSetCancelAction{
						{queue: testQueueName, jobSet: "set1"}: {cancelQueued: true, cancelLeased: true},
					},
				}, // 2
				InsertJobs{jobIds[1]: &schedulerdb.Job{JobID: jobIds[1], Queue: testQueueName, JobSet: "set1"}}, // 3
				MarkJobSetsCancelRequested{
					cancelUser: testfixtures.CancelUser,
					jobSets: map[JobSetKey]*JobSetCancelAction{
						{queue: testQueueName, jobSet: "set2"}: {cancelQueued: true, cancelLeased: true},
					}, // 3
				},
				InsertJobs{jobIds[2]: &schedulerdb.Job{JobID: jobIds[2], Queue: testQueueName, JobSet: "set1"}}, // 3
			},
			Scope: JobSetEventsLockKey,
			Error: false,
		},
		"Single dBop no scope required": {
			Ops: []DbOperation{
				UpsertExecutorSettings{executorIds[0]: &ExecutorSettingsUpsert{
					ExecutorID:   testfixtures.ExecutorId,
					Cordoned:     true,
					CordonReason: testfixtures.ExecutorCordonReason,
				}},
			},
			Scope: InvalidLockKey,
			Error: true,
		},
		"Multiple dBop no scope required": {
			Ops: []DbOperation{
				UpsertExecutorSettings{executorIds[0]: &ExecutorSettingsUpsert{
					ExecutorID:   testfixtures.ExecutorId,
					Cordoned:     true,
					CordonReason: testfixtures.ExecutorCordonReason,
				}},
				DeleteExecutorSettings{executorIds[0]: &ExecutorSettingsDelete{
					ExecutorID: testfixtures.ExecutorId,
				}},
			},
			Scope: InvalidLockKey,
			Error: true,
		},
		// We shouldn't see this in practice
		"Mixed events dBop no scope required": {
			Ops: []DbOperation{
				UpsertExecutorSettings{executorIds[0]: &ExecutorSettingsUpsert{
					ExecutorID:   testfixtures.ExecutorId,
					Cordoned:     true,
					CordonReason: testfixtures.ExecutorCordonReason,
				}},
				DeleteExecutorSettings{executorIds[0]: &ExecutorSettingsDelete{
					ExecutorID: testfixtures.ExecutorId,
				}},
				InsertJobs{jobIds[0]: &schedulerdb.Job{JobID: jobIds[0], Queue: testQueueName, JobSet: "set1"}},
			},
			Scope: JobSetEventsLockKey,
			Error: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			actualScope, actualErr := getLockKey(tc.Ops)

			assert.Equal(t, tc.Scope, actualScope)
			assert.Equal(t, tc.Error, actualErr != nil)
		})
	}
}

func addDefaultValues(op DbOperation) DbOperation {
	switch o := op.(type) {
	case InsertJobs:
		for _, job := range o {
			job.Groups = make([]byte, 0)
			job.SubmitMessage = make([]byte, 0)
			job.SchedulingInfo = make([]byte, 0)
		}
	case InsertRuns:
	case UpdateJobSetPriorities:
	case MarkJobSetsCancelRequested:
	case MarkJobsCancelRequested:
	case MarkJobsSucceeded:
	case MarkJobsFailed:
	case *UpdateJobPriorities:
	case MarkRunsSucceeded:
	case MarkRunsFailed:
	case MarkRunsRunning:
	}
	return op
}

func assertOpSuccess(t *testing.T, schedulerDb *SchedulerDb, serials map[string]int64, op DbOperation) error {
	ctx, cancel := armadacontext.WithTimeout(armadacontext.Background(), 10*time.Second)
	defer cancel()

	// Apply the op to the database.
	err := pgx.BeginTxFunc(ctx, schedulerDb.db, pgx.TxOptions{
		IsoLevel:       pgx.ReadCommitted,
		AccessMode:     pgx.ReadWrite,
		DeferrableMode: pgx.Deferrable,
	}, func(tx pgx.Tx) error {
		return schedulerDb.WriteDbOp(ctx, tx, op)
	})
	if err != nil {
		return err
	}

	// Read back the state from the db to compare.
	queries := schedulerdb.New(schedulerDb.db)
	selectNewJobs := func(ctx *armadacontext.Context, serial int64) ([]schedulerdb.Job, error) {
		return queries.SelectNewJobs(ctx, schedulerdb.SelectNewJobsParams{Serial: serial, Limit: 1000})
	}
	switch expected := op.(type) {
	case InsertJobs:
		jobs, err := selectNewJobs(ctx, serials["jobs"])
		if err != nil {
			return errors.WithStack(err)
		}
		actual := make(InsertJobs)
		for _, job := range jobs {
			job := job
			actual[job.JobID] = &job
			serials["jobs"] = max(serials["jobs"], job.Serial)
			if v, ok := expected[job.JobID]; ok {
				v.Serial = job.Serial
				v.LastModified = job.LastModified
			}
		}
		for k, v := range expected {
			assert.Equal(t, v, actual[k])
		}
	case InsertRuns:
		jobs, err := selectNewJobs(ctx, 0)
		if err != nil {
			return errors.WithStack(err)
		}
		jobIds := make([]string, 0)
		for _, job := range jobs {
			jobIds = append(jobIds, job.JobID)
		}

		runs, err := queries.SelectNewRunsForJobs(ctx, schedulerdb.SelectNewRunsForJobsParams{
			Serial: serials["runs"],
			JobIds: jobIds,
		})
		if err != nil {
			return errors.WithStack(err)
		}
		actual := make(InsertRuns)
		for _, run := range runs {
			run := run
			actual[run.RunID] = &JobRunDetails{Queue: testQueueName, DbRun: &run}
			serials["runs"] = max(serials["runs"], run.Serial)
			if v, ok := expected[run.RunID]; ok {
				v.DbRun.Serial = run.Serial
				v.DbRun.LastModified = run.LastModified
			}
		}
		assert.Equal(t, expected, actual)
	case UpdateJobQueuedState:
		jobs, err := selectNewJobs(ctx, serials["jobs"])
		if err != nil {
			return errors.WithStack(err)
		}
		numChanged := 0
		for _, job := range jobs {
			if e, ok := expected[job.JobID]; ok {
				assert.Equal(t, e.Queued, job.Queued)
				assert.Equal(t, e.QueuedStateVersion, job.QueuedVersion)
				numChanged++
			}
		}
		assert.Greater(t, numChanged, 0)
	case UpdateJobSchedulingInfo:
		jobs, err := selectNewJobs(ctx, serials["jobs"])
		if err != nil {
			return errors.WithStack(err)
		}
		numChanged := 0
		for _, job := range jobs {
			if e, ok := expected[job.JobID]; ok {
				assert.Equal(t, e.JobSchedulingInfoVersion, job.SchedulingInfoVersion)
				assert.Equal(t, e.JobSchedulingInfo, job.SchedulingInfo)
				numChanged++
			}
		}
		assert.Greater(t, numChanged, 0)
	case UpdateJobSetPriorities:
		jobs, err := selectNewJobs(ctx, serials["jobs"])
		if err != nil {
			return errors.WithStack(err)
		}
		numChanged := 0
		for _, job := range jobs {
			if e, ok := expected[JobSetKey{queue: job.Queue, jobSet: job.JobSet}]; ok {
				assert.Equal(t, e, job.Priority)
				numChanged++
			}
		}
		assert.Greater(t, numChanged, 0)
	case MarkJobSetsCancelRequested:
		jobs, err := selectNewJobs(ctx, serials["jobs"])
		if err != nil {
			return errors.WithStack(err)
		}
		numChanged := 0
		jobIds := make([]string, 0)
		for _, job := range jobs {
			if _, ok := expected.jobSets[JobSetKey{queue: job.Queue, jobSet: job.JobSet}]; ok {
				assert.True(t, job.CancelByJobsetRequested)
				numChanged++
				jobIds = append(jobIds, job.JobID)
			}
		}
		assert.Greater(t, numChanged, 0)
	case MarkJobsCancelRequested:
		jobs, err := selectNewJobs(ctx, serials["jobs"])
		if err != nil {
			return errors.WithStack(err)
		}
		numChanged := 0
		jobIds := make([]string, 0)
		for _, job := range jobs {
			if _, ok := expected.jobIds[JobSetKey{queue: job.Queue, jobSet: job.JobSet}]; ok {
				assert.True(t, job.CancelRequested)
				numChanged++
				jobIds = append(jobIds, job.JobID)
			}
		}
		assert.Equal(t, len(expected.jobIds), numChanged)
	case MarkRunsForJobPreemptRequested:
		jobs, err := selectNewJobs(ctx, 0)
		if err != nil {
			return errors.WithStack(err)
		}

		runsChanged := 0
		for _, job := range jobs {
			if req, ok := expected[JobSetKey{queue: job.Queue, jobSet: job.JobSet}]; ok {
				runs, err := queries.SelectNewRunsForJobs(ctx, schedulerdb.SelectNewRunsForJobsParams{
					Serial: serials["runs"],
					JobIds: []string{job.JobID},
				})
				if err != nil {
					return errors.WithStack(err)
				}
				for _, run := range runs {
					if slices.Contains(req, run.JobID) {
						assert.True(t, run.PreemptRequested)
						runsChanged++
					}
				}
			}
		}

		assert.Greater(t, runsChanged, 0)
	case MarkJobsCancelled:
		jobs, err := selectNewJobs(ctx, serials["jobs"])
		if err != nil {
			return errors.WithStack(err)
		}
		numChanged := 0
		jobIds := make([]string, 0)
		for _, job := range jobs {
			if _, ok := expected[job.JobID]; ok {
				assert.True(t, job.Cancelled)
				numChanged++
				jobIds = append(jobIds, job.JobID)
			}
		}
		assert.Equal(t, len(expected), numChanged)

		runs, err := queries.SelectNewRunsForJobs(ctx, schedulerdb.SelectNewRunsForJobsParams{
			Serial: serials["runs"],
			JobIds: jobIds,
		})
		if err != nil {
			return errors.WithStack(err)
		}
		runsChanged := 0
		for _, run := range runs {
			if _, ok := expected[run.JobID]; ok {
				assert.True(t, run.Cancelled)
				assert.Equal(t, expected[run.JobID], run.TerminatedTimestamp.UTC())
				runsChanged++
			}
		}
		assert.Equal(t, len(expected), runsChanged)
	case MarkJobsSucceeded:
		jobs, err := selectNewJobs(ctx, serials["jobs"])
		if err != nil {
			return errors.WithStack(err)
		}
		numChanged := 0
		for _, job := range jobs {
			if _, ok := expected[job.JobID]; ok {
				assert.True(t, job.Succeeded)
				numChanged++
			}
		}
		assert.Equal(t, len(expected), numChanged)
	case MarkJobsFailed:
		jobs, err := selectNewJobs(ctx, serials["jobs"])
		if err != nil {
			return errors.WithStack(err)
		}
		numChanged := 0
		for _, job := range jobs {
			if _, ok := expected[job.JobID]; ok {
				assert.True(t, job.Failed)
				numChanged++
			}
		}
		assert.Equal(t, len(expected), numChanged)
	case *UpdateJobPriorities:
		jobs, err := selectNewJobs(ctx, serials["jobs"])
		if err != nil {
			return errors.WithStack(err)
		}
		numChanged := 0
		for _, job := range jobs {
			jobSetKey := JobSetKey{queue: job.Queue, jobSet: job.JobSet}
			if expected.key.JobSetKey == jobSetKey {
				assert.Equal(t, expected.key.Priority, job.Priority)
				numChanged++
			}
		}
		assert.Equal(t, len(expected.jobIds), numChanged)
	case MarkRunsSucceeded:
		jobs, err := selectNewJobs(ctx, 0)
		if err != nil {
			return errors.WithStack(err)
		}
		jobIds := make([]string, 0)
		for _, job := range jobs {
			jobIds = append(jobIds, job.JobID)
		}

		runs, err := queries.SelectNewRunsForJobs(ctx, schedulerdb.SelectNewRunsForJobsParams{
			Serial: serials["runs"],
			JobIds: jobIds,
		})
		if err != nil {
			return errors.WithStack(err)
		}
		numChanged := 0
		for _, run := range runs {
			if _, ok := expected[run.RunID]; ok {
				assert.True(t, run.Succeeded)
				assert.Equal(t, expected[run.RunID], run.TerminatedTimestamp.UTC())
				numChanged++
			}
		}
		assert.Equal(t, len(expected), len(runs))
	case MarkRunsFailed:
		jobs, err := selectNewJobs(ctx, 0)
		if err != nil {
			return errors.WithStack(err)
		}
		jobIds := make([]string, 0)
		for _, job := range jobs {
			jobIds = append(jobIds, job.JobID)
		}

		runs, err := queries.SelectNewRunsForJobs(ctx, schedulerdb.SelectNewRunsForJobsParams{
			Serial: serials["runs"],
			JobIds: jobIds,
		})
		if err != nil {
			return errors.WithStack(err)
		}
		numChanged := 0
		for _, run := range runs {
			if expectedRun, ok := expected[run.RunID]; ok {
				assert.True(t, run.Failed)
				assert.Equal(t, expectedRun.LeaseReturned, run.Returned)
				assert.Equal(t, expectedRun.RunAttempted, run.RunAttempted)
				assert.Equal(t, expectedRun.FailureTime, run.TerminatedTimestamp.UTC())
				numChanged++
			}
		}
		assert.Equal(t, len(expected), len(runs))
	case MarkRunsRunning:
		jobs, err := selectNewJobs(ctx, 0)
		if err != nil {
			return errors.WithStack(err)
		}
		jobIds := make([]string, 0)
		for _, job := range jobs {
			jobIds = append(jobIds, job.JobID)
		}

		runs, err := queries.SelectNewRunsForJobs(ctx, schedulerdb.SelectNewRunsForJobsParams{
			Serial: serials["runs"],
			JobIds: jobIds,
		})
		if err != nil {
			return errors.WithStack(err)
		}
		numChanged := 0
		for _, run := range runs {
			if _, ok := expected[run.RunID]; ok {
				assert.True(t, run.Running)
				assert.Equal(t, expected[run.RunID], run.RunningTimestamp.UTC())
				numChanged++
			}
		}
		assert.Equal(t, len(expected), len(runs))
	case MarkRunsPending:
		jobs, err := selectNewJobs(ctx, 0)
		if err != nil {
			return errors.WithStack(err)
		}
		jobIds := make([]string, 0)
		for _, job := range jobs {
			jobIds = append(jobIds, job.JobID)
		}
		runs, err := queries.SelectNewRunsForJobs(ctx, schedulerdb.SelectNewRunsForJobsParams{
			Serial: serials["runs"],
			JobIds: jobIds,
		})
		if err != nil {
			return errors.WithStack(err)
		}
		numChanged := 0
		for _, run := range runs {
			if _, ok := expected[run.RunID]; ok {
				assert.True(t, run.Pending)
				assert.Equal(t, expected[run.RunID], run.PendingTimestamp.UTC())
				numChanged++
			}
		}
		assert.Equal(t, numChanged, 2)
		assert.Equal(t, len(expected), len(runs))
	case MarkRunsPreempted:
		jobs, err := selectNewJobs(ctx, 0)
		if err != nil {
			return errors.WithStack(err)
		}
		jobIds := make([]string, 0)
		for _, job := range jobs {
			jobIds = append(jobIds, job.JobID)
		}
		runs, err := queries.SelectNewRunsForJobs(ctx, schedulerdb.SelectNewRunsForJobsParams{
			Serial: serials["runs"],
			JobIds: jobIds,
		})
		if err != nil {
			return errors.WithStack(err)
		}
		numChanged := 0
		for _, run := range runs {
			if _, ok := expected[run.RunID]; ok {
				assert.True(t, run.Preempted)
				assert.Equal(t, expected[run.RunID], run.PreemptedTimestamp.UTC())
				numChanged++
			}
		}
		assert.Equal(t, numChanged, 1)
		assert.Equal(t, len(expected), len(runs))
	case InsertJobRunErrors:
		expectedIds := maps.Keys(expected)
		as, err := queries.SelectRunErrorsById(ctx, expectedIds)
		if err != nil {
			return errors.WithStack(err)
		}
		actual := make(InsertJobRunErrors, len(as))
		for _, a := range as {
			actual[a.RunID] = &schedulerdb.JobRunError{
				RunID: a.RunID,
				JobID: a.JobID,
				Error: a.Error,
			}
		}
		assert.Equal(t, expected, actual)
	case *InsertPartitionMarker:
		actual, err := queries.SelectAllMarkers(ctx)
		require.NoError(t, err)
		require.Equal(t, len(expected.markers), len(actual))
		for i, expectedMarker := range actual {
			actualMarker := actual[i]
			assert.Equal(t, expectedMarker.GroupID, actualMarker.GroupID)
			assert.Equal(t, expectedMarker.PartitionID, actualMarker.PartitionID)
			assert.Equal(t, expectedMarker.Created, actualMarker.Created)
		}
	case MarkJobsValidated:
		jobs, err := selectNewJobs(ctx, serials["jobs"])
		if err != nil {
			return errors.WithStack(err)
		}
		numChanged := 0
		jobIds := make([]string, 0)
		for _, job := range jobs {
			if _, ok := expected[job.JobID]; ok {
				assert.True(t, job.Validated)
				numChanged++
				jobIds = append(jobIds, job.JobID)
			}
		}
		assert.Equal(t, len(expected), numChanged)
	case UpsertExecutorSettings:
		allSettings, err := queries.SelectAllExecutorSettings(ctx)
		if err != nil {
			return errors.WithStack(err)
		}
		filtered := armadaslices.Filter(allSettings, func(e schedulerdb.ExecutorSetting) bool {
			_, ok := expected[e.ExecutorID]
			return ok
		})

		actual := UpsertExecutorSettings{}
		for _, a := range filtered {
			actual[a.ExecutorID] = &ExecutorSettingsUpsert{
				ExecutorID:   a.ExecutorID,
				Cordoned:     a.Cordoned,
				CordonReason: a.CordonReason,
			}
		}
		assert.Equal(t, expected, actual)
	case DeleteExecutorSettings:
		allSettings, err := queries.SelectAllExecutorSettings(ctx)
		if err != nil {
			return errors.WithStack(err)
		}
		filtered := armadaslices.Filter(allSettings, func(e schedulerdb.ExecutorSetting) bool {
			_, ok := expected[e.ExecutorID]
			return ok
		})
		assert.Equal(t, 0, len(filtered))
	default:
		return errors.Errorf("received unexpected op %+v", op)
	}
	return nil
}

func TestStore(t *testing.T) {
	jobId := util.ULID().String()
	runId := uuid.NewString()
	ops := []DbOperation{
		InsertJobs{
			jobId: &schedulerdb.Job{
				JobID:          jobId,
				JobSet:         "set1",
				Groups:         make([]byte, 0),
				SubmitMessage:  make([]byte, 0),
				SchedulingInfo: make([]byte, 0),
			},
		},
		InsertRuns{
			runId: &JobRunDetails{Queue: testQueueName, DbRun: &schedulerdb.Run{JobID: jobId, RunID: runId}},
		},
	}
	ctx, cancel := armadacontext.WithTimeout(armadacontext.Background(), 5*time.Second)
	defer cancel()
	err := schedulerdb.WithTestDb(func(q *schedulerdb.Queries, db *pgxpool.Pool) error {
		schedulerDb := NewSchedulerDb(db, metrics.NewMetrics("test"), time.Second, time.Second, 10*time.Second)
		err := schedulerDb.Store(ctx, &DbOperationsWithMessageIds{Ops: ops})
		require.NoError(t, err)

		jobIds, err := q.SelectAllJobIds(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{jobId}, jobIds)

		runIds, err := q.SelectAllRunIds(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{runId}, runIds)

		return nil
	})
	require.NoError(t, err)
}

func max[E constraints.Ordered](a, b E) E {
	if a > b {
		return a
	}
	return b
}
