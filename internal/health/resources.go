package health

import "runtime"

// Resource tuning constants used by ComputeResources.
const (
	// dbConnsPerCPU scales the database connection pool with the CPU count.
	dbConnsPerCPU = 2
	// minDBConns is the smallest database connection pool allowed.
	minDBConns = 10

	// minReservedConns keeps connections free for non-worker use such as
	// migrations, admin queries, and monitoring.
	minReservedConns = 4
	// minWorkerBudget is the smallest pool of connections available to workers.
	minWorkerBudget = 4

	// minQueueWorkers and minRetryWorkers ensure each worker type is always
	// represented.
	minQueueWorkers = 2
	minRetryWorkers = 1

	// queueWorkerShareNum and queueWorkerShareDen give queue workers a 2/3
	// share of the worker budget.
	queueWorkerShareNum = 2
	queueWorkerShareDen = 3

	// batchPerCPU scales worker batch sizes with the CPU count.
	queueBatchPerCPU = 6
	retryBatchPerCPU = 3

	// batch size bounds per worker type.
	minQueueBatch = 20
	maxQueueBatch = 100
	minRetryBatch = 10
	maxRetryBatch = 50
)

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
	dbMaxConns := max(numCPU*dbConnsPerCPU, minDBConns)

	// calculate the amount of worker instances to use for application
	budget := max(dbMaxConns-minReservedConns, minWorkerBudget)

	// allocate count for each worker type.
	queueWorkers := max(minQueueWorkers, budget*queueWorkerShareNum/queueWorkerShareDen)
	retryWorkers := max(minRetryWorkers, budget-queueWorkers)

	// calculate the batch count per worker
	queueBatch := clamp(numCPU*queueBatchPerCPU, minQueueBatch, maxQueueBatch)
	retryBatch := clamp(numCPU*retryBatchPerCPU, minRetryBatch, maxRetryBatch)

	return Resources{
		QueueWorkers: queueWorkers,
		RetryWorkers: retryWorkers,
		QueueBatch:   queueBatch,
		RetryBatch:   retryBatch,
		DBMaxConns:   dbMaxConns,
	}
}

// clamp restricts a value between a minimum and maximum bounds.
func clamp(val, lo, hi int) int {
	if val < lo {
		return lo
	}
	if val > hi {
		return hi
	}
	return val
}
