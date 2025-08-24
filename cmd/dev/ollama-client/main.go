package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/ollama/ollama/api"
)

const (
	uri   = "http://localhost:11434"
	model = "deepseek-r1:1.5b"
)

var defaultClient = &http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 15 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

func main() {
	ctx := context.Background()

	base, err := url.ParseRequestURI(uri)
	if err != nil {
		log.Fatal(err)
	}
	client := api.NewClient(base, defaultClient)

	if err := generate(ctx, client, model); err != nil {
		log.Fatal(err)
	}	
}

// generates text from the model using the provided prompt
func generate(ctx context.Context, client *api.Client, model string) error {
	req := &api.GenerateRequest{
		Model:  model,
		Prompt: "List some unusual animals",
	}

	respFunc := func(resp api.GenerateResponse) error {
		fmt.Printf("%+v", resp)
		return nil
	}

	return client.Generate(ctx, req, respFunc)
}
