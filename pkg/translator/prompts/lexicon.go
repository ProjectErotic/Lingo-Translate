package prompts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AnatomyTerms groups terms for male and female anatomy
type AnatomyTerms struct {
	Male   []string `json:"male"`
	Female []string `json:"female"`
}

// DirtyTalkTerms groups dialogue cues for submissive and dominant roles
type DirtyTalkTerms struct {
	Submissive []string `json:"submissive"`
	Dominant   []string `json:"dominant"`
}

// NSFWLexicon contains categorized erotic and sensual vocabulary for 18+ localization
type NSFWLexicon struct {
	Sensations []string       `json:"sensations"`
	Actions    []string       `json:"actions"`
	Anatomy    AnatomyTerms   `json:"anatomy"`
	Fluids     []string       `json:"fluids"`
	Moans      []string       `json:"moans"`
	DirtyTalk  DirtyTalkTerms `json:"dirty_talk"`
	Custom     []string       `json:"custom,omitempty"`
}

// NormalizeLangCode standardizes language identifiers (e.g. "Thai", "th-TH" -> "th")
func NormalizeLangCode(lang string) string {
	l := strings.ToLower(strings.TrimSpace(lang))
	switch {
	case strings.HasPrefix(l, "th"):
		return "th"
	case strings.HasPrefix(l, "en"):
		return "en"
	case strings.HasPrefix(l, "zh") || strings.Contains(l, "chinese"):
		return "zh"
	case strings.HasPrefix(l, "ja") || strings.Contains(l, "japanese"):
		return "ja"
	case strings.HasPrefix(l, "ko") || strings.Contains(l, "korean"):
		return "ko"
	case strings.HasPrefix(l, "es") || strings.Contains(l, "spanish"):
		return "es"
	case strings.HasPrefix(l, "fr") || strings.Contains(l, "french"):
		return "fr"
	case strings.HasPrefix(l, "de") || strings.Contains(l, "german"):
		return "de"
	case strings.HasPrefix(l, "ru") || strings.Contains(l, "russian"):
		return "ru"
	default:
		return l
	}
}

// DefaultNSFWLexicon returns the rich built-in erotic lexicon for the target language
func DefaultNSFWLexicon(targetLang string) *NSFWLexicon {
	code := NormalizeLangCode(targetLang)
	switch code {
	case "zh":
		return defaultChineseNSFWLexicon()
	case "en":
		return defaultEnglishNSFWLexicon()
	case "th":
		return defaultThaiNSFWLexicon()
	default:
		// Default to English base for international languages
		return defaultEnglishNSFWLexicon()
	}
}

func defaultThaiNSFWLexicon() *NSFWLexicon {
	return &NSFWLexicon{
		Sensations: []string{
			"เสียวซ่าน", "สยิวลึก", "เสียวสะท้าน", "วาบหวิว", "กระตุกเกร็ง",
			"แน่นคับ", "คับแน่น", "แฉะเยิ้ม", "ฉ่ำเยิ้ม", "มิดด้าม",
			"เสียวแทบขาดใจ", "ฟินจนตาค้าง", "ร้อนผ่าว", "ซาบซ่าน",
			"เสียวจนทนไม่ไหว", "สติจะหลุด",
		},
		Actions: []string{
			"กระแทกกระทั้น", "ซอยยับ", "ขย่ม", "รัวสะโพก", "สอดลึก",
			"รูดรั้ง", "เลียลาก", "ดูดดุน", "บดคลึง", "ตอกเสาเข็ม",
			"ซอยถี่", "ยัดเยียด", "กลืนกิน", "ตอดรัด",
		},
		Anatomy: AnatomyTerms{
			Male: []string{
				"แท่งร้อน", "ดุ้น", "แก่นกาย", "ท่อนเอ็น", "หัวเห็ด",
				"ควย", "ลำกาย", "ความใหญ่โต",
			},
			Female: []string{
				"ร่องสวาท", "กลีบกุหลาบ", "ช่องทางรัก", "ติ่งเสียว", "ยอดอก",
				"รูรัก", "หี", "กลีบเนื้อ", "ปากมดลูก", "ถ้ำสวาท",
			},
		},
		Fluids: []string{
			"น้ำรัก", "น้ำกาม", "น้ำขาวขุ่น", "น้ำเงี่ยน", "แตกใน",
			"พุ่งทะลัก", "ฉีดพ่น", "เยิ้มทะลัก", "หลั่งเปรอะ", "ไหลเยิ้ม",
		},
		Moans: []string{
			"อ๊ะ...", "อ๊างงง...", "อึก...", "อื้อออ...", "ฮ้าาา...",
			"ซี๊ดดด...", "จ๊วบ...", "แจ๊ะ...", "พั่บๆๆ!",
		},
		DirtyTalk: DirtyTalkTerms{
			Submissive: []string{
				"ไม่ไหวแล้ว...", "จะเสร็จแล้ว...", "ขอแรงๆ อีก...",
				"ตรงนั้นเสียวจนจะตายแล้ว...", "ลึกเกินไปแล้ว...", "แตกข้างในเลยนะ...",
				"ฉีดน้ำรักเข้ามาให้หมดเลย...",
			},
			Dominant: []string{
				"รัดแน่นฉิบหาย...", "เงี่ยนจนแฉะไปหมดแล้ว", "จะเย็ดให้ลืมทางกลับบ้านเลย",
				"ครางออกมาดังๆ", "ตอดแน่นขนาดนี้ กะจะไม่ให้ปล่อยรอดไปเลยใช่ไหม",
				"ยอมแพ้แล้วร้องครางซะ",
			},
		},
	}
}

func defaultEnglishNSFWLexicon() *NSFWLexicon {
	return &NSFWLexicon{
		Sensations: []string{
			"throbbing", "aching with pleasure", "melting", "dripping wet",
			"overwhelmed with ecstasy", "mind-numbing bliss", "shivering",
			"tightly clenched", "electric tingling", "heat radiating",
			"trembling with lust", "verge of snapping",
		},
		Actions: []string{
			"thrusting deep", "pounding", "grinding hips", "stroking relentlessly",
			"teasing the edge", "licking greedily", "sucking eagerly", "bottoming out",
			"clamping down tight", "relentless friction", "slamming together",
		},
		Anatomy: AnatomyTerms{
			Male: []string{
				"hard cock", "shaft", "tip", "thick rod", "throbbing member",
				"length", "swollen head",
			},
			Female: []string{
				"pussy", "tight folds", "clit", "sweet entrance", "womb",
				"dripping slit", "soft walls", "g-spot", "sensitive nub",
			},
		},
		Fluids: []string{
			"cum", "pre-cum", "love juices", "creampie", "milking",
			"bursting inside", "gushing", "warm spill", "spurting",
		},
		Moans: []string{
			"ahhh...", "nghh...", "haah...", "f-fuck...", "oh god...",
			"aahhn...", "mmph...", "h-harder...!",
		},
		DirtyTalk: DirtyTalkTerms{
			Submissive: []string{
				"I can't take it anymore...", "I'm gonna cum...", "Please, harder...",
				"Fill me up inside...", "It's too deep...", "Don't stop...",
				"Make me lose my mind...",
			},
			Dominant: []string{
				"You're clamping down so hard...", "So wet and needy...",
				"Take every inch of it...", "Beg for it...", "Cum for me...",
				"You love this cock, don't you?",
			},
		},
	}
}

func defaultChineseNSFWLexicon() *NSFWLexicon {
	return &NSFWLexicon{
		Sensations: []string{
			"酥麻", "燥热", "紧致湿润", "阵阵痉挛", "快感直冲脑门",
			"爽到失神", "蜜液横流", "被填满的充实感", "快感如触电般蔓延",
			"快要疯掉了", "爽到极致", "难以言喻的快感",
		},
		Actions: []string{
			"抽插", "猛烈挺进", "深顶", "研磨", "套弄",
			"猛烈撞击", "舔舐", "紧紧绞住", "大肆索取",
			"疯狂律动", "直抵花心", "狠狠贯穿",
		},
		Anatomy: AnatomyTerms{
			Male: []string{
				"肉棒", "阳具", "龟头", "粗长", "滚烫的巨物",
				"炽热的硬挺", "棒身",
			},
			Female: []string{
				"小穴", "蜜穴", "花径", "阴蒂", "花瓣",
				"子宫口", "敏感点", "深处花心", "湿热的蜜道",
			},
		},
		Fluids: []string{
			"爱液", "精液", "浓精", "内射", "白浊",
			"喷涌而出", "汁水四溢", "倾泻而出", "泛滥成灾",
		},
		Moans: []string{
			"啊啊…", "哈啊…", "呜…", "嗯嗯…", "呀啊…",
			"唔咕…", "啊呜…", "快… 快点…！",
		},
		DirtyTalk: DirtyTalkTerms{
			Submissive: []string{
				"不行了… 要去了…", "求求你… 再深一点…", "要坏掉了…",
				"全都射在里面吧…", "好舒服… 顶到最里面了…", "受不了了…",
			},
			Dominant: []string{
				"里面咬得这么紧…", "湿成这样了还嘴硬", "叫得再大声一点",
				"全都射给你… 怀上我的孩子吧", "爽不爽？自己动动看",
			},
		},
	}
}

func mergeSlice(base, custom []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range base {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range custom {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// GetLexiconSearchPaths returns candidate file paths for nsfw_lexicon for the given language
func GetLexiconSearchPaths(targetLang string) []string {
	code := NormalizeLangCode(targetLang)
	var filenames []string
	if code != "" {
		filenames = append(filenames, fmt.Sprintf("nsfw_lexicon_%s.json", code))
	}
	filenames = append(filenames, "nsfw_lexicon.json")

	var dirs []string
	if homeDir, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(homeDir, ".lingo"))
		dirs = append(dirs, filepath.Join(homeDir, ".lingo", "templates"))
		dirs = append(dirs, filepath.Join(homeDir, ".nst")) // legacy
	}
	if cfgDir, err := os.UserConfigDir(); err == nil {
		dirs = append(dirs, filepath.Join(cfgDir, "lingo"))
		dirs = append(dirs, filepath.Join(cfgDir, "lingo", "templates"))
	}
	dirs = append(dirs, "templates")
	dirs = append(dirs, ".")

	var paths []string
	for _, dir := range dirs {
		for _, fn := range filenames {
			paths = append(paths, filepath.Join(dir, fn))
		}
	}
	return paths
}

// LoadNSFWLexicon loads custom NSFW lexicon from disk for targetLang and merges with default
func LoadNSFWLexicon(targetLang string) *NSFWLexicon {
	lex := DefaultNSFWLexicon(targetLang)
	foundCustom := false

	for _, p := range GetLexiconSearchPaths(targetLang) {
		if data, err := os.ReadFile(p); err == nil {
			var custom NSFWLexicon
			if err := json.Unmarshal(data, &custom); err == nil {
				foundCustom = true
				lex.Sensations = mergeSlice(lex.Sensations, custom.Sensations)
				lex.Actions = mergeSlice(lex.Actions, custom.Actions)
				lex.Anatomy.Male = mergeSlice(lex.Anatomy.Male, custom.Anatomy.Male)
				lex.Anatomy.Female = mergeSlice(lex.Anatomy.Female, custom.Anatomy.Female)
				lex.Fluids = mergeSlice(lex.Fluids, custom.Fluids)
				lex.Moans = mergeSlice(lex.Moans, custom.Moans)
				lex.DirtyTalk.Submissive = mergeSlice(lex.DirtyTalk.Submissive, custom.DirtyTalk.Submissive)
				lex.DirtyTalk.Dominant = mergeSlice(lex.DirtyTalk.Dominant, custom.DirtyTalk.Dominant)
				lex.Custom = mergeSlice(lex.Custom, custom.Custom)
				return lex
			}
		}
	}

	// Auto-provision starter file in ~/.lingo/ for user visibility if none exists
	if !foundCustom {
		go func() {
			_, _ = EnsureUserLexiconFile(targetLang)
		}()
	}

	return lex
}

// EnsureUserLexiconFile creates a starter nsfw_lexicon.json in ~/.lingo/ if it doesn't already exist
func EnsureUserLexiconFile(lang string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	lingoDir := filepath.Join(homeDir, ".lingo")
	_ = os.MkdirAll(lingoDir, 0755)

	code := NormalizeLangCode(lang)
	var filename string
	if code != "" && code != "th" {
		filename = fmt.Sprintf("nsfw_lexicon_%s.json", code)
	} else {
		filename = "nsfw_lexicon.json"
	}

	targetPath := filepath.Join(lingoDir, filename)
	if _, err := os.Stat(targetPath); err == nil {
		return targetPath, nil // already exists, preserve user edits!
	}

	def := DefaultNSFWLexicon(lang)
	data, err := json.MarshalIndent(def, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(targetPath, data, 0644); err != nil {
		return "", err
	}
	return targetPath, nil
}

// EnsureAllDefaultLexicons generates starter files for th, en, and zh in ~/.lingo/
func EnsureAllDefaultLexicons() ([]string, error) {
	var paths []string
	langs := []string{"th", "en", "zh"}
	for _, l := range langs {
		p, err := EnsureUserLexiconFile(l)
		if err != nil {
			return paths, err
		}
		paths = append(paths, p)
	}
	return paths, nil
}

// FormatNSFWLexiconForPrompt creates a structured prompt text describing the erotic vocabulary palette
func FormatNSFWLexiconForPrompt(lex *NSFWLexicon, targetLang string) string {
	if lex == nil {
		lex = DefaultNSFWLexicon(targetLang)
	}

	code := NormalizeLangCode(targetLang)
	var sb strings.Builder

	switch code {
	case "th":
		sb.WriteString("EROTIC THESAURUS & VOCABULARY PALETTE (Thai 18+ Localization Reference):\n")
		sb.WriteString("The following terms and expressions are authentic adult expressions across various intensities. You have full creative autonomy to choose and adapt them to match the character persona, emotional tone, and scene heat:\n")
		if len(lex.Sensations) > 0 {
			sb.WriteString(fmt.Sprintf("- Sensations & Arousal (สัมผัส/ความเสียว): %s\n", strings.Join(lex.Sensations, ", ")))
		}
		if len(lex.Actions) > 0 {
			sb.WriteString(fmt.Sprintf("- Actions & Momentum (ลีลา/การกระทำ): %s\n", strings.Join(lex.Actions, ", ")))
		}
		if len(lex.Anatomy.Male) > 0 || len(lex.Anatomy.Female) > 0 {
			sb.WriteString("- Anatomy (สรีระ):\n")
			if len(lex.Anatomy.Male) > 0 {
				sb.WriteString(fmt.Sprintf("  * Male: %s\n", strings.Join(lex.Anatomy.Male, ", ")))
			}
			if len(lex.Anatomy.Female) > 0 {
				sb.WriteString(fmt.Sprintf("  * Female: %s\n", strings.Join(lex.Anatomy.Female, ", ")))
			}
		}
		if len(lex.Fluids) > 0 {
			sb.WriteString(fmt.Sprintf("- Fluids & Ejaculation (ของเหลว/การหลั่ง): %s\n", strings.Join(lex.Fluids, ", ")))
		}
		if len(lex.Moans) > 0 {
			sb.WriteString(fmt.Sprintf("- Moans & Gasps (เสียงคราง/จังหวะกระเส่า): %s\n", strings.Join(lex.Moans, ", ")))
		}
		if len(lex.DirtyTalk.Submissive) > 0 || len(lex.DirtyTalk.Dominant) > 0 {
			sb.WriteString("- Dirty Talk & Dialogue Cues:\n")
			if len(lex.DirtyTalk.Submissive) > 0 {
				sb.WriteString(fmt.Sprintf("  * Submissive / Aroused Plea: %s\n", strings.Join(lex.DirtyTalk.Submissive, ", ")))
			}
			if len(lex.DirtyTalk.Dominant) > 0 {
				sb.WriteString(fmt.Sprintf("  * Dominant / Dirty Talk: %s\n", strings.Join(lex.DirtyTalk.Dominant, ", ")))
			}
		}
	case "zh":
		sb.WriteString("EROTIC THESAURUS & VOCABULARY PALETTE (Chinese 18+ Localization Reference):\n")
		sb.WriteString("The following terms and expressions are authentic adult expressions across various intensities. You have full creative autonomy to choose and adapt them to match the character persona, emotional tone, and scene heat:\n")
		if len(lex.Sensations) > 0 {
			sb.WriteString(fmt.Sprintf("- Sensations & Arousal (感官/快感描写): %s\n", strings.Join(lex.Sensations, ", ")))
		}
		if len(lex.Actions) > 0 {
			sb.WriteString(fmt.Sprintf("- Actions & Momentum (动作/抽插律动): %s\n", strings.Join(lex.Actions, ", ")))
		}
		if len(lex.Anatomy.Male) > 0 || len(lex.Anatomy.Female) > 0 {
			sb.WriteString("- Anatomy (身体部位):\n")
			if len(lex.Anatomy.Male) > 0 {
				sb.WriteString(fmt.Sprintf("  * Male: %s\n", strings.Join(lex.Anatomy.Male, ", ")))
			}
			if len(lex.Anatomy.Female) > 0 {
				sb.WriteString(fmt.Sprintf("  * Female: %s\n", strings.Join(lex.Anatomy.Female, ", ")))
			}
		}
		if len(lex.Fluids) > 0 {
			sb.WriteString(fmt.Sprintf("- Fluids & Ejaculation (体液/绝顶内射): %s\n", strings.Join(lex.Fluids, ", ")))
		}
		if len(lex.Moans) > 0 {
			sb.WriteString(fmt.Sprintf("- Moans & Gasps (娇喘/呻吟): %s\n", strings.Join(lex.Moans, ", ")))
		}
		if len(lex.DirtyTalk.Submissive) > 0 || len(lex.DirtyTalk.Dominant) > 0 {
			sb.WriteString("- Dirty Talk & Dialogue Cues:\n")
			if len(lex.DirtyTalk.Submissive) > 0 {
				sb.WriteString(fmt.Sprintf("  * Submissive / Aroused Plea: %s\n", strings.Join(lex.DirtyTalk.Submissive, ", ")))
			}
			if len(lex.DirtyTalk.Dominant) > 0 {
				sb.WriteString(fmt.Sprintf("  * Dominant / Dirty Talk: %s\n", strings.Join(lex.DirtyTalk.Dominant, ", ")))
			}
		}
	case "en":
		sb.WriteString("EROTIC THESAURUS & VOCABULARY PALETTE (English 18+ Localization Reference):\n")
		sb.WriteString("The following terms and expressions are authentic adult expressions across various intensities. You have full creative autonomy to choose and adapt them to match the character persona, emotional tone, and scene heat:\n")
		if len(lex.Sensations) > 0 {
			sb.WriteString(fmt.Sprintf("- Sensations & Arousal: %s\n", strings.Join(lex.Sensations, ", ")))
		}
		if len(lex.Actions) > 0 {
			sb.WriteString(fmt.Sprintf("- Actions & Momentum: %s\n", strings.Join(lex.Actions, ", ")))
		}
		if len(lex.Anatomy.Male) > 0 || len(lex.Anatomy.Female) > 0 {
			sb.WriteString("- Anatomy:\n")
			if len(lex.Anatomy.Male) > 0 {
				sb.WriteString(fmt.Sprintf("  * Male: %s\n", strings.Join(lex.Anatomy.Male, ", ")))
			}
			if len(lex.Anatomy.Female) > 0 {
				sb.WriteString(fmt.Sprintf("  * Female: %s\n", strings.Join(lex.Anatomy.Female, ", ")))
			}
		}
		if len(lex.Fluids) > 0 {
			sb.WriteString(fmt.Sprintf("- Fluids & Ejaculation: %s\n", strings.Join(lex.Fluids, ", ")))
		}
		if len(lex.Moans) > 0 {
			sb.WriteString(fmt.Sprintf("- Moans & Gasps: %s\n", strings.Join(lex.Moans, ", ")))
		}
		if len(lex.DirtyTalk.Submissive) > 0 || len(lex.DirtyTalk.Dominant) > 0 {
			sb.WriteString("- Dirty Talk & Dialogue Cues:\n")
			if len(lex.DirtyTalk.Submissive) > 0 {
				sb.WriteString(fmt.Sprintf("  * Submissive / Aroused Plea: %s\n", strings.Join(lex.DirtyTalk.Submissive, ", ")))
			}
			if len(lex.DirtyTalk.Dominant) > 0 {
				sb.WriteString(fmt.Sprintf("  * Dominant / Dirty Talk: %s\n", strings.Join(lex.DirtyTalk.Dominant, ", ")))
			}
		}
	default:
		langLabel := targetLang
		if langLabel == "" {
			langLabel = "Universal"
		}
		sb.WriteString(fmt.Sprintf("EROTIC THESAURUS & VOCABULARY PALETTE (%s 18+ Localization Reference):\n", langLabel))
		sb.WriteString("The following terms and expressions are authentic adult expressions across various intensities. You have full creative autonomy to choose and adapt them to match the character persona, emotional tone, and scene heat:\n")
		if len(lex.Sensations) > 0 {
			sb.WriteString(fmt.Sprintf("- Sensations & Arousal: %s\n", strings.Join(lex.Sensations, ", ")))
		}
		if len(lex.Actions) > 0 {
			sb.WriteString(fmt.Sprintf("- Actions & Momentum: %s\n", strings.Join(lex.Actions, ", ")))
		}
		if len(lex.Anatomy.Male) > 0 || len(lex.Anatomy.Female) > 0 {
			sb.WriteString("- Anatomy:\n")
			if len(lex.Anatomy.Male) > 0 {
				sb.WriteString(fmt.Sprintf("  * Male: %s\n", strings.Join(lex.Anatomy.Male, ", ")))
			}
			if len(lex.Anatomy.Female) > 0 {
				sb.WriteString(fmt.Sprintf("  * Female: %s\n", strings.Join(lex.Anatomy.Female, ", ")))
			}
		}
		if len(lex.Fluids) > 0 {
			sb.WriteString(fmt.Sprintf("- Fluids & Ejaculation: %s\n", strings.Join(lex.Fluids, ", ")))
		}
		if len(lex.Moans) > 0 {
			sb.WriteString(fmt.Sprintf("- Moans & Gasps: %s\n", strings.Join(lex.Moans, ", ")))
		}
		if len(lex.DirtyTalk.Submissive) > 0 || len(lex.DirtyTalk.Dominant) > 0 {
			sb.WriteString("- Dirty Talk & Dialogue Cues:\n")
			if len(lex.DirtyTalk.Submissive) > 0 {
				sb.WriteString(fmt.Sprintf("  * Submissive / Aroused Plea: %s\n", strings.Join(lex.DirtyTalk.Submissive, ", ")))
			}
			if len(lex.DirtyTalk.Dominant) > 0 {
				sb.WriteString(fmt.Sprintf("  * Dominant / Dirty Talk: %s\n", strings.Join(lex.DirtyTalk.Dominant, ", ")))
			}
		}
	}

	if len(lex.Custom) > 0 {
		sb.WriteString(fmt.Sprintf("- Custom Erotic Lexicon: %s\n", strings.Join(lex.Custom, ", ")))
	}

	return sb.String()
}
