package services

import (
	"context"
	"docker-monitoring-ui/internal/entities"
	"fmt"
	"os/exec"
	"strings"
)

type ContainerService struct {
	containers []entities.Container
	ctx        context.Context
}

func (c *ContainerService) Init(ctx context.Context) {
	c.startContainers(ctx)
}

func (c *ContainerService) startContainers(ctx context.Context) {
	// Get all containers
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--format", "{{.ID}}|{{.Image}}|{{.Names}}")
	cmdOut, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}

	// Get running container IDs
	runningMap, err := c.getRunningContainersMap(ctx)
	if err != nil {
		panic(err)
	}

	c.appendContainers(string(cmdOut), runningMap)
}

// Helper method
func (c *ContainerService) getRunningContainersMap(ctx context.Context) (map[string]bool, error) {
	cmd := exec.CommandContext(ctx, "docker", "ps", "--format", "{{.ID}}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	running := make(map[string]bool)
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, id := range lines {
		running[strings.TrimSpace(id)] = true
	}
	return running, nil
}

func (c *ContainerService) appendContainers(cmdOut string, runningMap map[string]bool) {
	c.containers = nil

	lines := strings.Split(strings.TrimSpace(cmdOut), "\n")
	for _, line := range lines {
		parts := strings.Split(line, "|")
		if len(parts) != 3 {
			continue
		}

		id := parts[0]
		image := parts[1]
		name := parts[2]

		container := entities.Container{
			ID:      id,
			Image:   image,
			Name:    name,
			Running: runningMap[id],
		}

		c.containers = append(c.containers, container)
	}
}

// Helper method
func (c *ContainerService) checkIfRunning(ctx context.Context, id string) bool {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Running}}", id)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

func (c *ContainerService) listContainers(ctx context.Context) {
	for _, container := range c.containers {
		status := "stopped"
		if container.Running {
			status = "running"
		}
		fmt.Printf("Name: %s | ID: %s | Image: %s | Status: %s\n", container.Name, container.ID, container.Image, status)
	}
}

func (c *ContainerService) getAllRunningContainers() ([]entities.Container, error) {
	var out []entities.Container
	for _, container := range c.containers {
		if container.Running {
			out = append(out, container)
		}
	}
	return out, nil
}

func (c *ContainerService) GetAllContainers() []entities.Container {
	return c.containers
}

func (c *ContainerService) StartContainer(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, "docker", "start", id)
	return cmd.Run()
}

func (c *ContainerService) StopContainer(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, "docker", "stop", id)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop container %s: %v\nOutput: %s", id, err, string(out))
	}
	return nil
}

func (c *ContainerService) RemoveContainer(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, "docker", "rm", id)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error removing container: %s\n%s", err, out)
	}
	return err
}

func (c *ContainerService) CreateContainer(ctx context.Context, name string, image string) error {
	if name == "" {
		return fmt.Errorf("container name is required")
	}
	if image == "" {
		return fmt.Errorf("image name is required")
	}

	cmd := exec.CommandContext(ctx, "docker", "run", "-d", "--name", name, image)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create container: %v\nOutput: %s", err, string(out))
	}
	return nil
}

func (c *ContainerService) InspectContainer(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Running}}", id)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to inspect container %s: %v\nOutput: %s", id, err, string(out))
	}

	return nil
}
