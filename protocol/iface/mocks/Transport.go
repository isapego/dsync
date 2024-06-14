// Code generated by mockery v2.41.0. DO NOT EDIT.

package mocks

import (
	iface "github.com/adiom-data/dsync/protocol/iface"
	mock "github.com/stretchr/testify/mock"
)

// Transport is an autogenerated mock type for the Transport type
type Transport struct {
	mock.Mock
}

// CloseDataChannel provides a mock function with given fields: _a0
func (_m *Transport) CloseDataChannel(_a0 iface.DataChannelID) {
	_m.Called(_a0)
}

// CreateDataChannel provides a mock function with given fields:
func (_m *Transport) CreateDataChannel() (iface.DataChannelID, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for CreateDataChannel")
	}

	var r0 iface.DataChannelID
	var r1 error
	if rf, ok := ret.Get(0).(func() (iface.DataChannelID, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() iface.DataChannelID); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(iface.DataChannelID)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCoordinatorEndpoint provides a mock function with given fields: location
func (_m *Transport) GetCoordinatorEndpoint(location string) (iface.CoordinatorIConnectorSignal, error) {
	ret := _m.Called(location)

	if len(ret) == 0 {
		panic("no return value specified for GetCoordinatorEndpoint")
	}

	var r0 iface.CoordinatorIConnectorSignal
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (iface.CoordinatorIConnectorSignal, error)); ok {
		return rf(location)
	}
	if rf, ok := ret.Get(0).(func(string) iface.CoordinatorIConnectorSignal); ok {
		r0 = rf(location)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(iface.CoordinatorIConnectorSignal)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(location)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetDataChannelEndpoint provides a mock function with given fields: _a0
func (_m *Transport) GetDataChannelEndpoint(_a0 iface.DataChannelID) (chan iface.DataMessage, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for GetDataChannelEndpoint")
	}

	var r0 chan iface.DataMessage
	var r1 error
	if rf, ok := ret.Get(0).(func(iface.DataChannelID) (chan iface.DataMessage, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(iface.DataChannelID) chan iface.DataMessage); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(chan iface.DataMessage)
		}
	}

	if rf, ok := ret.Get(1).(func(iface.DataChannelID) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewTransport creates a new instance of Transport. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewTransport(t interface {
	mock.TestingT
	Cleanup(func())
}) *Transport {
	mock := &Transport{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
