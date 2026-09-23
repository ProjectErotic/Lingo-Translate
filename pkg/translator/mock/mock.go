package mock

import (
	"context"
	"fmt"
	"strings"

	"lingo-translate/pkg/translator"
)

type MockTranslator struct {
	Prefix string
}

func New(prefix string) *MockTranslator {
	if prefix == "" {
		prefix = "[TH] "
	}
	return &MockTranslator{Prefix: prefix}
}

func (m *MockTranslator) Name() string {
	return "mock"
}

func (m *MockTranslator) Translate(ctx context.Context, texts []string, opts translator.Options) ([]translator.Result, error) {
	results := make([]translator.Result, len(texts))
	for i, text := range texts {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		target := fmt.Sprintf("%s%s", m.Prefix, text)
		// Check glossary if applicable
		for k, v := range opts.Glossary {
			if strings.Contains(target, k) {
				target = strings.ReplaceAll(target, k, v)
			}
		}

		results[i] = translator.Result{
			Source:     text,
			Target:     target,
			Translator: "mock",
		}
	}
	return results, nil
}
