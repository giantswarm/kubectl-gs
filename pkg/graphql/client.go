package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/giantswarm/microerror"
)

type ClientImplConfig struct {
	HttpClient *http.Client
	Url        string
}

type ClientImpl struct {
	httpClient *http.Client
	url        string
}

var _ Client = (*ClientImpl)(nil)

func NewClient(config ClientImplConfig) (*ClientImpl, error) {
	if config.HttpClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.HttpClient must not be empty", config)
	}
	if len(config.Url) < 1 {
		return nil, microerror.Maskf(invalidConfigError, "%T.Url must not be empty", config)
	}

	c := &ClientImpl{
		httpClient: config.HttpClient,
		url:        config.Url,
	}

	return c, nil
}

func (c *ClientImpl) ExecuteQuery(ctx context.Context, query string, variables map[string]string, v interface{}) error {
	var err error

	if len(query) < 1 {
		return microerror.Maskf(queryError, "empty query")
	}

	var req *http.Request
	{
		body := requestBody{
			Query:     query,
			Variables: variables,
		}
		buf := &bytes.Buffer{}
		err = json.NewEncoder(buf).Encode(body)
		if err != nil {
			return microerror.Mask(err)
		}

		req, err = http.NewRequest(http.MethodPost, c.url, buf)
		if err != nil {
			return microerror.Mask(err)
		}

		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return microerror.Mask(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return microerror.Mask(httpError)
	}

	var resBody responseBody
	{
		err = json.NewDecoder(res.Body).Decode(&resBody)
		if err != nil {
			return microerror.Mask(err)
		}

		if len(resBody.Errors) > 0 {
			return microerror.Mask(resBody.Errors)
		}

		if resBody.Data == nil {
			return microerror.Maskf(queryError, "empty response")
		}

		err = json.Unmarshal(*resBody.Data, &v)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
