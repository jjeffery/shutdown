package shutdown_test

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/jjeffery/shutdown"
)

func Example() {
	shutdown.RegisterCallback(callback)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		goroutine1()
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		goroutine2(shutdown.Context())
		wg.Done()
	}()

	wg.Wait()
}

// callback is called when a shutdown is first requested
func callback() {
	log.Println("shutdown requested")
}

// goroutine1 prints a message once per second until shutdown is requested
func goroutine1() {
	for {
		select {
		case <-time.After(time.Second):
			log.Println("heartbeat")
		case <-shutdown.InProgress():
			log.Println("shutdown (1)")
			return
		}
	}
}

// goroutine2 waits for a context to be done before returning
func goroutine2(ctx context.Context) {
	select {
	case <-ctx.Done():
		log.Println("shutdown (2)")
	}
}
