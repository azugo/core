package core

import (
	"context"
	"errors"
	"testing"

	"github.com/go-quicktest/qt"
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
	qt.Assert(t, qt.IsNil(err))
	t.Cleanup(cleanup)

	task := &testTask{}
	err = a.AddTask(task)
	qt.Check(t, qt.IsNil(err))

	qt.Check(t, qt.IsFalse(task.started))

	err = a.Start()
	qt.Check(t, qt.IsNil(err))

	qt.Check(t, qt.IsTrue(task.started))
}

func TestAddTaskAfterStart(t *testing.T) {
	a, cleanup, _, err := newTestApp()
	qt.Assert(t, qt.IsNil(err))
	t.Cleanup(cleanup)

	err = a.Start()
	qt.Assert(t, qt.IsNil(err))

	task := &testTask{}
	err = a.AddTask(task)
	qt.Check(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(task.started))
}

func TestTaskStartError(t *testing.T) {
	a, cleanup, _, err := newTestApp()
	qt.Assert(t, qt.IsNil(err))
	t.Cleanup(cleanup)

	task := &testErrTask{}
	err = a.AddTask(task)
	qt.Check(t, qt.IsNil(err))

	err = a.Start()
	qt.Check(t, qt.ErrorMatches(err, "test start error"))
}

func TestTaskAddError(t *testing.T) {
	a, cleanup, _, err := newTestApp()
	qt.Assert(t, qt.IsNil(err))
	t.Cleanup(cleanup)

	err = a.Start()
	qt.Assert(t, qt.IsNil(err))

	task := &testErrTask{}
	err = a.AddTask(task)
	qt.Check(t, qt.ErrorMatches(err, "test start error"))
}
