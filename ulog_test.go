package ulog

import (
	"bytes"
	"encoding/json"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"testing"
)

type testLog struct {
	ID    string `json:"ID"`
	Level string `json:"level"`
}

// 测试ID功能
func TestUlog_ID(t *testing.T) {
	var out bytes.Buffer
	logger := zerolog.New(&out).With().Logger()
	u := Ulog{Logger: logger}

	u.Debug().ID("test-id1").Send()

	need := testLog{ID: "test-id1", Level: "debug"}
	got := testLog{}

	err := json.Unmarshal(out.Bytes(), &got)
	assert.NoError(t, err)
	assert.Equal(t, need, got)
}
