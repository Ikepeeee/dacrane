package module

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/SIOS-Technology-Inc/dacrane/v0/src/core/repository"
)

type Plugin struct {
	Name  string
	Apply func(
		instanceAddress string,
		argument any,
		instances *repository.DocumentRepository,
	)
	Destroy func(
		instanceAddress string,
		instances *repository.DocumentRepository,
	)
}

func IsPluginPathString(module string) bool {
	keys := strings.Split(module, "/")
	return len(keys) == 3 || len(keys) == 4
}

func NewPlugin(module string) Plugin {
	keys := strings.Split(module, "/")
	var kind string
	if len(keys) == 3 {
		kind = keys[1]
	} else if len(keys) == 4 {
		kind = keys[2]
	} else {
		panic("module name should be {container_image}/{resource|data}/{name}")
	}

	switch kind {
	case "resource":
		return NewResourcePlugin(module)
	case "data":
		return NewDataPlugin(module)
	default:
		panic("module kind should be resource or data")
	}
}

func NewResourcePlugin(module string) Plugin {
	keys := strings.Split(module, "/")
	var image, kind, name string
	if len(keys) == 3 {
		image = "dacrane/" + keys[0]
		kind = keys[1]
		name = keys[2]
	} else if len(keys) == 4 {
		image = keys[0] + "/" + keys[1]
		kind = keys[2]
		name = keys[3]
	} else {
		panic("module name should be {container_image}/{resource|data}/{name}")
	}

	return Plugin{
		Name: module,
		Apply: func(
			instanceAddress string,
			argument any,
			instances *repository.DocumentRepository,
		) {
			if instances.Exists(instanceAddress) {
				fmt.Printf("[%s (%s)] Updating...\n", instanceAddress, module)

				document := instances.Find(instanceAddress)
				instance := NewInstanceFromDocument(document).(pluginInstance)

				arguments := []any{argument, instance.ToState(*instances)}
				input := buildPluginInput(kind, name, "update", instance.CustomStateDir, arguments)
				output, err := runPlugin(image, input)
				if err != nil {
					panic(err)
				}
				instance.Output = output
				instances.Upsert(instanceAddress, instance)
				fmt.Printf("[%s (%s)] Updated.\n", instanceAddress, module)
			} else {
				fmt.Printf("[%s (%s)] Creating...\n", instanceAddress, module)
				// TODO Specify from entry point
				arguments := []any{argument}
				customStateDir := fmt.Sprintf(".dacrane/custom_state/%s", instanceAddress)
				input := buildPluginInput(kind, name, "create", customStateDir, arguments)
				output, err := runPlugin(image, input)
				if err != nil {
					panic(err)
				}
				instance := NewPluginInstance(module, customStateDir, argument, output)
				instances.Upsert(instanceAddress, instance)
				fmt.Printf("[%s (%s)] Created.\n", instanceAddress, module)
			}
		},
		Destroy: func(instanceAddress string, instances *repository.DocumentRepository) {
			if !instances.Exists(instanceAddress) {
				fmt.Printf("[%s (%s)] Skipped. %s is not exist.\n",
					instanceAddress, module, instanceAddress)
			}
			fmt.Printf("[%s (%s)] Deleting...\n", instanceAddress, module)
			document := instances.Find(instanceAddress)
			instance := NewInstanceFromDocument(document).(pluginInstance)

			arguments := []any{instance.ToState(*instances)}
			input := buildPluginInput(kind, name, "delete", instance.CustomStateDir, arguments)
			_, err := runPlugin(image, input)
			if err != nil {
				panic(err)
			}
			instances.Delete(instanceAddress)
			fmt.Printf("[%s (%s)] Deleted.\n", instanceAddress, module)
		},
	}
}

func NewDataPlugin(module string) Plugin {
	keys := strings.Split(module, "/")
	if len(keys) != 3 {
		panic("module name should be {container_image}/{resource|data}/{name}")
	}
	image := keys[0]
	kind := keys[1]
	name := keys[2]
	return Plugin{
		Name: module,
		Apply: func(
			instanceAddress string,
			argument any,
			instances *repository.DocumentRepository,
		) {
			fmt.Printf("[%s (%s)] Reading...\n", instanceAddress, module)
			// TODO Specify from entry point
			arguments := []any{argument}
			customStateDir := fmt.Sprintf(".dacrane/custom_state/%s", instanceAddress)
			input := buildPluginInput(kind, name, "get", customStateDir, arguments)
			output, err := runPlugin(image, input)
			if err != nil {
				panic(err)
			}
			instance := NewPluginInstance(module, customStateDir, argument, output)
			instances.Upsert(instanceAddress, instance)
			fmt.Printf("[%s (%s)] Read.\n", instanceAddress, module)
		},
		Destroy: nil,
	}
}

func buildPluginInput(kind, name, operation, customStateDir string, arguments []any) map[string]any {
	return map[string]any{
		"kind":             kind,
		"name":             name,
		"operation":        operation,
		"custom_state_dir": customStateDir,
		"arguments":        arguments,
	}
}

func runPlugin(image string, input any) (any, error) {
	// preflight
	preflight := buildPluginInput("preflight", "", "", "", []any{})
	var setting map[string]any
	err := appPlugin(image, preflight, &setting, nil)
	if err != nil {
		return nil, err
	}
	// execute crud resource
	var output any
	err = appPlugin(image, input, &output, setting)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func appPlugin(image string, input any, output any, setting map[string]any) error {
	inputJson, err := json.Marshal(input)
	if err != nil {
		return err
	}
	script := `docker run --rm`
	if setting != nil && setting["working_dir"] != nil {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}
		script = fmt.Sprintf("%s -e HOST_WORKING_DIR=%s -v %s:%s", script, dir, dir, setting["working_dir"])
	}
	if setting != nil && setting["docker_host"] != nil {
		// TODO get docker host from environment
		script = fmt.Sprintf("%s -v /var/run/docker.sock:%s", script, setting["docker_host"])
	}
	script = fmt.Sprintf("%s %s", script, image)
	script = fmt.Sprintf("%s '%s'", script, base64.StdEncoding.EncodeToString(inputJson))
	// fmt.Printf("> %s\n", script)
	cmd := exec.Command("bash", "-c", script)
	writer := new(bytes.Buffer)
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}
	err = json.Unmarshal(writer.Bytes(), output)
	if err != nil {
		return err
	}
	return nil
}
