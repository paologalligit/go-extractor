package team

import (
	"sync"
)

// Requirement is a receive-only channel of T (jobs)
type Requirement[T any] <-chan T

// Outcome is a send-only channel of U (results)
type Outcome[U any] chan<- U

// Task represents a unit of work: requirements in, outcome out
type Task[T any, U any] struct {
	Requirements Requirement[T]
	Outcome      Outcome[U]
}

// WorkerFunc is a function that processes a job of type T and returns a result of type U (and optionally error)
type WorkerFunc[T any, U any] func(T) (U, error)

// Team is a generic worker pool
// WorkerCount: number of concurrent workers
// Worker: the function to process each job
type Team[T any, U any] struct {
	WorkerCount int
	Worker      WorkerFunc[T, U]
}

// Run executes the worker pool: feeds jobs, collects results, returns result slice
func (t *Team[T, U]) Run(jobs []T) []U {
	jobChan := make(chan T, len(jobs))
	resultChan := make(chan U, len(jobs))
	var wg sync.WaitGroup

	// Start workers
	for range t.WorkerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobChan {
				if res, err := t.Worker(job); err == nil {
					resultChan <- res
				}
			}
		}()
	}

	// Feed jobs
	for _, job := range jobs {
		jobChan <- job
	}
	close(jobChan)

	// Wait for workers to finish, then close resultChan
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var results []U
	for res := range resultChan {
		results = append(results, res)
	}
	return results
}
