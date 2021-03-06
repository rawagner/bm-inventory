// Code generated by MockGen. DO NOT EDIT.
// Source: job.go

// Package job is a generated GoMock package.
package job

import (
	context "context"
	gomock "github.com/golang/mock/gomock"
	runtime "k8s.io/apimachinery/pkg/runtime"
	reflect "reflect"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

// MockAPI is a mock of API interface.
type MockAPI struct {
	ctrl     *gomock.Controller
	recorder *MockAPIMockRecorder
}

// MockAPIMockRecorder is the mock recorder for MockAPI.
type MockAPIMockRecorder struct {
	mock *MockAPI
}

// NewMockAPI creates a new mock instance.
func NewMockAPI(ctrl *gomock.Controller) *MockAPI {
	mock := &MockAPI{ctrl: ctrl}
	mock.recorder = &MockAPIMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAPI) EXPECT() *MockAPIMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockAPI) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, obj}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Create", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Create indicates an expected call of Create.
func (mr *MockAPIMockRecorder) Create(ctx, obj interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, obj}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockAPI)(nil).Create), varargs...)
}

// Monitor mocks base method.
func (m *MockAPI) Monitor(ctx context.Context, name, namespace string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Monitor", ctx, name, namespace)
	ret0, _ := ret[0].(error)
	return ret0
}

// Monitor indicates an expected call of Monitor.
func (mr *MockAPIMockRecorder) Monitor(ctx, name, namespace interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Monitor", reflect.TypeOf((*MockAPI)(nil).Monitor), ctx, name, namespace)
}
