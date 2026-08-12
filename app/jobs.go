package main

import "sync"

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
	//currentJob  int
	//previousJob int
}

var cleanup = make(chan *Job, 10)

//var jobs = make(map[int]*Job)

var jobHandler = &JobHandler{
	mu:        &sync.Mutex{},
	jobs:      make(map[int]*Job),
	runningId: 1,
}
