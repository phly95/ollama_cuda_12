package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnloadBestFitRunner(t *testing.T) {
	r := &runnerRef{
		model: "dummy",
	}
	r.refCond.L = &r.refMu

	loaded["dummy"] = r
	unloadBestFitRunner()
	assert.Len(t, loaded, 0)
}
