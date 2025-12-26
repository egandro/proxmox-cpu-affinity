package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newHelloWorldCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "helloworld",
		Short: "Print a hello world message in JSON",
		Run: func(cmd *cobra.Command, args []string) {
			type Response struct {
				Message string `json:"message"`
			}

			resp := Response{
				Message: "Hello World from Golang!",
			}

			// Output JSON to stdout
			jsonData, err := json.Marshal(resp)
			if err != nil {
				fmt.Println("Error generating JSON")
				os.Exit(1)
			}

			fmt.Println(string(jsonData))
		},
	}
}
