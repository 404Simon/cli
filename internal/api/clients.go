package api

type ClientResource struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type CreateClientRequest struct {
	Name string `json:"name"`
}

type CreateClientResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type ListClientsResponse struct {
	Clients []ClientResource `json:"clients"`
}

func GetClient(client *Client, clientID string) (*ClientResource, error) {
	var clientResource ClientResource
	err := client.Get("/api/clients/"+clientID, &clientResource)
	if err != nil {
		return nil, err
	}
	return &clientResource, nil
}

func ListClients(client *Client) ([]ClientResource, error) {
	var response ListClientsResponse
	err := client.Get("/api/clients", &response)
	if err != nil {
		return nil, err
	}
	return response.Clients, nil
}

func CreateClient(client *Client, req CreateClientRequest) (*CreateClientResponse, error) {
	var response CreateClientResponse
	err := client.Post("/api/clients", req, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func UpdateClient(client *Client, clientID string, req CreateClientRequest) (*ClientResource, error) {
	var clientResource ClientResource
	err := client.Put("/api/clients/"+clientID, req, &clientResource)
	if err != nil {
		return nil, err
	}
	return &clientResource, nil
}

func DeleteClient(client *Client, clientID string) error {
	return client.Delete("/api/clients/"+clientID, nil)
}
