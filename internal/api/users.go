package api

// GetUser retrieves the current user information
func (c *Client) GetUser() (*User, error) {
	var user User
	err := c.Get("/user", &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// ListUserOrgs retrieves all organizations for a user
func (c *Client) ListUserOrgs(userID string) (*ListUserOrgsResponse, error) {
	var response ListUserOrgsResponse
	endpoint := "/user/" + userID + "/orgs"
	err := c.Get(endpoint, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}
