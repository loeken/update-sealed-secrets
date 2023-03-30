package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	// Define command-line flags
	pflag.String("context", "", "Kubernetes context to use")
	pflag.String("namespace", "", "Namespace of the secret")
	pflag.String("secret", "", "Name of the secret")
	pflag.Parse()

	// Bind flags to Viper configuration
	viper.BindPFlags(pflag.CommandLine)

	// Retrieve values from Viper configuration
	context := viper.GetString("context")
	namespace := viper.GetString("namespace")
	secret := viper.GetString("secret")
	outputFile := fmt.Sprintf("updated_secret_%s_%s.yaml", namespace, secret)

	// Execute kubectl command to fetch the secret data
	cmd := exec.Command("kubectl", "get", "secret", secret, "-n", namespace, "-o", "yaml", "--context", context)
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error executing kubectl command: %s\n", err.Error())
		os.Exit(1)
	}

	// Use yq to extract data section
	yqCmd := exec.Command("yq", ".data", "-j")
	yqCmd.Stdin = ioutil.NopCloser(strings.NewReader(string(out)))
	yqOut, err := yqCmd.Output()
	if err != nil {
		fmt.Printf("Error executing yq command: %s\n", err.Error())
		os.Exit(1)
	}

	// Parse JSON output to get data values
	var data map[string]string
	err = json.Unmarshal(yqOut, &data)
	if err != nil {
		fmt.Printf("Error decoding yq output: %s\n", err.Error())
		os.Exit(1)
	}

	// Update data values
	for key, value := range data {
		decoded, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			fmt.Printf("Error decoding data value for key %s: %s\n", key, err.Error())
			continue
		}
		data[key] = (string(decoded))
	}
	// Generate YAML output with updated data section
	yamlOut := mapToYaml(data)

	// Write YAML output to file
	err = ioutil.WriteFile(outputFile, []byte(yamlOut), 0644)
	if err != nil {
		fmt.Printf("Error writing output file: %s\n", err.Error())
		os.Exit(1)
	}
}

func mapToYaml(m map[string]string) string {
	yaml := "apiVersion: v1\n"
	yaml += "kind: Secret\n"
	yaml += "metadata:\n"
	yaml += "  name: " + viper.GetString("secret") + "\n"
	yaml += "  namespace: " + viper.GetString("namespace") + "\n"
	yaml += "type: Opaque\n"
	yaml += "stringData:\n"
	for key, value := range m {
		yaml += fmt.Sprintf("  %s: %s\n", key, value)
	}
	return yaml
}
