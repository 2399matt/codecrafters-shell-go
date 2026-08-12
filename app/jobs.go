package main

import (
	"cmp"
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"sync"
)

type Job struct {
	id       int
	command  string
	pid      int
	isDone   bool
	reported bool
}

type JobHandler struct {
	mu        *sync.Mutex
	jobs      map[int]*Job
	runningId int
}

var cleanup = make(chan *Job, 10)

var jobHandler = &JobHandler{
	mu:        &sync.Mutex{},
	jobs:      make(map[int]*Job),
	runningId: 1,
}

func (command *Command) setupJob() {
	cmd := exec.Command(command.keyword, command.parts...)
	cmd.Stderr = command.errOut
	cmd.Stdout = command.out
	if err := cmd.Start(); err != nil {
		return
	}
	jobHandler.mu.Lock()
	curr := jobHandler.runningId
	jobHandler.runningId++
	task := fmt.Sprintf("%s", command.keyword+" "+strings.Join(command.parts, " "))
	job := &Job{id: curr, pid: cmd.Process.Pid, command: task}
	jobHandler.jobs[curr] = job
	jobHandler.mu.Unlock()
	fmt.Fprintf(command.out, "[%d] %d\n", curr, cmd.Process.Pid)
	go func(job *Job) {
		cmd.Wait()
		jobHandler.mu.Lock()
		job.isDone = true
		jobHandler.mu.Unlock()
		cleanup <- job
	}(job)
}

func handleJobs(command *Command) {
	sJobs := make([]*Job, 0)
	jobHandler.mu.Lock()
	for _, v := range jobHandler.jobs {
		sJobs = append(sJobs, v)
	}
	jobHandler.mu.Unlock()
	slices.SortFunc(sJobs, func(a, b *Job) int {
		return cmp.Compare(a.id, b.id)
	})
	for i, job := range sJobs {
		status := ""
		trailing := ""
		jobHandler.mu.Lock()
		if job.isDone {
			status = "Done"
			if !job.reported {
				job.reported = true
				delete(jobHandler.jobs, job.id)
				jobHandler.runningId--
			} else {
				jobHandler.mu.Unlock()
				continue
			}
		} else {
			status = "Running"
			trailing = "&"
		}
		jobHandler.mu.Unlock()
		switch i {
		case len(sJobs) - 1:
			fmt.Fprintf(command.out, "[%d]+  %s                 %s %s\n", job.id, status, job.command, trailing)
		case len(sJobs) - 2:
			fmt.Fprintf(command.out, "[%d]-  %s                 %s %s\n", job.id, status, job.command, trailing)
		default:
			fmt.Fprintf(command.out, "[%d]   %s                 %s %s\n", job.id, status, job.command, trailing)
		}
	}
}
