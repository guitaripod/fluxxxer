package flux

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
)

type Client struct {
	apiURL string
}

func NewClient() *Client {
	return &Client{
		apiURL: os.Getenv("FLUX_API_URL"),
	}
}

func (c *Client) GenerateImages(prompt string) ([]string, error) {
	input := Input{
		Prompt:             prompt,
		NumOutputs:         4,
		AspectRatio:        "1:1",
		OutputFormat:       "png",
		OutputQuality:      1,
		DisableSafetyCheck: true,
	}

	payload := map[string]interface{}{"input": input}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(
		c.apiURL,
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var urls []string
	if err := json.NewDecoder(resp.Body).Decode(&urls); err != nil {
		return nil, err
	}

	return urls, nil
}
