package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// --- Consul HTTP API shapes (only the fields used) ---

type agentSelf struct {
	Config struct {
		Datacenter string `json:"Datacenter"`
	} `json:"Config"`
}

type agentMember struct {
	Name string `json:"Name"`
}

type catalogNode struct {
	Node    string `json:"Node"`
	Address string `json:"Address"`
	ID      string `json:"ID"`
}

type catalogNodeServices struct {
	Services map[string]struct {
		Service string `json:"Service"`
	} `json:"Services"`
}

// registerRequest is the body of PUT /v1/catalog/register. Service is omitted
// (nil) when registering a bare node.
type registerRequest struct {
	Datacenter string           `json:"Datacenter"`
	Node       string           `json:"Node"`
	Address    string           `json:"Address"`
	ID         string           `json:"ID"`
	Service    *registerService `json:"Service,omitempty"`
}

type registerService struct {
	Service string `json:"Service"`
	ID      string `json:"ID"`
	Port    int    `json:"Port"`
}

// --- HTTP client ---

type client struct {
	http *http.Client
}

// httpError carries the status code so callers can special-case 404.
type httpError struct {
	status int
	body   string
}

func (e *httpError) Error() string { return fmt.Sprintf("HTTP %d: %s", e.status, e.body) }

func (c *client) do(method, addr, path string, body []byte) ([]byte, error) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, "http://"+addr+path, r)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, &httpError{status: resp.StatusCode, body: strings.TrimSpace(string(data))}
	}
	return data, nil
}

func (c *client) getJSON(addr, path string, out any) error {
	data, err := c.do(http.MethodGet, addr, path, nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func statusOf(err error) int {
	var he *httpError
	if errors.As(err, &he) {
		return he.status
	}
	return 0
}
