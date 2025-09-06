package messages

// ClientErrorMessages contains all error messages sent to clients
var ClientErrorMessages = struct {
	Unauthorized        string
	InternalServerError string
	DailyLimitExceeded  string
}{
	Unauthorized:        "[AFL] Unauthorized",
	InternalServerError: "[AFL] Internal Server Error",
	DailyLimitExceeded:  "[AFL] Reached daily limit. Resets at 4am UTC+8.",
}
