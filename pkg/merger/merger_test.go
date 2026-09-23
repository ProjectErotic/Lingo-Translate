package merger

import (
	"testing"

	"lingo-translate/pkg/model"
)

func TestMergeEntries(t *testing.T) {
	// Existing translations from game v1.0
	existing := []model.TextEntry{
		{
			ID:       "msg1",
			FilePath: "Map001.json",
			KeyPath:  "events[1].pages[0].parameters[0]",
			Source:   "Welcome to town!",
			Target:   "ยินดีต้อนรับสู่เมือง!",
			Status:   model.StatusTranslated,
		},
		{
			ID:       "msg2",
			FilePath: "Map001.json",
			KeyPath:  "events[1].pages[0].parameters[1]",
			Source:   "I am the mayor.",
			Target:   "ฉันคือนายกเทศมนตรี",
			Status:   model.StatusTranslated,
		},
		{
			ID:       "msg_removed",
			FilePath: "Map001.json",
			KeyPath:  "events[1].pages[0].parameters[2]",
			Source:   "This quest will be deleted in v1.1.",
			Target:   "เควสนี้จะถูกลบใน v1.1",
			Status:   model.StatusTranslated,
		},
	}

	// Incoming extraction from game v1.1
	// - msg1: identical location (exact match)
	// - msg2: line moved to parameters[2] (fuzzy match)
	// - msg_new: brand new line added in v1.1
	// - msg_removed: was deleted by developer in v1.1
	incoming := []model.TextEntry{
		{
			ID:       "new_msg1",
			FilePath: "Map001.json",
			KeyPath:  "events[1].pages[0].parameters[0]",
			Source:   "Welcome to town!",
		},
		{
			ID:       "new_msg2",
			FilePath: "Map001.json",
			KeyPath:  "events[1].pages[0].parameters[2]", // Shifted key
			Source:   "I am the mayor.",
		},
		{
			ID:       "new_msg3",
			FilePath: "Map001.json",
			KeyPath:  "events[1].pages[0].parameters[3]",
			Source:   "Here is a brand new quest!", // New line
		},
	}

	merger := New()
	merged, stats := merger.MergeEntries(existing, incoming)

	if stats.TotalNew != 3 {
		t.Fatalf("Expected 3 incoming entries, got %d", stats.TotalNew)
	}
	if stats.ExactMatches != 1 {
		t.Errorf("Expected 1 exact match, got %d", stats.ExactMatches)
	}
	if stats.FuzzyMatches != 1 {
		t.Errorf("Expected 1 fuzzy match, got %d", stats.FuzzyMatches)
	}
	if stats.NewUntranslated != 1 {
		t.Errorf("Expected 1 new untranslated entry, got %d", stats.NewUntranslated)
	}
	if stats.ObsoleteCount != 1 {
		t.Errorf("Expected 1 obsolete entry, got %d", stats.ObsoleteCount)
	}

	// Check carried over translations
	if merged[0].Target != "ยินดีต้อนรับสู่เมือง!" || merged[0].Status != model.StatusTranslated {
		t.Errorf("Exact match failed: target='%s', status='%s'", merged[0].Target, merged[0].Status)
	}
	if merged[1].Target != "ฉันคือนายกเทศมนตรี" || merged[1].Status != model.StatusReviewed {
		t.Errorf("Fuzzy match failed: target='%s', status='%s'", merged[1].Target, merged[1].Status)
	}
	if merged[2].Target != "" || merged[2].Status != model.StatusUntranslated {
		t.Errorf("New line failed: target='%s', status='%s'", merged[2].Target, merged[2].Status)
	}

	t.Logf("Merge stats verified in %v!", stats.Duration)
}
