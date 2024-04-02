package nodemanager

import "fmt"

// Progress 节点步骤进度条
type Progress struct {
	i   int
	sum int
}

func NewProgress(sum int) *Progress {
	return &Progress{
		sum: sum,
	}
}

func (p *Progress) Add() string {
	p.i++
	if p.i > p.sum {
		p.i = p.sum
	}

	return p.String()
}

func (p *Progress) Reset() {
	p.i = 0
}

func (p *Progress) String() string {
	return fmt.Sprintf("%d/%d", p.i, p.sum)
}

func defaultPort(in string) string {
	if in == "" {
		return "22"
	}

	return in
}
