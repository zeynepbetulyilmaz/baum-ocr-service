package ocr

type Job struct {
	DocumentID string
	FilePath   string
	Ext        string
	Lang       string
}

type Queue struct {
	ch chan Job
}

func NewQueue(buffer int) *Queue {
	return &Queue{ch: make(chan Job, buffer)}
}

func (q *Queue) Enqueue(j Job) {
	q.ch <- j
}

func (q *Queue) Chan() <-chan Job {
	return q.ch
}

func (q *Queue) StartWorkers(n int, process func(Job)) {
	for i := 0; i < n; i++ {
		go func() {
			for j := range q.ch {
				process(j)
			}
		}()
	}
}
