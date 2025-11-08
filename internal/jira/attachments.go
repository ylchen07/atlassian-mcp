package jira

import (
	"context"
	"fmt"
)

// AddAttachment uploads a file attachment to the specified issue.
func (s *Service) AddAttachment(ctx context.Context, key, filename string, data []byte) error {
	if key == "" {
		return fmt.Errorf("jira: issue key required")
	}
	if filename == "" {
		return fmt.Errorf("jira: attachment filename required")
	}
	if len(data) == 0 {
		return fmt.Errorf("jira: attachment data required")
	}

	// For multipart upload, we'll need to implement a specialized method in the HTTP client
	// For now, return an error indicating this needs implementation
	return fmt.Errorf("jira: attachment upload not yet implemented in HTTP client")
}
