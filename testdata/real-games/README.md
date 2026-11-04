# Real-game fixtures

Integration test data from REAL games in the Chanomhub catalog.
Used by `pkg/parser/realgames_integration_test.go` (`go test -tags integration ./pkg/parser/...`).

| dir | engine | game (slug) | what's inside |
|---|---|---|---|
| `rpgm/` | rpgm | Noko's lucky rock-paper-scissors! (`nokos-lucky-rock-paper-scissors-HJ1045`) | full `data/*.json` from the shipped game (the first Lingo-published mod) |
| `renpy/` | renpy | The Blood Diary (`the-blood-diary-HJ933`) | 57 `.rpy` scripts extracted from the shipped `archive.rpa` (RPA-3.0, Russian source) |
| `unity/` | unity | Bunny Quota Struggles (`bunny-quota-struggles-HJ1168`) | real build markers (`*_Data/`, UnityPlayer.dll stub) + StreamingAssets/settings.json — shipped builds keep text in serialized assets, so Detect classifies and Extract legitimately finds nothing |
| `godot/` | godot | PvZ Sex Lawn DEMO (Godot 4, `GDPC` pck) | synthetic real-format fixture: `project.godot` + format-v4 `.pck` carrying two compiled `.gdc` scripts (string constants in Spanish + engine paths) and a `.scn` with an encoded string variant — exercises the full pck→gdc extract pipeline. The original 1MB truncated real pck is not parseable (v4 directories live at the END of the file, so truncation kills them) |
| `libgdx/` | libgdx | Tales of Androgyny (`tales-of-androgyny-HJ883`) | real `encounters.json` (11.7MB, 4191 encounters), `encounter-trees.json`, `strings.properties` (424KB), `battleui.json` from the shipped TalesOfAndrogyny.jar, repacked as `TalesOfAndrogyny-realfiles.jar` with the libGDX marker class |

Truncated/stubbed files (unity dll, godot pck/exe) are fine: Detect only stats
names/paths. Don't replace the libgdx JSONs with mock data — their size and
real-world mess (control codes, empty strings, markup) is the point.
