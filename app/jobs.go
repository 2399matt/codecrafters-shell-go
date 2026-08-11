package main

type Job struct {
	id      int
	command string
	pid     int
	isDone  bool
}

var runningId int = 1

var jobs = make(map[int]*Job)
