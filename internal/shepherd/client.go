package shepherd

// MockSHEPHERD provides mock implementations for SHEPHERD client

type Client struct{}

func (c *Client) ValidateSecurity() bool {
	return true
}

func NewSHEPHERDClient() *Client {
	return &Client{}
}
