package pipeline

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"lingo-translate/pkg/decision"
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
	cfg            Config
	store          *storage.Storage
	translator     translator.Translator
	fastTranslator translator.Translator
	autoRouteShort bool
	maxShortLen    int
	decisionEngine decision.DecisionEngine
	acceptUIDrafts bool
	verifyQA       bool
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
		cfg:         cfg,
		store:       store,
		translator:  trans,
		maxShortLen: 60,
	}
}

// SetTaskRouting configures the fast bulk translator and short-text routing (Hermes Architecture)
func (p *Pipeline) SetTaskRouting(fastTrans translator.Translator, autoRoute bool, maxLen int) {
	p.fastTranslator = fastTrans
	p.autoRouteShort = autoRoute
	if maxLen > 0 {
		p.maxShortLen = maxLen
	} else {
		p.maxShortLen = 60
	}
}

// SetSystemOne configures the auxiliary System One decision engine (TypeSafe AI Jev)
func (p *Pipeline) SetSystemOne(engine decision.DecisionEngine, acceptUIDrafts, verifyQA bool) {
	p.decisionEngine = engine
	p.acceptUIDrafts = acceptUIDrafts
	p.verifyQA = verifyQA
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

	// 2. Speculative Draft & Fast Task Routing Pass (if FastTranslator or SystemOne enabled)
	if p.fastTranslator != nil && (p.autoRouteShort || p.acceptUIDrafts) {
		var shortIndices []int
		var remainingIndices []int

		for _, idx := range toTranslateIndices {
			src := entries[idx].Source
			if len(src) <= p.maxShortLen {
				shortIndices = append(shortIndices, idx)
			} else {
				remainingIndices = append(remainingIndices, idx)
			}
		}

		if len(shortIndices) > 0 {
			shortTexts := make([]string, len(shortIndices))
			for i, sIdx := range shortIndices {
				shortTexts[i] = entries[sIdx].Source
			}

			fastResults, fastErr := p.fastTranslator.Translate(ctx, shortTexts, opts)
			if fastErr == nil && len(fastResults) == len(shortIndices) {
				for i, sIdx := range shortIndices {
					res := fastResults[i]
					if res.Error == nil && res.Target != "" {
						accepted := false
						if p.decisionEngine != nil && p.acceptUIDrafts {
							// System One (Jev) verifies draft
							isAccurate, conf, dErr := p.decisionEngine.Noul(ctx,
								fmt.Sprintf("Source: %s\nDraft: %s", entries[sIdx].Source, res.Target),
								"Is this draft translation accurate and natural for Thai game UI?")
							if dErr == nil && isAccurate && conf >= 0.80 {
								accepted = true
							}
						} else if p.autoRouteShort {
							accepted = true
						}

						if accepted {
							entries[sIdx].Target = res.Target
							entries[sIdx].Status = model.StatusTranslated
							entries[sIdx].Translator = res.Translator
							entries[sIdx].UpdatedAt = time.Now()
							if p.store != nil {
								_ = p.store.UpdateEntryTarget(entries[sIdx].ID, res.Target, model.StatusTranslated, entries[sIdx].Translator)
								_ = p.store.SetCache(entries[sIdx].Source, res.Target, opts.SourceLang, opts.TargetLang, entries[sIdx].Translator)
							}
							atomic.AddInt64(&completedCount, 1)
							continue
						}
					}
					// If not accepted or error, fall through to primary frontier model
					remainingIndices = append(remainingIndices, sIdx)
				}
			} else {
				remainingIndices = append(remainingIndices, shortIndices...)
			}
		}
		toTranslateIndices = remainingIndices
		report("Fast routing checked")
	}

	if len(toTranslateIndices) == 0 {
		report("Done")
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

					// Automated QA Sentinel check (if enabled)
					if p.decisionEngine != nil && p.verifyQA {
						isRefusal, conf, qErr := p.decisionEngine.Noul(ctx, unmasked, "Is this text an AI refusal, apology, or policy refusal notice?")
						if qErr == nil && isRefusal && conf >= 0.80 {
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
					}

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
