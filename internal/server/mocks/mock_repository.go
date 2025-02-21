// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/armadaproject/armada/internal/server/queue (interfaces: QueueRepository)
//
// Generated by this command:
//
//	mockgen -destination=./mock_repository.go -package=mocks github.com/armadaproject/armada/internal/server/queue QueueRepository
//

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	armadacontext "github.com/armadaproject/armada/internal/common/armadacontext"
	queue "github.com/armadaproject/armada/pkg/client/queue"
	gomock "go.uber.org/mock/gomock"
)

// MockQueueRepository is a mock of QueueRepository interface.
type MockQueueRepository struct {
	ctrl     *gomock.Controller
	recorder *MockQueueRepositoryMockRecorder
	isgomock struct{}
}

// MockQueueRepositoryMockRecorder is the mock recorder for MockQueueRepository.
type MockQueueRepositoryMockRecorder struct {
	mock *MockQueueRepository
}

// NewMockQueueRepository creates a new mock instance.
func NewMockQueueRepository(ctrl *gomock.Controller) *MockQueueRepository {
	mock := &MockQueueRepository{ctrl: ctrl}
	mock.recorder = &MockQueueRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockQueueRepository) EXPECT() *MockQueueRepositoryMockRecorder {
	return m.recorder
}

// CordonQueue mocks base method.
func (m *MockQueueRepository) CordonQueue(ctx *armadacontext.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CordonQueue", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// CordonQueue indicates an expected call of CordonQueue.
func (mr *MockQueueRepositoryMockRecorder) CordonQueue(ctx, name any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CordonQueue", reflect.TypeOf((*MockQueueRepository)(nil).CordonQueue), ctx, name)
}

// CreateQueue mocks base method.
func (m *MockQueueRepository) CreateQueue(arg0 *armadacontext.Context, arg1 queue.Queue) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateQueue", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateQueue indicates an expected call of CreateQueue.
func (mr *MockQueueRepositoryMockRecorder) CreateQueue(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateQueue", reflect.TypeOf((*MockQueueRepository)(nil).CreateQueue), arg0, arg1)
}

// DeleteQueue mocks base method.
func (m *MockQueueRepository) DeleteQueue(ctx *armadacontext.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteQueue", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteQueue indicates an expected call of DeleteQueue.
func (mr *MockQueueRepositoryMockRecorder) DeleteQueue(ctx, name any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteQueue", reflect.TypeOf((*MockQueueRepository)(nil).DeleteQueue), ctx, name)
}

// GetAllQueues mocks base method.
func (m *MockQueueRepository) GetAllQueues(ctx *armadacontext.Context) ([]queue.Queue, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAllQueues", ctx)
	ret0, _ := ret[0].([]queue.Queue)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAllQueues indicates an expected call of GetAllQueues.
func (mr *MockQueueRepositoryMockRecorder) GetAllQueues(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllQueues", reflect.TypeOf((*MockQueueRepository)(nil).GetAllQueues), ctx)
}

// GetQueue mocks base method.
func (m *MockQueueRepository) GetQueue(ctx *armadacontext.Context, name string) (queue.Queue, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetQueue", ctx, name)
	ret0, _ := ret[0].(queue.Queue)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetQueue indicates an expected call of GetQueue.
func (mr *MockQueueRepositoryMockRecorder) GetQueue(ctx, name any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetQueue", reflect.TypeOf((*MockQueueRepository)(nil).GetQueue), ctx, name)
}

// UncordonQueue mocks base method.
func (m *MockQueueRepository) UncordonQueue(ctx *armadacontext.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UncordonQueue", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// UncordonQueue indicates an expected call of UncordonQueue.
func (mr *MockQueueRepositoryMockRecorder) UncordonQueue(ctx, name any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UncordonQueue", reflect.TypeOf((*MockQueueRepository)(nil).UncordonQueue), ctx, name)
}

// UpdateQueue mocks base method.
func (m *MockQueueRepository) UpdateQueue(arg0 *armadacontext.Context, arg1 queue.Queue) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateQueue", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateQueue indicates an expected call of UpdateQueue.
func (mr *MockQueueRepositoryMockRecorder) UpdateQueue(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateQueue", reflect.TypeOf((*MockQueueRepository)(nil).UpdateQueue), arg0, arg1)
}
