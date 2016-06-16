package main

import "github.com/Jeffail/gabs"
import "github.com/buger/jsonparser"
import "bytes"
import "fmt"
import "strings"

type Message struct {
	Key       string
	Topic     string
	Partition int32
	Offset    int64
	Data      []byte
	Container *gabs.Container
}

func NewMessageFromJSON(topic string, msg []byte) Message {

	m := Message{}
	m.Topic = topic
	m.Data = msg

	return m
}

func (m *Message) ParseJSON() error {

	if m.Container != nil {
		return nil
	}

	parsedJson, err := gabs.ParseJSON(m.Data)
	if err != nil {
		return err
	}

	m.Container = parsedJson

	return nil
}

func (m *Message) GetString(keys ...string) (string, error) {

	v, t, _, e := jsonparser.Get(m.Data, keys...)

	if e != nil {
		return "", e
	}

	if t != jsonparser.String {
		return "", fmt.Errorf("Value is not a number: %s", string(v))
	}

	// If no escapes return raw conten
	if bytes.IndexByte(v, '\\') == -1 {
		return string(v), nil
	}

	m.ParseJSON()
	value, _ := m.Container.Path(strings.Join(keys, ".")).Data().(string)
	return value, nil
}

func (m *Message) IsService(service []byte) bool {

	value, _, _, err := jsonparser.Get(m.Data, "service")
	if err != nil {
		return false
	}
	if bytes.Equal(value, service) == true {
		return true
	}

	return false
}
