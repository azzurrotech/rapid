package pod

// MockPOD provides mock implementations for POD client

type Client struct{}

type Project struct {
	Name      string
	Language  string
	Status    string
	Size      int64
	Complexity float64
	HasTests  bool
	HasCI     bool
}

func (c *Client) GetProjects() ([]Project, error) {
	return []Project{
		{
			Name:      "Go Project",
			Language:  "Go",
			Status:    "active",
			Size:      1024,
			Complexity: 2.5,
			HasTests:  true,
			HasCI:     true,
		},
		{
			Name:      "Docker Project",
			Language:  "JavaScript",
			Status:    "active",
			Size:      2048,
			Complexity: 3.2,
			HasTests:  false,
			HasCI:     true,
		},
	}, nil
}

func NewPODClient() *Client {
	return &Client{}
}
