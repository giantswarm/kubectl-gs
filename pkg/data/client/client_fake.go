package client

func NewFakeClient(config Config) (*Client, error) {
	k8sClient := NewFakeK8sClient()
	client := &Client{
		Logger:    config.Logger,
		K8sClient: k8sClient,
	}

	return client, nil
}
