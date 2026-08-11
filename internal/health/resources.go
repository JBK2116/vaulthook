package health

import "runtime"

// Resources stores the amount of resources that should be allocated to run the application optimally.
type Resources struct {
	QueueWorkers int
	RetryWorkers int
	QueueBatch   int
	RetryBatch   int
	DBMaxConns   int
}

// ComputeResources calculates the amount of resources that should be allocated to run the application optimally.
func ComputeResources() Resources {
	numCPU := runtime.NumCPU()

	// calculate amount of connections reserved for db access
	dbMaxConns := max(int32(numCPU*2), 10)

	// calculate the amount of worker instances to use for application
	reserved := int32(4)
	budget := max(dbMaxConns-reserved, 4)

	// allocate count for each worker type.
	queueWorkers := max(2, int(budget)*2/3)
	retryWorkers := max(1, int(budget)-queueWorkers)

	// calculate the batch count per worker
	queueBatch := clamp(numCPU*6, 20, 100)
	retryBatch := clamp(numCPU*3, 10, 50)

	return Resources{
		QueueWorkers: queueWorkers,
		RetryWorkers: retryWorkers,
		QueueBatch:   queueBatch,
		RetryBatch:   retryBatch,
		DBMaxConns:   int(dbMaxConns),
	}

}

// clamp restricts a value between a minimum and maximum bounds.
func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
