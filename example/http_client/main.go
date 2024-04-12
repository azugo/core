package main

import (
	"fmt"

	"azugo.io/core/http"
)

func main() {
	cfg := &http.Configuration{
		Clients: map[string]http.NamedClient{
			"example": {
				BaseURI: "https://example.com",
			},
		},
	}

	c, err := http.NewClient(cfg).WithConfiguration("example")
	if err != nil {
		panic(err)
	}
	resp, err := c.Get("/")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(resp))
}
