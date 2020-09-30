package vssh_test

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/yahoo/vssh"
)

func Example_stream() {
	var wg sync.WaitGroup

	vs := vssh.New().Start()
	config, err := vssh.GetConfigPEM("ubuntu", "myaws.pem")
	if err != nil {
		log.Fatal(err)
	}

	for _, addr := range []string{"3.101.78.17:22", "13.57.12.15:22"} {
		vs.AddClient(addr, config, vssh.SetMaxSessions(4))
	}

	vs.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := "uname -a"
	timeout, _ := time.ParseDuration("6s")
	respChan := vs.Run(ctx, cmd, timeout)

	for resp := range respChan {
		if err := resp.Err(); err != nil {
			log.Println("error", err)
			continue
		}

		wg.Add(1)
		go func(resp *vssh.Response) {
			defer wg.Done()
			handler(resp)
		}(resp)
	}

	wg.Wait()
}

func handler(resp *vssh.Response) {
	stream := resp.GetStream()
	defer stream.Close()

	for stream.ScanStdout() {
		txt := stream.TextStdout()
		fmt.Println(resp.ID(), txt)
	}

	if err := stream.Err(); err != nil {
		log.Println("error", err)
	}
}
