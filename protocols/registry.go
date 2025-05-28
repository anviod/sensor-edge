package protocols

import (
	"errors"
)

var registry = make(map[string]func() Protocol)

func Register(name string, constructor func() Protocol) {
	registry[name] = constructor
}

func Create(name string) (Protocol, error) {
	if ctor, ok := registry[name]; ok {
		return ctor(), nil
	}
	return nil, errors.New("unknown protocol: " + name)
}
