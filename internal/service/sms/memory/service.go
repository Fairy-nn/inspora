package memory

import (
	"context"
	"fmt"
)

type Service struct{}

func NewMemorySMSService() *Service {
	return &Service{}
}

func (s *Service) Send(ctx context.Context, tpl string, args []string, numbers ...string) error {
	// Simulate sending SMS by printing to console
	for _, number := range numbers {
		fmt.Printf("Sending SMS to %s with template %s and args %v\n", number, tpl, args)
	}
	return nil
}
