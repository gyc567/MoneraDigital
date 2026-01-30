//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func main() {
	data, err := ioutil.ReadFile("internal/docs/swagger_template.go")
	if err != nil {
		fmt.Println("Error reading file:", err)
		os.Exit(1)
	}

	content := string(data)
	start := strings.Index(content, "const swaggerTemplate = `{")
	if start == -1 {
		fmt.Println("Could not find template start")
		os.Exit(1)
	}
	// Find the closing backtick
	end := strings.Index(content[start+len("const swaggerTemplate = `"):], "`}")
	if end == -1 {
		fmt.Println("Could not find template end")
		os.Exit(1)
	}

	// Adjust end to be relative to the whole string
	end = start + len("const swaggerTemplate = `") + end

	jsonStr := content[start+len("const swaggerTemplate = `") : end]

	var jsonMap map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &jsonMap); err != nil {
		fmt.Println("Invalid JSON:", err)
		os.Exit(1)
	}
	fmt.Println("JSON is valid!")
}
