# Work less, do more

This library provides storage agnostic leader election.

## Usage

Create the candidate using `New` and pass your storage, that implements `Storage` interface. For additional configuration look at `Option` and `WithXxx` functions.

Now the candidate is already launched and (after a bit) will have the candidate status. To get it, just call `IsLeader`. 

Here is a simple demo:

```go
func main() {
	ctx := context.Background()

	var storage Storage
	// Fill with any shared storage implementation
    // You can also use our adapters from `adapter` package

	candidate := New(ctx, storage)

	workFn := func() {
		if !candidate.IsLeader() {
			return
		}
		// do work
		fmt.Println("notification sent")
	}

	for range time.Tick(1 * time.Minute) {
		workFn()
	}

	// Now if we run this code on 2 and more nodes, we can be sure that
	// only one of them will actively run the job
}
```

If you have multiple workers and you don't want them all to depend on single candidate (i.e. if leader then all workers will run, if not then no worker will run) you can create candidate for each of them with different keys using `WithKey` option:

```go
func main() {
    // ...

	candidate1 := New(ctx, storage, WithKey("work1"))
	work1Fn := func() {
		if !candidate1.IsLeader() {
			return
		}
		// do work
		fmt.Println("notification sent")
	}

	candidate2 := New(ctx, storage, WithKey("work2"))
	work2Fn := func() {
		if !candidate2.IsLeader() {
			return
		}
		// do work
		fmt.Println("notification sent")
	}

    // etc
    // ...
}
```

This way each of the workers will has it's own indepentend leader election process.

You can integrate this easely with your asynchronous workers. E.g. you can pass it to [go-co-op/gocron](https://github.com/go-co-op/gocron) job scheduler, with their [`WithDistributedElector`](https://pkg.go.dev/github.com/go-co-op/gocron/v2#WithDistributedElector) option (you need to decorate the `Candidate` by yourself to match the interface).
