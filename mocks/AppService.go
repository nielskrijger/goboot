// Code generated by mockery 2.9.0. DO NOT EDIT.

package mocks

import (
	"github.com/nielskrijger/goboot"
	mock "github.com/stretchr/testify/mock"
)

// AppService is an autogenerated mock type for the AppService type
type AppService struct {
	mock.Mock
}

// Close provides a mock function with given fields:
func (_m *AppService) Close() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Configure provides a mock function with given fields: ctx
func (_m *AppService) Configure(env *goboot.AppEnv) error {
	ret := _m.Called(env)

	var r0 error
	if rf, ok := ret.Get(0).(func(*goboot.AppEnv) error); ok {
		r0 = rf(env)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Init provides a mock function with given fields:
func (_m *AppService) Init() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Name provides a mock function with given fields:
func (_m *AppService) Name() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}
