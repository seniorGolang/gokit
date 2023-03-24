package queue

import (
	"io"
	"time"
)

func (queue *Queue) ReadChan(errs chan error, pollInterval time.Duration) (rd chan []byte) {

	rd = make(chan []byte)

	go func() {
		for {
			payload, err := queue.Get()

			if err != nil {

				if err == io.EOF {
					time.Sleep(pollInterval)
					continue
				}

				if errs != nil {
					errs <- err
				}
				continue
			}
			rd <- payload
		}
	}()
	return
}

func (queue *Queue) WriteChan(errs chan error) (wr chan []byte) {

	wr = make(chan []byte)

	go func() {
		for {
			payload := <-wr
			if err := queue.Set(payload); err != nil {
				if errs != nil {
					errs <- err
				}
				continue
			}
		}
	}()
	return
}
