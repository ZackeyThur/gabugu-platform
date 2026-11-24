//go:build unit

package service

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadBoundedOpenAICountTokensBody(t *testing.T) {
	t.Run("accepts normal payload", func(t *testing.T) {
		body, err := readBoundedOpenAICountTokensBody(strings.NewReader(`{"input_tokens":42}`))
		require.NoError(t, err)
		require.Equal(t, `{"input_tokens":42}`, string(body))
	})

	t.Run("accepts exact limit", func(t *testing.T) {
		exact := bytes.Repeat([]byte("a"), maxOpenAICountTokensResponseBytes)
		body, err := readBoundedOpenAICountTokensBody(bytes.NewReader(exact))
		require.NoError(t, err)
		require.Len(t, body, maxOpenAICountTokensResponseBytes)
	})

	t.Run("rejects oversized upstream body", func(t *testing.T) {
		oversized := bytes.Repeat([]byte("b"), maxOpenAICountTokensResponseBytes+1)
		body, err := readBoundedOpenAICountTokensBody(bytes.NewReader(oversized))
		require.Error(t, err)
		require.Nil(t, body)
		require.Contains(t, err.Error(), "exceeds")
	})
}
