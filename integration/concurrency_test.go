//go:build integration

package integration

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/ollama/ollama/api"
)

func TestMultiModelConcurrency(t *testing.T) {
	var (
		req = [2]api.GenerateRequest{
			{
				Model:  "orca-mini",
				Prompt: "why is the ocean blue?",
				Stream: &stream,
				Options: map[string]interface{}{
					"seed":        42,
					"temperature": 0.0,
				},
			}, {
				Model:  "tinydolphin",
				Prompt: "what is the origin of the us thanksgiving holiday?",
				Stream: &stream,
				Options: map[string]interface{}{
					"seed":        42,
					"temperature": 0.0,
				},
			},
		}
		resp = [2][]string{
			[]string{"sunlight"},
			[]string{"england", "english", "massachusetts", "pilgrims"},
		}
	)
	var wg sync.WaitGroup
	wg.Add(len(req))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
	defer cancel()
	for i := 0; i < len(req); i++ {
		go func(i int) {
			defer wg.Done()
			GenerateTestHelper(ctx, t, &http.Client{}, req[i], resp[i])
		}(i)
	}
	wg.Wait()
}
