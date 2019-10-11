package action

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

// Service is a action service client
// operates with actions via http
type Service struct {
	cli     *http.Client
	baseURL string
}

const defaultRequestTimeout = 15 * time.Second

func New(port int) (Service, error) {
	if !(1 <= port && port <= 65535) {
		return Service{}, errors.New("action: bad port")
	}

	s := Service{&http.Client{
		Timeout: defaultRequestTimeout,
	}, "http://127.0.0.1:" + strconv.Itoa(port)}

	return s, s.ping()
}

func (s Service) ping() error {
	resp, err := s.cli.Head(s.baseURL)
	if err != nil {
		return err
	}

	resp.Body.Close()

	return nil
}

func (s Service) Add(payload []byte) (body []byte, err error) {
	resp, err := s.cli.Post(
		s.baseURL+"/func", "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	body, err = checkResp(resp)
	resp.Body.Close()
	return
}

func (s Service) Delete(name string) (body []byte, err error) {
	req, err := http.NewRequest("DELETE", s.baseURL+"/func/"+name, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.cli.Do(req)
	if err != nil {
		return nil, err
	}

	body, err = checkResp(resp)
	resp.Body.Close()
	return
}

func (s Service) Call(name string, payload []byte) (body []byte, err error) {
	resp, err := s.cli.Post(s.baseURL+"/func/"+name, "application/json",
		bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	body, err = checkResp(resp)
	if err != nil {
		return
	}

	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	body = result

	resp.Body.Close()
	return
}

func checkResp(resp *http.Response) ([]byte, error) {
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		result, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		return result, errors.New("action: error response")
	}

	return nil, nil
}
