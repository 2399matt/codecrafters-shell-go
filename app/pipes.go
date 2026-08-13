package main

import (
	"os"
	"sync"
)

type Pipe struct {
	w *os.File
	r *os.File
}

type Pipeline struct {
	commands []*Command
	pipes    []Pipe
}

func connect(commands []*Command) *Pipeline {
	p := &Pipeline{
		commands: commands,
		pipes:    make([]Pipe, 0),
	}
	for i := 0; i < len(commands)-1; i++ {
		r, w, err := os.Pipe()
		if err != nil {
			return nil
		}
		commands[i].out = w
		commands[i+1].in = r
		p.pipes = append(p.pipes, Pipe{w: w, r: r})
	}
	return p
}

func (p *Pipeline) launchCommands() {
	wg := &sync.WaitGroup{}
	for i, cmd := range p.commands {
		wg.Add(1)
		go func(i int, cmd *Command) {
			defer wg.Done()
			cmd.handleCommand()
			if i < len(p.pipes) {
				p.pipes[i].w.Close()
			}
		}(i, cmd)
	}
	wg.Wait()
	p.close()
}

// no point in catching close errors
// most should already be closed at this point?
func (p *Pipeline) close() {
	for _, pipe := range p.pipes {
		pipe.w.Close()
		pipe.r.Close()
	}
	p.pipes = nil
}
