package core

import (
	"context"
)

// Tasker interface for staring task
type Tasker interface {
	// Start task
	//
	// This method is called when task is started and MUST not block.
	Start(ctx context.Context) error
	// Stop task
	Stop()
	// Name returns task name
	Name() string
}

func (a *App) startTasks() error {
	a.stlock.Lock()
	defer a.stlock.Unlock()

	if a.started {
		return nil
	}

	for _, task := range a.tasks {
		if err := task.Start(a.BackgroundContext()); err != nil {
			return err
		}
	}

	a.started = true
	return nil
}

func (a *App) stopTasks() {
	a.stlock.Lock()
	defer a.stlock.Unlock()

	for _, task := range a.tasks {
		task.Stop()
	}

	a.started = false
}

// AddTask adds task to the app.
//
// If app is already started, task is started immediately.
func (a *App) AddTask(task Tasker) error {
	a.stlock.Lock()
	defer a.stlock.Unlock()

	a.tasks = append(a.tasks, task)

	if a.started {
		return task.Start(a.BackgroundContext())
	}
	return nil
}
