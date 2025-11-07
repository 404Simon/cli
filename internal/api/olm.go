package api

import "fmt"

// CreateOlm creates a new OLM with the specified name
func (c *Client) CreateOlm(name string, userID string) (*CreateOlmResponse, error) {
	var response CreateOlmResponse
	request := CreateOlmRequest{
		Name: name,
	}
	err := c.Put(fmt.Sprintf("/user/%s/olm", userID), request, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}
