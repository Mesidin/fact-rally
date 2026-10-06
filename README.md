# Fact Rally

Pick addition, subtraction, multiplication, division, or all four. Five correct answers reach a gate. The time you have left becomes points and extra time for the next gate. When the clock hits zero, the gate closes.

The problem is huge. Type the answer into the boxes. The last digit locks it in, and the next problem is already up. Backspace fixes a digit before that.

Built in Go with [Ebitengine](https://ebitengine.org/). Local only: each racer stays on this computer. No network.

© 2026 Bradley Erickson

## Play

From this folder:

```
go run .\cmd\factrally
```

On macOS or Linux:

```
go run ./cmd/factrally
```

To put it in the Dock, build the app on a Mac from this folder:

```
sh scripts/macos-app.sh
cp -R "Fact Rally.app" /Applications/
```

Drag Fact Rally onto the Dock. The first time, macOS may ask you to right-click the app and choose Open. `make dist` also writes Intel and Apple Silicon binaries in `dist/`.

Type a racer name, then Race. More than one racer can share the computer. Scores is that racer's best on each course. Hall of Fame ranks every racer by their best single race. Arrow keys and Enter work on the menus, and so does the mouse. During a race, use the number row or the numpad.

| Key | Action |
|---|---|
| `0`–`9` | Type a digit. The last box submits. |
| `backspace` | Undo a digit |
| `esc` | Pull off to the right and park. The score stands. |
| `M` | Quiet the music, or bring it back. On the name screen, M is a letter. |
| `ctrl+c` | Quit |

Maximize the window if you want the problems bigger. The layout scales with the window.

## Courses

Each course has Easy, Medium, and Hard.

| Course | Easy | Medium | Hard |
|---|---|---|---|
| Addition | Sums up to 10 | Sums from 11 to 20 | Two-digit plus two-digit |
| Subtraction | Within 10 | Within 20 | Two-digit minus |
| Multiplication | Tables 0–5 | Tables 0–10 | 11s, 12s, and two-digit × one-digit |
| Division | Exact, tables 0–5 | Exact, tables 0–10 | Exact, bigger tables and two-digit answers |
| Mixed | All four, easy | All four, medium | All four, hard |

Subtraction never asks for a negative. Division is always exact.

## The race

A leg is 5 problems. The clock starts at 40 seconds on every band. Easy, Medium, and Hard change the problems, not the speed.

A finished wrong answer clears the boxes, keeps the same problem, and costs 1.5 seconds. Run the clock out and the race is over.

Reach the gate and the score grows by the difficulty of those five problems times the seconds you had left (10 points per difficulty per second). Up to 4 saved seconds are added to the next clock. The fresh time drops every gate, by 4 seconds at first and then by less, so the leg that starts gate 9 is 10 seconds and the leg that starts gate 12 is 5 seconds (about one second a problem). After that it keeps dropping until it is zero.

Profiles and best scores are stored in `~/.mathskill/profiles.json`. Override the folder with `MATHSKILL_HOME`. A finished race counts toward that racer's total, including a pull-off. A higher score replaces the best for that course.

## Look

The highway is a night city: a banded sun, neon lane lines, and a motorcycle seen from behind. Pixel palms, lamp posts, and a billboard line the road. The billboard's line changes with the gate, starting with "Math rocks!" The rider is dark, with a magenta stripe and cyan trim. The road bends ahead of the rider. The gate stays hidden on the first problem of a leg, shows up small on the second, and jumps closer with each answer after that. Menus use that same picture. The palette belongs to the game. A correct answer flares the exhaust. A wrong answer banks the bike into a rail. Esc steers off the right shoulder, the kickstand comes down, and then the score comes up. The problem stays large and light so it can be read over the sky.

## Music

Two songs play in the background, one after the other, then they repeat.

- Cut The Check by Spitfya and Desembra
- fade by dashie

Music provided by [NCS](https://ncs.io/).

Press M to quiet the music. The motorcycle engine is off for now. The synth is still in the program so a better one can come back.

## Requirements

Go 1.27 or later. Ebitengine is pure Go on the desktop, so you do not need a C compiler.

## Develop

```
go test ./...
go run .\cmd\factrally
```

`make test` and `make run` do the same where `make` is installed.

See [AGENTS.md](AGENTS.md) if you are picking the project up later.
