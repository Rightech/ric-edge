package action

import "errors"

// Noop no operation action service
// used when action disabled
type Noop struct{}

func (Noop) Add([]byte) ([]byte, error) {
	return nil, errors.New("action: disabled")
}

func (Noop) Delete(string) ([]byte, error) {
	return nil, errors.New("action: disabled")
}

func (Noop) Call(string, []byte) ([]byte, error) {
	return nil, errors.New("action: disabled")
}
