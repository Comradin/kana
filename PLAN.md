# Kana Game - Development Plan

This document outlines planned improvements and features for the Kana learning game.

## Character Sets

### Priority: High
- [ ] Add Katakana character set
- [ ] Add Dakuten characters (が、ぎ、ぐ、げ、ご, etc.)
- [ ] Add Handakuten characters (ぱ、ぴ、ぷ、ぺ、ぽ)
- [ ] Add Yoon combinations (きゃ、きゅ、きょ, しゃ、しゅ、しょ, etc.)
- [ ] Mode selector: Hiragana only, Katakana only, Mixed, Dakuten/Handakuten, Yoon

## Learning Progress System

### Priority: High

**Character Mastery Tracking**
- Track consecutive correct identifications per character
- Save progress to local file (JSON or similar)
- Color-coded character indication:
  - **Red**: Characters frequently missed (hit bottom multiple times)
  - **Yellow**: Characters with some correct streak progress
  - **Green**: Well-learned characters (high consecutive correct count)

**Anti-Spam Mechanism**
- Punish typing romaji not currently on screen
- Implementation: Increase falling speed slightly when incorrect romaji is typed
- Prevents players from randomly typing in hopes of hitting something

**Spawn Distribution**
- Replace simple RNG with a shuffled “bag” system so each enabled kana appears before repeats
- Optional weighting layer to surface characters the learner has seen less often

## Training Modes

### Priority: Medium

**Focused Practice Modes**
- Train specific characters by mastery level:
  - Red character training (slower speed for learning)
  - Yellow character training (normal speed)
  - Green character maintenance (normal speed)
- Goal: Move red → yellow → green, avoid degrading

**Character Set Practice**
- Practice specific kana groups (ka-row, sa-row, etc.)
- Combine character types (e.g., hiragana + dakuten)

## Visual Layout Redesign

### Priority: High

**Tetris-Style Layout**
- Narrow central playing field (similar to Tetris proportions)
- Right sidebar for statistics and information:
  - Correct character counts
  - Current streak
  - Characters that reached bottom (without showing romaji translation)
  - Mastery level distribution (red/yellow/green counts)
  - High score / session stats

**Visual Enhancements**
- Color-code falling kanas by mastery level (red/yellow/green)
- Show proximity to bottom with intensity or additional visual cues
- Better feedback animations for correct/incorrect answers
- Highlight which kana would match current input

## Gameplay Enhancements

### Priority: Medium
- [ ] Pause functionality
- [ ] Restart game without exiting program
- [ ] Lives system with visual representation
- [ ] Combo/streak system with bonus points
- [ ] Show correct answer briefly when kana is missed
- [ ] Progressive difficulty (speed/spawn rate increases)

### Priority: Low
- [ ] Power-ups (slow time, clear screen, reveal romaji)
- [ ] Sound effects via terminal bell
- [ ] Particle effects for matched characters

## Difficulty & Progression

### Priority: Medium
- Difficulty presets (beginner/intermediate/advanced)
- Dynamic difficulty adjustment based on performance
- Spawn rate increases over time
- Speed variation based on character mastery level

## Statistics & Persistence

### Priority: High
- [ ] Save high scores locally
- [ ] Track per-character accuracy and mastery progress
- [ ] Session statistics (accuracy, characters learned, etc.)
- [ ] Historical data (daily/weekly progress)

### Priority: Low
- [ ] Export learning statistics
- [ ] Graphs/charts of progress over time

## Code Refactoring

### Priority: Low (only if project grows significantly)
- Split into multiple files:
  - `game.go`: Core game logic
  - `kana.go`: Character set definitions and mastery tracking
  - `ui.go`: Rendering and layout
  - `stats.go`: Statistics and persistence
  - `config.go`: Configuration loading
- Move character sets to JSON/TOML config files
- Add unit tests for game logic

## Implementation Notes

**File Structure for Persistence**
```json
{
  "characters": {
    "あ": {
      "correct_streak": 5,
      "times_missed": 2,
      "total_attempts": 15,
      "mastery_level": "yellow"
    }
  },
  "high_score": 450,
  "total_games": 23
}
```

**Speed Penalty Mechanism**
- Base speed: 0.15-0.25 per tick
- Penalty for incorrect input: +0.02 to all falling kanas
- Penalty decay over time or after correct answer

**Mastery Level Thresholds**
- Red: < 3 consecutive correct, or missed > 3 times
- Yellow: 3-10 consecutive correct
- Green: > 10 consecutive correct
- Degrade level if missed again

## Priority Summary

1. **Phase 1**: Character sets expansion + Visual layout redesign + Anti-spam mechanism
2. **Phase 2**: Learning progress system + Mastery tracking + Persistence
3. **Phase 3**: Training modes + Focused practice
4. **Phase 4**: Additional gameplay enhancements + Statistics
5. **Phase 5**: Code refactoring (if needed)
