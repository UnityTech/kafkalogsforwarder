package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMessageIsService(t *testing.T) {

	m := Message{}

	m.Data = []byte("{\"service\":\"myservice\",\"msg\":\"Hello, World!\\n\"}")

	value := m.IsService([]byte("myservice"))
	assert.Equal(t, value, true)

	value = m.IsService([]byte("otherservice"))
	assert.Equal(t, value, false)
}

func TestMessageIsServiceInvalidJSON(t *testing.T) {

	m := Message{}

	m.Data = []byte("{\"service\":\"myse")

	value := m.IsService([]byte("myservice"))
	assert.Equal(t, value, false)
}

func TestMessageParseJSON(t *testing.T) {

	m := Message{}

	m.Data = []byte("{\"service\":\"myservice\"}")

	m.ParseJSON()

	value, ok := m.Container.Path("service").Data().(string)
	assert.Equal(t, ok, true)
	assert.Equal(t, value, "myservice")
}

func TestMessageGetString(t *testing.T) {

	m := Message{}

	m.Data = []byte("{\"service\":\"myservice\"}")

	value, err := m.GetString("service")
	assert.Equal(t, err, nil)
	assert.Equal(t, value, "myservice")
}

func TestMessageGetString2(t *testing.T) {

	m := Message{}

	m.Data = []byte("{\"foo\":{\"bar\":\"test\"}}")

	value, err := m.GetString("foo", "bar")
	assert.Equal(t, err, nil)
	assert.Equal(t, value, "test")
}

func TestMessageGetString2WithENDL(t *testing.T) {

	m := Message{}

	m.Data = []byte("{\"foo\":{\"bar\":\"test\\n\"}}")

	value, err := m.GetString("foo", "bar")
	assert.Equal(t, err, nil)
	assert.Equal(t, value, "test\n")
}

func TestMessageGetStringWithENDL(t *testing.T) {

	m := Message{}

	m.Data = []byte("{\"msg\":\"myservice\\n\"}")

	value, err := m.GetString("msg")
	assert.Equal(t, err, nil)
	assert.Equal(t, value, "myservice\n")
}
