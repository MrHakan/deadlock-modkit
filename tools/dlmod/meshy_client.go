//go:build !nomeshy

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// A small client for Meshy's REST API (https://docs.meshy.ai), used by the studio's character
// creator. Requests go through the system's curl, as dlmod's other downloads do. The API key is
// the user's own: it is kept in game/dlmod/meshy/key (or read from MESHY_API_KEY) and only ever
// sent to Meshy, in a header file curl reads, never on a command line.

var meshyBase = "https://api.meshy.ai" // tests point it at a local server

type meshyClient struct {
	base, key string
	tmp       string // folder for the request files curl reads
}

// meshyTask is a Meshy task as its GET endpoint returns it, for every task type dlmod uses.
type meshyTask struct {
	ID           string            `json:"id"`
	Status       string            `json:"status"` // PENDING, IN_PROGRESS, SUCCEEDED, FAILED, CANCELED
	Progress     int               `json:"progress"`
	Preceding    int               `json:"preceding_tasks"`
	ImageURLs    []string          `json:"image_urls"`
	ModelURLs    map[string]string `json:"model_urls"`
	ThumbnailURL string            `json:"thumbnail_url"`
	Credits      int               `json:"consumed_credits"`
	TaskError    *struct {
		Message string `json:"message"`
	} `json:"task_error"`
	Result *struct {
		RiggedGLB string `json:"rigged_character_glb_url"`
	} `json:"result"`
}

func (t meshyTask) finished() bool {
	return t.Status == "SUCCEEDED" || t.Status == "FAILED" || t.Status == "CANCELED"
}

func (t meshyTask) failure() string {
	if t.TaskError != nil && t.TaskError.Message != "" {
		return t.TaskError.Message
	}
	return "Meshy reports the task as " + strings.ToLower(t.Status)
}

func meshyKeyFile(game Game) string { return filepath.Join(game.StateDir(), "meshy", "key") }

// loadMeshyKey finds the user's Meshy API key: MESHY_API_KEY, or the one saved from the studio.
func loadMeshyKey(game Game) string {
	if key := strings.TrimSpace(os.Getenv("MESHY_API_KEY")); key != "" {
		return key
	}
	data, err := os.ReadFile(meshyKeyFile(game))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func saveMeshyKey(game Game, key string) error {
	if err := os.MkdirAll(filepath.Dir(meshyKeyFile(game)), 0o700); err != nil {
		return err
	}
	return os.WriteFile(meshyKeyFile(game), []byte(strings.TrimSpace(key)+"\n"), 0o600)
}

func newMeshyClient(game Game, key string) *meshyClient {
	base := meshyBase
	if dev := os.Getenv("DLMOD_MESHY_API"); dev != "" { // a stand-in for Meshy on this computer, to try the creator without spending credits
		base = strings.TrimRight(dev, "/")
	}
	return &meshyClient{base: base, key: key, tmp: filepath.Join(game.StateDir(), "meshy")}
}

// call sends one request and returns the response body.
func (c *meshyClient) call(method, path string, body any) ([]byte, error) {
	if c.key == "" {
		return nil, errors.New("no Meshy API key: add yours in the Create tab (meshy.ai → Settings → API)")
	}
	address := c.base + path
	if err := allowedURL(address); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(c.tmp, 0o700); err != nil {
		return nil, err
	}
	header, err := os.CreateTemp(c.tmp, "request-*.txt")
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(header, "Authorization: Bearer %s\n", c.key)
	header.Close()
	defer os.Remove(header.Name())
	args := []string{"--silent", "--show-error", "--fail-with-body", "--max-time", "180", "--user-agent", "dlmod/" + version,
		"--header", "@" + header.Name(), "--request", method}
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		file, err := os.CreateTemp(c.tmp, "body-*.json")
		if err != nil {
			return nil, err
		}
		file.Write(data)
		file.Close()
		defer os.Remove(file.Name())
		args = append(args, "--header", "Content-Type: application/json", "--data-binary", "@"+file.Name())
	}
	command := exec.Command("curl", append(args, address)...)
	var out, problems bytes.Buffer
	command.Stdout, command.Stderr = &out, &problems
	if err := command.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, errors.New("curl was not found; it comes with Windows 10 and later")
		}
		var reply struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(out.Bytes(), &reply) == nil && reply.Message != "" {
			return nil, fmt.Errorf("Meshy: %s", reply.Message)
		}
		return nil, fmt.Errorf("Meshy did not answer: %s", strings.TrimSpace(problems.String()))
	}
	return out.Bytes(), nil
}

// create starts a task of a kind (text-to-image, image-to-image, image-to-3d,
// multi-image-to-3d, rigging) and returns its ID.
func (c *meshyClient) create(kind string, body map[string]any) (string, error) {
	data, err := c.call("POST", "/openapi/v1/"+kind, body)
	if err != nil {
		return "", err
	}
	var reply struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(data, &reply); err != nil || reply.Result == "" {
		return "", fmt.Errorf("Meshy sent no task ID: %s", strings.TrimSpace(string(data)))
	}
	return reply.Result, nil
}

func (c *meshyClient) task(kind, id string) (meshyTask, error) {
	var t meshyTask
	data, err := c.call("GET", "/openapi/v1/"+kind+"/"+id, nil)
	if err != nil {
		return t, err
	}
	err = json.Unmarshal(data, &t)
	return t, err
}

// balance is the account's credits.
func (c *meshyClient) balance() (int, error) {
	data, err := c.call("GET", "/openapi/v1/balance", nil)
	if err != nil {
		return 0, err
	}
	var reply struct {
		Balance int `json:"balance"`
	}
	err = json.Unmarshal(data, &reply)
	return reply.Balance, err
}

// meshyPrice is what a task costs in credits, by Meshy's price list (docs.meshy.ai/en/api/pricing,
// as of 2026-10): image models by name, 3D by geometry model (textured), rigging flat.
func meshyPrice(kind, model string) int {
	switch kind {
	case "text-to-image", "image-to-image":
		switch model {
		case "nano-banana", "nano-banana-2-lite":
			return 3
		case "nano-banana-2-1":
			return 4
		case "nano-banana-2":
			return 6
		case "nano-banana-pro":
			return 9
		}
		if strings.HasPrefix(model, "gpt-image") {
			if kind == "image-to-image" {
				return 12
			}
			return 9
		}
	case "image-to-3d", "multi-image-to-3d":
		switch model {
		case "meshy-t2", "meshy-t2.5", "meshy-6-lite":
			return 15
		default:
			return 30
		}
	case "rigging":
		return 5
	}
	return 0
}
