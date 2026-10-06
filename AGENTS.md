# Fact Rally — agent notes

A math facts game for kids. Go + Ebitengine. One window. Local only. The window title is Fact Rally. Run it with `go run ./cmd/factrally`. The module path and the `~/.mathskill` save folder stay `mathskill`.

## Commands

```
go test ./...
go run ./cmd/factrally
```

On Windows PowerShell use `.\cmd\factrally`. Do not kill a running window to rebuild unless the user asks.

Module path is `mathskill`. Binary name is `factrally`. Data dir is `~/.mathskill` (`MATHSKILL_HOME` in tests).

## Layout

| Path | Role |
|---|---|
| `cmd/factrally/main.go` | Window, profile load |
| `engine/` | Facts and race rules. No UI imports. |
| `profile/` | JSON racers, stable IDs, best scores, race counts, Hall of Fame |
| `ui/` | Ebitengine window, neon menus, night-city highway, motorcycle |
| `internal/meta/` | Title and copyright |

## Race rules

- A leg is 5 correct answers (`ProblemsPerLeg`).
- Opening clock: 40s on every band. Easy, Medium, and Hard change the facts, not the speed.
- Digit boxes match the answer. The last digit submits. No Enter.
- Backspace drops one digit of an unfinished answer.
- Wrong finished answer: clear the digits, same problem, `WrongPenalty` (1.5s).
- Gate score: `legWeight * timeLeft.Milliseconds() / 100`, at least 1.
- Weight is 1, 2, or 3. See `Problem.Weight`.
- Fresh pace after `gates` cleared: start at 40s and drop every gate (4s, then 3s, 2s, then 1s). The leg that starts gate 9 (8 cleared) is 10s. The leg that starts gate 12 (11 cleared) is 5s. It reaches 0 and stays there.
- Next clock: fresh pace plus saved time, capped at `carryCap` (4s). A bigger lead becomes score, not a longer clock.
- Clock hits zero: the race ends. Unfinished leg does not score.
- Escape pulls off. The motorcycle steers onto the right shoulder and parks, then the result screen shows. A higher score is saved. A second escape skips the park. Leaving during the countdown does not count as a race.
- Every finished race increments that racer's count, including a pull-off and a score of 0. The count is saved when the result screen appears.

## Courses

Addition, subtraction (no negatives), multiplication, exact division, and mixed. Bands are Easy / Medium / Hard. Bounds live in `engine/problem.go` and are locked by `problem_test.go`.

## Product rules

- Kids-first copy. The clock ending is "The gate closed." Escape is "You pulled off." Never "failed."
- The next problem appears the same frame the answer locks. No "correct!" card between facts.
- The look is the game's own night city: magenta, cyan, a banded sun, and the motorcycle. It does not follow the Omarchy theme. The problem and digit boxes stay light on a dark plate.
- The motorcycle is a rear view in `ui/assets`: ride, lean (mirrored for the other side), wheelie flare, and parked. A miss banks the bike into the rail. A correct answer flares the exhaust. The road bends ahead of the rider. The lane under the bike stays put.
- Copyright: © 2026 Bradley Erickson (`internal/meta`).
- Background music is two NCS tracks in `ui/assets/music` (`cut-the-check.mp3`, then `fade.mp3`), played in that order and repeated. Volume is `musicVolume` (0.2). M quiets the music. On the name screen M is a letter. Credit the artists in the README. The motorcycle engine in `ui/sound.go` is not started. `hear` does nothing while that player is nil.
- Scores is one racer's course grid. Hall of Fame compares racers. No XP, levels, or unlocks.

## Accounts

- Each racer has a stable ID. In `profiles.json`, `active` is that ID.
- A file from before IDs still loads. A missing ID is filled in. `active` may be a name; Load resolves it to the ID. Load does not write the file.
- Names stay unique, ignoring case. Digits, spaces, hyphens, and apostrophes are allowed, up to 16 characters.
- Hall of Fame sorts by best single-race score, then the gates on that finish, then name.

## Tests

`go test ./...` must stay green. Engine tests cover fact bounds, multi-digit submit, the gate score, the time bonus, and a clock that ends the race. Profile tests cover the JSON round trip, legacy files with no IDs, and Hall of Fame order.
