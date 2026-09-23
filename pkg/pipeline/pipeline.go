package pipeline

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"lingo-translate/pkg/masker"
	"lingo-translate/pkg/model"
	"lingo-translate/pkg/storage"
	"lingo-translate/pkg/translator"
)

type Config struct {
	BatchSize   int
	Concurrency int
	Retries     int
	RetryDelay  time.Duration
}

type Pipeline struct {
	cfg        Config
	store      *storage.Storage
	translator translator.Translator
}

func New(store *storage.Storage, trans translator.Translator, cfg Config) *Pipeline {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 10
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 4
	}
	if cfg.Retries <= 0 {
		cfg.Retries = 2
	}
	if cfg.RetryDelay == 0 {
		cfg.RetryDelay = 1 * time.Second
	}

	return &Pipeline{
		cfg:        cfg,
		store:      store,
		translator: trans,
	}
}

// ProgressCallback is called whenever translation progress updates
type ProgressCallback func(p model.TranslationProgress)

// Run processes entries through TM cache and Translator
func (p *Pipeline) Run(ctx context.Context, entries []model.TextEntry, opts translator.Options, progressCb ProgressCallback) ([]model.TextEntry, error) {
	total := len(entries)
	if total == 0 {
		return entries, nil
	}

	var completedCount int64
	var failedCount int64

	report := func(currFile string) {
		if progressCb != nil {
			c := atomic.LoadInt64(&completedCount)
			f := atomic.LoadInt64(&failedCount)
			pct := 0.0
			if total > 0 {
				pct = float64(c) / float64(total) * 100
			}
			progressCb(model.TranslationProgress{
				Total:       total,
				Completed:   int(c),
				Failed:      int(f),
				CurrentFile: currFile,
				Percent:     pct,
			})
		}
	}

	// 1. First pass: check Translation Memory cache
	var toTranslateIndices []int
	for i := range entries {
		if entries[i].Target != "" && entries[i].Status != model.StatusUntranslated {
			atomic.AddInt64(&completedCount, 1)
			continue
		}

		if p.store != nil {
			cached, hit, err := p.store.GetCache(entries[i].Source, opts.SourceLang, opts.TargetLang)
			if err == nil && hit && cached != "" {
				entries[i].Target = cached
				entries[i].Status = model.StatusTranslated
				entries[i].Translator = "tm_cache"
				entries[i].UpdatedAt = time.Now()
				atomic.AddInt64(&completedCount, 1)
				continue
			}
		}

		toTranslateIndices = append(toTranslateIndices, i)
	}

	report("Cache checked")

	if len(toTranslateIndices) == 0 {
		return entries, nil
	}

	// Chunk indices into batches
	var batches [][]int
	for i := 0; i < len(toTranslateIndices); i += p.cfg.BatchSize {
		end := i + p.cfg.BatchSize
		if end > len(toTranslateIndices) {
			end = len(toTranslateIndices)
		}
		batches = append(batches, toTranslateIndices[i:end])
	}

	batchChan := make(chan []int, len(batches))
	for _, b := range batches {
		batchChan <- b
	}
	close(batchChan)

	var wg sync.WaitGroup
	var mu sync.Mutex

	for w := 0; w < p.cfg.Concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for batch := range batchChan {
				select {
				case <-ctx.Done():
					return
				default:
				}

				// Prepare masked texts and tags
				maskedTexts := make([]string, len(batch))
				maskResults := make([]masker.MaskResult, len(batch))
				for idx, entryIdx := range batch {
					maskResults[idx] = masker.Mask(entries[entryIdx].Source)
					maskedTexts[idx] = maskResults[idx].MaskedText
				}

				// Translate with retry
				var transResults []translator.Result
				var err error
				for attempt := 0; attempt <= p.cfg.Retries; attempt++ {
					transResults, err = p.translator.Translate(ctx, maskedTexts, opts)
					if err == nil {
						break
					}
					time.Sleep(p.cfg.RetryDelay)
				}

				if err != nil {
					atomic.AddInt64(&failedCount, int64(len(batch)))
					continue
				}

				// Unmask and update results
				for idx, entryIdx := range batch {
					if idx >= len(transResults) || transResults[idx].Error != nil || transResults[idx].Target == "" {
						mu.Lock()
						entries[entryIdx].Status = model.StatusUntranslated
						entries[entryIdx].UpdatedAt = time.Now()
						if p.store != nil {
							_ = p.store.UpdateEntryTarget(entries[entryIdx].ID, entries[entryIdx].Source, model.StatusUntranslated, "")
						}
						mu.Unlock()
						atomic.AddInt64(&failedCount, 1)
						continue
					}

					unmasked := masker.Unmask(transResults[idx].Target, maskResults[idx].TagMap)

					mu.Lock()
					entries[entryIdx].Target = unmasked
					entries[entryIdx].Status = model.StatusTranslated
					entries[entryIdx].Translator = transResults[idx].Translator
					entries[entryIdx].UpdatedAt = time.Now()

					// Save to DB and Cache
					if p.store != nil {
						_ = p.store.UpdateEntryTarget(entries[entryIdx].ID, unmasked, model.StatusTranslated, entries[entryIdx].Translator)
						_ = p.store.SetCache(entries[entryIdx].Source, unmasked, opts.SourceLang, opts.TargetLang, entries[entryIdx].Translator)
					}
					mu.Unlock()

					atomic.AddInt64(&completedCount, 1)
				}

				report(entries[batch[0]].FilePath)
			}
		}()
	}

	wg.Wait()
	report("Done")

	return entries, nil
}
