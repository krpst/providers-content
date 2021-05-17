package main

// ClientMock is a mock for Client interface.
type ClientMock struct {
	GetContentFn func(userIP string, count int) ([]*ContentItem, error)
}

// GetContent is a mock for GetContent function.
func (c *ClientMock) GetContent(userIP string, count int) ([]*ContentItem, error) {
	return c.GetContentFn(userIP, count)
}
