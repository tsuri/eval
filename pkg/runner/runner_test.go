package runner

import "testing"

func TestStatus(t *testing.T) {
	testCases := []struct {
		value  taskStatus
		result string
	}{
		{value: Unknown, result: "Unknown"},
		{value: Blocked, result: "Blocked"},
		{value: Queued, result: "Queued"},
		{value: Running, result: "Running"},
		{value: Succeded, result: "Succeded"},
		{value: Failed, result: "Failed"},
		{value: Lost, result: "Lost"},
		{value: Skipped, result: "Skipped"},
	}

	for _, testCase := range testCases {
		if testCase.value.String() != testCase.result {
			t.Errorf("%d is rendered as \"%s\" (expecting \"%s\")",
				testCase.value,
				testCase.value.String(),
				testCase.result)
		}
	}
}
