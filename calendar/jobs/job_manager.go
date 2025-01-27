// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package jobs

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/mattermost/mattermost/server/public/pluginapi/cluster"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/engine"
)

type JobManager struct {
	env            engine.Env
	papi           cluster.JobPluginAPI
	registeredJobs sync.Map
	activeJobs     sync.Map
}

type RegisteredJob struct {
	work     func(env engine.Env)
	id       string
	interval time.Duration
}

var scheduleFunc = func(api cluster.JobPluginAPI, id string, wait cluster.NextWaitInterval, cb func()) (io.Closer, error) {
	return cluster.Schedule(api, id, wait, cb)
}

type activeJob struct {
	ScheduledJob io.Closer
	Context      context.Context
	RegisteredJob
}

func newActiveJob(ctx context.Context, rj RegisteredJob, sched io.Closer) *activeJob {
	return &activeJob{
		RegisteredJob: rj,
		ScheduledJob:  sched,
		Context:       ctx,
	}
}

// NewJobManager creates a JobManager for to let plugin.go coordinate with the scheduled jobs.
func NewJobManager(papi cluster.JobPluginAPI, env engine.Env) *JobManager {
	return &JobManager{
		papi: papi,
		env:  env,
	}
}

// AddJob accepts a RegisteredJob, stores it, and activates it if enabled.
func (jm *JobManager) AddJob(job RegisteredJob) {
	jm.registeredJobs.Store(job.id, job)
	err := jm.activateJob(job)
	if err != nil {
		jm.env.Logger.Warnf("Error activating %s job. %v", job.id, err)
	}
}

// Close deactivates all active jobs. It is called in the plugin hook OnDeactivate.
func (jm *JobManager) Close() error {
	jm.env.Logger.Debugf("Deactivating all jobs due to plugin deactivation.")
	jm.activeJobs.Range(func(k interface{}, v interface{}) bool {
		job := v.(*activeJob)
		err := jm.deactivateJob(job.RegisteredJob)
		if err != nil {
			jm.env.Logger.Warnf("Failed to deactivate %s job: %v", job.id, err)
		}

		return true
	})
	return nil
}

// activateJob creates an ActiveJob, starts it, and stores it in the job manager.
func (jm *JobManager) activateJob(job RegisteredJob) error {
	scheduled, err := scheduleFunc(jm.papi, job.id, cluster.MakeWaitForRoundedInterval(job.interval), func() { job.work(jm.getEnv()) })
	if err != nil {
		return err
	}

	actJob := newActiveJob(context.Background(), job, scheduled)

	jm.activeJobs.Store(job.id, actJob)
	jm.env.Logger.Debugf("Activated %s job", job.id)
	return nil
}

// deactivateJob closes the job, releasing the cluster mutex, then removes the job from the job manager.
func (jm *JobManager) deactivateJob(job RegisteredJob) error {
	v, ok := jm.activeJobs.Load(job.id)
	if !ok {
		return errors.Errorf("attempted to deactivate a non-active job %s", job.id)
	}

	scheduledJob := v.(*activeJob)
	err := scheduledJob.ScheduledJob.Close()
	if err != nil {
		return err
	}

	jm.activeJobs.Delete(job.id)
	jm.env.Logger.Debugf("Deactivated %s job", job.id)
	return nil
}

// getEnv returns the engine.Env stored on the job manager
func (jm *JobManager) getEnv() engine.Env {
	return jm.env
}
