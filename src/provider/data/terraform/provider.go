package terraform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

type TerraformDataProvider struct{}

type ProviderConfig struct {
	User     string `hcl:"user"`
	Password string `hcl:"password"`
}

type DataConfig struct {
	A int `hcl:"a"`
	B int `hcl:"b"`
}

var ctx = context.Background()

func (p TerraformDataProvider) Get(parameters map[string]any) (map[string]any, error) {
	f := hclwrite.NewEmptyFile()
	rootBody := f.Body()

	// Setting Provider
	if provider, ok := parameters["provider"].(string); ok {
		providerBlock := rootBody.AppendNewBlock("provider", []string{provider})
		providerBody := providerBlock.Body()
		if configs, ok := parameters["configurations"].(map[string]interface{}); ok {
			for k, v := range configs {
				writeHCL(providerBody, k, v)
			}
		}
	} else {
		return nil, fmt.Errorf("provider name is required and must be a string")
	}

	// Setting Resource
	resourceType, resourceName := "", ""
	if resType, ok := parameters["resource"].(string); ok {
		resourceType = resType
	} else {
		return nil, fmt.Errorf("resource type is required and must be a string")
	}

	if resName, ok := parameters["name"].(string); ok {
		resourceName = resName
	} else {
		return nil, fmt.Errorf("resource name is required and must be a string")
	}

	resourceBlock := rootBody.AppendNewBlock("data", []string{resourceType, resourceName})
	resourceBody := resourceBlock.Body()
	if args, ok := parameters["argument"].(map[string]interface{}); ok {
		for k, v := range args {
			writeHCL(resourceBody, k, v)
		}
	}

	// write file
	instanceName := "your_instance_name"
	localModuleName := "your_module_name"
	filename := "your_filename.tf"
	dir := filepath.Join(".dacrane", "instances", instanceName, "custom_states", localModuleName)
	filePath := filepath.Join(dir, filename)

	// Ensure the directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}

	// Write the file
	if err := os.WriteFile(filePath, f.Bytes(), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("HCL written to %s\n", filePath)

	// Terraform exec
	if err := p.ApplyTerraform(filePath); err != nil {
		return nil, fmt.Errorf("failed to apply terraform: %w", err)
	}

	return nil, nil
}

func writeHCL(body *hclwrite.Body, key string, value interface{}) {
	switch v := value.(type) {
	case map[string]interface{}:
		block := body.AppendNewBlock(key, nil)
		blockBody := block.Body()
		for k, val := range v {
			writeHCL(blockBody, k, val)
		}
	case string:
		body.SetAttributeValue(key, cty.StringVal(v))
	case []interface{}:
		values := make([]cty.Value, len(v))
		for i, val := range v {
			values[i] = cty.StringVal(val.(string))
		}
		body.SetAttributeValue(key, cty.ListVal(values))
	default:
		fmt.Printf("Unsupported type: %T\n", v)
	}
}

func (TerraformDataProvider) ApplyTerraform(filePath string) error {
	// Terraform init
	dir := filepath.Dir(filePath)
	
	initCmd := exec.Command("terraform", "init")
	initCmd.Dir = dir
	if output, err := initCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to run terraform init: %s, %s", err, output)
	}

	// Terraform apply
	applyCmd := exec.Command("terraform", "apply", "-auto-approve")
	applyCmd.Dir = dir
	if output, err := applyCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to run terraform apply: %s, %s", err, output)
	}

	fmt.Println("Terraform apply complete")
	return nil
}
