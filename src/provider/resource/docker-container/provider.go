package docker

import (
	"dacrane/utils"
	"fmt"
	"strings"
)

type DockerContainerProvider struct{}

func (DockerContainerProvider) Create(parameter any) (any, error) {
	params := parameter.(map[string]any)
	image := params["image"].(string)
	name := params["name"].(string)
	env := params["env"].([]any)
	port := params["port"].(string)
	tag := params["tag"].(string)

	envOpts := []string{}
	for _, e := range env {
		name := e.(map[string]any)["name"].(string)
		value := e.(map[string]any)["value"].(string)
		opt := fmt.Sprintf(`-e "%s=%s"`, name, value)
		envOpts = append(envOpts, opt)
	}

	cmd := fmt.Sprintf("docker run -d --name %s -p %s %s %s:%s", name, port, strings.Join(envOpts, " "), image, tag)

	_, err := utils.RunOnBash(cmd)
	if err != nil {
		panic(err)
	}

	return parameter, nil
}

func (provider DockerContainerProvider) Update(current any, previous any) (any, error) {
	err := provider.Delete(previous)
	if err != nil {
		return nil, err
	}
	return provider.Create(current)
}

func (DockerContainerProvider) Delete(parameter any) error {
	params := parameter.(map[string]any)
	name := params["name"].(string)
	_, err := utils.RunOnBash(fmt.Sprintf("docker stop %s", name))
	if err != nil {
		panic(err)
	}
	_, err = utils.RunOnBash(fmt.Sprintf("docker rm %s", name))
	if err != nil {
		panic(err)
	}
	return nil
}
