package safehttp

import (
	"errors"
	"net/http"
	"sync"
)

type Client struct {
	mu          *sync.Mutex
	client      *http.Client
	isAvailable bool
}

func NewClient(mu *sync.Mutex) *Client {
	return &Client{
		mu:          mu,
		client:      &http.Client{},
		isAvailable: true,
	}
}

func (c *Client) Lock() error {
	defer c.mu.Unlock()
	c.mu.Lock()

	if !c.isAvailable {
		return errors.New("client is already locked")
	}

	c.isAvailable = false

	return nil
}

func (c *Client) Unlock() {
	defer c.mu.Unlock()
	c.mu.Lock()

	if !c.isAvailable {
		c.isAvailable = true
	}
}

func (c *Client) Do(request *Req) (*Resp, error) {
	defer c.mu.Unlock()
	c.mu.Lock()

	if !c.isAvailable {
		return nil, errors.New("client is already locked")
	}

	c.isAvailable = false
	defer func() {
		c.isAvailable = true
	}()

	if err := request.Timestamp.SetFrom(); err != nil {
		return nil, errors.New("httpclient: " + err.Error())
	}

	resp, err := c.client.Do(request.Request)
	if err != nil {
		request.MarkFailed()
		if errSec := request.Timestamp.SetTo(); errSec != nil {
			return nil, errors.New(err.Error() + ". " + errSec.Error())
		}
		return nil, err
	}

	if err = request.Timestamp.SetTo(); err != nil {
		return nil, err
	}

	if err = request.Timestamp.SetDuration(); err != nil {
		return nil, err
	}

	return NewResp(resp)
}