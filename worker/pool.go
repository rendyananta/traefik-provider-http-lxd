package worker

type Pool struct {
	queue     chan func()
	maxWorker int
}

func NewWorkerPool(maxWorker int) *Pool {
	p := &Pool{
		maxWorker: maxWorker,
	}

	p.listen()
	return p
}

func (p *Pool) AddTask(task func()) {
	p.queue <- task
}

func (p *Pool) listen() {
	for i := 0; i < p.maxWorker; i++ {
		go func(workerID int) {
			for job := range p.queue {
				job()
			}
		}(i + 1)
	}
}
