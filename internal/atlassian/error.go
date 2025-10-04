package atlassian

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Error represents an Atlassian REST error response.
type Error struct {
	StatusCode    int               `json:"-"`
	Message       string            `json:"message"`
	ErrorMessages []string          `json:"errorMessages"`
	Errors        map[string]string `json:"errors"`
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}

	if e.Message != "" {
		return fmt.Sprintf("atlassian: %d %s", e.StatusCode, e.Message)
	}

	if len(e.ErrorMessages) > 0 {
		return fmt.Sprintf("atlassian: %d %s", e.StatusCode, e.ErrorMessages[0])
	}

	return fmt.Sprintf("atlassian: %d", e.StatusCode)
}

func parseError(res *http.Response) error {
	data, _ := io.ReadAll(res.Body)
	errRes := &Error{StatusCode: res.StatusCode}
	if len(data) > 0 {
		_ = json.Unmarshal(data, errRes)
	}

	if errRes.Message == "" && len(errRes.ErrorMessages) == 0 && len(errRes.Errors) == 0 {
		errRes.Message = string(data)
	}

	return errRes
}
