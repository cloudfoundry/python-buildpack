// Code generated by MockGen. DO NOT EDIT.
// Source: conda.go

// Package conda_test is a generated GoMock package.
package conda_test

import (
	gomock "github.com/golang/mock/gomock"
	io "io"
	reflect "reflect"
)

// MockStager is a mock of Stager interface
type MockStager struct {
	ctrl     *gomock.Controller
	recorder *MockStagerMockRecorder
}

// MockStagerMockRecorder is the mock recorder for MockStager
type MockStagerMockRecorder struct {
	mock *MockStager
}

// NewMockStager creates a new mock instance
func NewMockStager(ctrl *gomock.Controller) *MockStager {
	mock := &MockStager{ctrl: ctrl}
	mock.recorder = &MockStagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockStager) EXPECT() *MockStagerMockRecorder {
	return m.recorder
}

// BuildDir mocks base method
func (m *MockStager) BuildDir() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BuildDir")
	ret0, _ := ret[0].(string)
	return ret0
}

// BuildDir indicates an expected call of BuildDir
func (mr *MockStagerMockRecorder) BuildDir() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BuildDir", reflect.TypeOf((*MockStager)(nil).BuildDir))
}

// CacheDir mocks base method
func (m *MockStager) CacheDir() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CacheDir")
	ret0, _ := ret[0].(string)
	return ret0
}

// CacheDir indicates an expected call of CacheDir
func (mr *MockStagerMockRecorder) CacheDir() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CacheDir", reflect.TypeOf((*MockStager)(nil).CacheDir))
}

// DepDir mocks base method
func (m *MockStager) DepDir() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DepDir")
	ret0, _ := ret[0].(string)
	return ret0
}

// DepDir indicates an expected call of DepDir
func (mr *MockStagerMockRecorder) DepDir() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DepDir", reflect.TypeOf((*MockStager)(nil).DepDir))
}

// DepsIdx mocks base method
func (m *MockStager) DepsIdx() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DepsIdx")
	ret0, _ := ret[0].(string)
	return ret0
}

// DepsIdx indicates an expected call of DepsIdx
func (mr *MockStagerMockRecorder) DepsIdx() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DepsIdx", reflect.TypeOf((*MockStager)(nil).DepsIdx))
}

// LinkDirectoryInDepDir mocks base method
func (m *MockStager) LinkDirectoryInDepDir(arg0, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LinkDirectoryInDepDir", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// LinkDirectoryInDepDir indicates an expected call of LinkDirectoryInDepDir
func (mr *MockStagerMockRecorder) LinkDirectoryInDepDir(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LinkDirectoryInDepDir", reflect.TypeOf((*MockStager)(nil).LinkDirectoryInDepDir), arg0, arg1)
}

// WriteProfileD mocks base method
func (m *MockStager) WriteProfileD(arg0, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WriteProfileD", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// WriteProfileD indicates an expected call of WriteProfileD
func (mr *MockStagerMockRecorder) WriteProfileD(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WriteProfileD", reflect.TypeOf((*MockStager)(nil).WriteProfileD), arg0, arg1)
}

// MockInstaller is a mock of Installer interface
type MockInstaller struct {
	ctrl     *gomock.Controller
	recorder *MockInstallerMockRecorder
}

// MockInstallerMockRecorder is the mock recorder for MockInstaller
type MockInstallerMockRecorder struct {
	mock *MockInstaller
}

// NewMockInstaller creates a new mock instance
func NewMockInstaller(ctrl *gomock.Controller) *MockInstaller {
	mock := &MockInstaller{ctrl: ctrl}
	mock.recorder = &MockInstallerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockInstaller) EXPECT() *MockInstallerMockRecorder {
	return m.recorder
}

// InstallOnlyVersion mocks base method
func (m *MockInstaller) InstallOnlyVersion(arg0, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InstallOnlyVersion", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// InstallOnlyVersion indicates an expected call of InstallOnlyVersion
func (mr *MockInstallerMockRecorder) InstallOnlyVersion(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InstallOnlyVersion", reflect.TypeOf((*MockInstaller)(nil).InstallOnlyVersion), arg0, arg1)
}

// MockCommand is a mock of Command interface
type MockCommand struct {
	ctrl     *gomock.Controller
	recorder *MockCommandMockRecorder
}

// MockCommandMockRecorder is the mock recorder for MockCommand
type MockCommandMockRecorder struct {
	mock *MockCommand
}

// NewMockCommand creates a new mock instance
func NewMockCommand(ctrl *gomock.Controller) *MockCommand {
	mock := &MockCommand{ctrl: ctrl}
	mock.recorder = &MockCommandMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockCommand) EXPECT() *MockCommandMockRecorder {
	return m.recorder
}

// Execute mocks base method
func (m *MockCommand) Execute(arg0 string, arg1, arg2 io.Writer, arg3 string, arg4 ...string) error {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1, arg2, arg3}
	for _, a := range arg4 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Execute", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Execute indicates an expected call of Execute
func (mr *MockCommandMockRecorder) Execute(arg0, arg1, arg2, arg3 interface{}, arg4 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1, arg2, arg3}, arg4...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Execute", reflect.TypeOf((*MockCommand)(nil).Execute), varargs...)
}

// Output mocks base method
func (m *MockCommand) Output(dir, program string, args ...string) (string, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{dir, program}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Output", varargs...)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Output indicates an expected call of Output
func (mr *MockCommandMockRecorder) Output(dir, program interface{}, args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{dir, program}, args...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Output", reflect.TypeOf((*MockCommand)(nil).Output), varargs...)
}
