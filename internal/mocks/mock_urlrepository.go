// Code generated by MockGen. DO NOT EDIT.
// Source: C:\Users\admin\GolandProjects\URLShortener\internal\repositories\urlrepository.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	model "github.com/Totarae/URLShortener/internal/model"
	gomock "github.com/golang/mock/gomock"
)

// MockURLRepositoryInterface is a mock of URLRepositoryInterface interface.
type MockURLRepositoryInterface struct {
	ctrl     *gomock.Controller
	recorder *MockURLRepositoryInterfaceMockRecorder
}

// MockURLRepositoryInterfaceMockRecorder is the mock recorder for MockURLRepositoryInterface.
type MockURLRepositoryInterfaceMockRecorder struct {
	mock *MockURLRepositoryInterface
}

// NewMockURLRepositoryInterface creates a new mock instance.
func NewMockURLRepositoryInterface(ctrl *gomock.Controller) *MockURLRepositoryInterface {
	mock := &MockURLRepositoryInterface{ctrl: ctrl}
	mock.recorder = &MockURLRepositoryInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockURLRepositoryInterface) EXPECT() *MockURLRepositoryInterfaceMockRecorder {
	return m.recorder
}

// GetURL mocks base method.
func (m *MockURLRepositoryInterface) GetURL(ctx context.Context, shorten string) (*model.URLObject, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetURL", ctx, shorten)
	ret0, _ := ret[0].(*model.URLObject)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetURL indicates an expected call of GetURL.
func (mr *MockURLRepositoryInterfaceMockRecorder) GetURL(ctx, shorten interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetURL", reflect.TypeOf((*MockURLRepositoryInterface)(nil).GetURL), ctx, shorten)
}

// Ping mocks base method.
func (m *MockURLRepositoryInterface) Ping(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Ping", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// Ping indicates an expected call of Ping.
func (mr *MockURLRepositoryInterfaceMockRecorder) Ping(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Ping", reflect.TypeOf((*MockURLRepositoryInterface)(nil).Ping), ctx)
}

// SaveBatchURLs mocks base method.
func (m *MockURLRepositoryInterface) SaveBatchURLs(ctx context.Context, urlObjs []*model.URLObject) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveBatchURLs", ctx, urlObjs)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveBatchURLs indicates an expected call of SaveBatchURLs.
func (mr *MockURLRepositoryInterfaceMockRecorder) SaveBatchURLs(ctx, urlObjs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveBatchURLs", reflect.TypeOf((*MockURLRepositoryInterface)(nil).SaveBatchURLs), ctx, urlObjs)
}

// SaveURL mocks base method.
func (m *MockURLRepositoryInterface) SaveURL(ctx context.Context, urlObj *model.URLObject) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveURL", ctx, urlObj)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveURL indicates an expected call of SaveURL.
func (mr *MockURLRepositoryInterfaceMockRecorder) SaveURL(ctx, urlObj interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveURL", reflect.TypeOf((*MockURLRepositoryInterface)(nil).SaveURL), ctx, urlObj)
}
