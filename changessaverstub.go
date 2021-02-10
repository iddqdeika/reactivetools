package reactivetools

import (
	"fmt"
)

func NewStub() ChangesProcessor {
	return stubSaver{}
}

type stubSaver struct {
}

func (s stubSaver) Process(event ChangeEvent) error {
	fmt.Printf("saving new value (%v) for entity(%v): %v\r\n", event.Data(), event.ObjectType(), event.ObjectIdentifier())
	return nil
}
