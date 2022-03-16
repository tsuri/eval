package runner

type taskStatus int

const (
	Unknown taskStatus = iota
	Blocked
	Queued
	Running
	Succeded
	Failed
	Lost
	Skipped
)

func (s taskStatus) String() string {
	return [...]string{
		"Unknown",
		"Blocked",
		"Queued",
		"Running",
		"Succeded",
		"Failed",
		"Lost",
		"Skipped"}[s]
}

type bachJob struct {
	name string
}

type batchTask struct {
	name   string
	status taskStatus
}

func NewBatchTask(name string) *batchTask {
	t := batchTask{
		name:   name,
		status: Unknown,
	}
	return &t
}

type batchExecutor struct {
}

func (executor batchExecutor) Add(task batchTask) {

}

func (executor batchExecutor) AdvanceTime(t int) {
}
