package main

type Job struct {
	id  int
	pid int
}

var runningId int = 1

var jobs = make(map[int]Job)
