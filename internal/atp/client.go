package atp

// MockATP provides mock implementations for ATP interface
import "fmt"

type Client struct{}

func (c *Client) HealthCheck() error {
	return nil
}

func (c *Client) GetPlatformVersion() string {
	return "1.0.0"
}

func (c *Client) SecretVault() interface{} {
	return &SecretVault{}
}

type SecretVault struct{}

func (s *SecretVault) GetSecret(namespace, name string) ([]byte, error) {
	return []byte("mock-secret-value"), nil
}

func (s *SecretVault) SetSecret(namespace, name string, value []byte) error {
	return nil
}

type Secret struct {
	Namespace string
	Name      string
	Value     []byte
}

func (s *Secret) GetToken() string {
	return fmt.Sprintf("token-%s-%s", s.Namespace, s.Name)
}

func NewATPClient() *Client {
	return &Client{}
}
