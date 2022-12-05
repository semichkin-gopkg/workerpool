# Worker pool pattern realization

```go
package main

import (
	"context"
	"errors"
	"github.com/semichkin-gopkg/workerpool"
	"log"
)

type Input struct {
	A float64
	B float64
}

func main() {
	divider := workerpool.NewPool(func(ctx context.Context, job *workerpool.Job[Input, float64]) {
		if job.Payload.B == 0 {
			job.Promise.Reject(errors.New("division by zero"))
			return
		}

		job.Promise.Resolve(job.Payload.A / job.Payload.B)
	})

	go func() { divider.Run() }()

	ctx := context.Background()

	log.Println(divider.Do(Input{1, 1}).Wait(ctx).Payload) // 1
	log.Println(divider.Do(Input{1, 0}).Wait(ctx).Error)   // division by zero

	divider.Stop(context.Background())

	log.Println(divider.Do(Input{1, 1}).Wait(ctx).Error) // workerpool: pool stopped
}
```