// Code generated by MockGen. DO NOT EDIT.
// Source: input/v2/input-stateless/stateless.go
package mocks

import (
	reflect "reflect"

	beat "github.com/elastic/beats/v7/libbeat/beat"
	v2 "github.com/elastic/inputrunner/input/v2"
	input_stateless "github.com/elastic/inputrunner/input/v2/input-stateless"
	gomock "github.com/golang/mock/gomock"
)

// MockInput is a mock of Input interface.
type MockInput struct {
	ctrl     *gomock.Controller
	recorder *MockInputMockRecorder
}

// MockInputMockRecorder is the mock recorder for MockInput.
type MockInputMockRecorder struct {
	mock *MockInput
}

// NewMockInput creates a new mock instance.
func NewMockInput(ctrl *gomock.Controller) *MockInput {
	mock := &MockInput{ctrl: ctrl}
	mock.recorder = &MockInputMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockInput) EXPECT() *MockInputMockRecorder {
	return m.recorder
}

// Name mocks base method.
func (m *MockInput) Name() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name.
func (mr *MockInputMockRecorder) Name() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Name", reflect.TypeOf((*MockInput)(nil).Name))
}

// Run mocks base method.
func (m *MockInput) Run(ctx v2.Context, publish input_stateless.Publisher) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Run", ctx, publish)
	ret0, _ := ret[0].(error)
	return ret0
}

// Run indicates an expected call of Run.
func (mr *MockInputMockRecorder) Run(ctx, publish interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Run", reflect.TypeOf((*MockInput)(nil).Run), ctx, publish)
}

// Test mocks base method.
func (m *MockInput) Test(arg0 v2.TestContext) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Test", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Test indicates an expected call of Test.
func (mr *MockInputMockRecorder) Test(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Test", reflect.TypeOf((*MockInput)(nil).Test), arg0)
}

// MockPublisher is a mock of Publisher interface.
type MockPublisher struct {
	ctrl     *gomock.Controller
	recorder *MockPublisherMockRecorder
}

// MockPublisherMockRecorder is the mock recorder for MockPublisher.
type MockPublisherMockRecorder struct {
	mock *MockPublisher
}

// NewMockPublisher creates a new mock instance.
func NewMockPublisher(ctrl *gomock.Controller) *MockPublisher {
	mock := &MockPublisher{ctrl: ctrl}
	mock.recorder = &MockPublisherMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPublisher) EXPECT() *MockPublisherMockRecorder {
	return m.recorder
}

// Publish mocks base method.
func (m *MockPublisher) Publish(arg0 beat.Event) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Publish", arg0)
}

// Publish indicates an expected call of Publish.
func (mr *MockPublisherMockRecorder) Publish(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Publish", reflect.TypeOf((*MockPublisher)(nil).Publish), arg0)
}
