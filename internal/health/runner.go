package health

import "time"

const defaultTimeout = 3 * time.Second

func Run(checkers []Checker) Report {
	type entry struct {
		name   string
		result Result
	}
	ch := make(chan entry, len(checkers))
	for _, c := range checkers {
		go func(c Checker) {
			ch <- entry{name: c.Name(), result: c.Check(defaultTimeout)}
		}(c)
	}
	checks := make(map[string]Result, len(checkers))
	overall := StatusOk
	for range checkers {
		e := <-ch
		checks[e.name] = e.result
		if e.result.Status == StatusFail {
			overall = StatusFail
		} else if e.result.Status == StatusDegre && overall != StatusFail {
			overall = StatusDegre
		}
	}
	return Report{Status: overall, Checkes: checks}
}
