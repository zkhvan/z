package gh

import (
	"github.com/zkhvan/z/pkg/exec"
)

var defaultExecutor exec.Interface = exec.New()

type Client struct {
	executor exec.Interface
}

func NewClient() *Client {
	return &Client{executor: defaultExecutor}
}

func (c *Client) SetExecutor(executor exec.Interface) *Client {
	c.executor = executor
	return c
}
