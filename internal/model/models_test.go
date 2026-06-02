package model_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timaogurtzova/gophermart/internal/model"
)

func TestPointsJSON(t *testing.T) {
	points, err := model.NewPoints("500.50")
	require.NoError(t, err)

	data, err := json.Marshal(points)
	require.NoError(t, err)

	assert.Equal(t, "500.5", string(data))
}

func TestPointsValue(t *testing.T) {
	points, err := model.NewPoints("42.00")
	require.NoError(t, err)

	value, err := points.Value()
	require.NoError(t, err)

	assert.Equal(t, "42", value)
}

func TestPointsUnmarshalJSON(t *testing.T) {
	var points model.Points

	err := json.Unmarshal([]byte("751.25"), &points)
	require.NoError(t, err)

	assert.Equal(t, "751.25", points.String())
}

func TestPointsZeroValue(t *testing.T) {
	var points model.Points

	data, err := json.Marshal(points)
	require.NoError(t, err)

	value, err := points.Value()
	require.NoError(t, err)

	assert.Equal(t, "0", string(data))
	assert.Equal(t, "0", value)
}
