package server

import (
	"context"
	"errors"
	"os"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/jmorganca/ollama/api"
	"github.com/jmorganca/ollama/llm"
)

// TODO - this would ideally be in the llm package, but that would require some refactoring of interfaces in the server
//        package to avoid circular dependencies

func TestIntegrationOrcaMini(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get pwd failed: %s", err)
	}

	req := api.GenerateRequest{
		Model:   "orca-mini",
		Prompt:  "hello world",
		Options: map[string]interface{}{},
	}

	// TODO - add logic to check for existence of test data else skip this test
	_, filename, _, _ := runtime.Caller(0)
	modelDir := path.Dir(path.Dir(filename) + "/../test_data/models/.")
	if _, err := os.Stat(modelDir); errors.Is(err, os.ErrNotExist) {
		t.Skipf("%s does not exist - skipping integration tests", modelDir)
	}
	os.Setenv("OLLAMA_MODELS", modelDir)
	model, err := GetModel(req.Model)
	if err != nil {
		t.Fatalf("GetModel failed: %s", err)
	}
	opts := api.DefaultOptions()
	llmRunner, err := llm.New("unused", model.ModelPath, model.AdapterPaths, opts)
	if err != nil {
		t.Fatalf("llm.New failed (%s): %s", pwd, err)
	}
	prompt, err := model.Prompt(req)
	if err != nil {
		t.Fatalf("prompt generation failed: %s", err)
	}
	success := make(chan bool, 1)
	response := ""
	cb := func(resp api.GenerateResponse) {
		response += resp.Response
		if resp.Done {
			success <- true
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	err = llmRunner.Predict(ctx, []int{}, prompt, "", cb)
	if err != nil {
		t.Fatalf("predict call failed: %s", err)
	}

	select {
	case <-ctx.Done():
		t.Fatalf("failed to complete before timeout: \n%s", response)
	case <-success:
		// TODO - what sort of additional checks should we put here?
		t.Logf("Completed:\n%s", response)
	}
}
