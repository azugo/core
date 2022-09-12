package core

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testTask struct {
	started bool
}

func (t *testTask) Start(_ context.Context) error {
	t.started = true
	return nil
}

func (t *testTask) Stop() {
	t.started = false
}

func (t *testTask) Name() string {
	return "test"
}

type testErrTask struct{}

func (t *testErrTask) Start(_ context.Context) error {
	return errors.New("test start error")
}

func (t *testErrTask) Stop() {}

func (t *testErrTask) Name() string {
	return "err-test"
}

func TestTaskStart(t *testing.T) {
	a, cleanup, _, err := newTestApp()
	require.NoError(t, err)
	t.Cleanup(cleanup)

	task := &testTask{}
	assert.NoError(t, a.AddTask(task))

	assert.False(t, task.started)

	assert.NoError(t, a.Start())

	assert.True(t, task.started)
}

func TestAddTaskAfterStart(t *testing.T) {
	a, cleanup, _, err := newTestApp()
	require.NoError(t, err)
	t.Cleanup(cleanup)

	assert.NoError(t, a.Start())

	task := &testTask{}
	assert.NoError(t, a.AddTask(task))
	assert.True(t, task.started)
}

func TestTaskStartError(t *testing.T) {
	a, cleanup, _, err := newTestApp()
	require.NoError(t, err)
	t.Cleanup(cleanup)

	task := &testErrTask{}
	assert.NoError(t, a.AddTask(task))

	assert.ErrorContains(t, a.Start(), "test start error")
}

func TestTaskAddError(t *testing.T) {
	a, cleanup, _, err := newTestApp()
	require.NoError(t, err)
	t.Cleanup(cleanup)

	assert.NoError(t, a.Start())

	task := &testErrTask{}
	assert.ErrorContains(t, a.AddTask(task), "test start error")
}
