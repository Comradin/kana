# Kana

A typing game for learning Japanese kana and vocabulary: tiles fall, the learner types the romaji.

## Word lists

**Word list**:
A file of words the game keeps in its words folder, next to the bundled JLPT list; named after the file.
_Avoid_: vocabulary file, word file

**Source file**:
The learner's own file somewhere on their computer, before it becomes a word list.
_Avoid_: original, upload

**Import**:
Turning a source file into a word list: the game checks it and keeps its own copy.
_Avoid_: upload, add list

**Skipped row**:
A line of a word list the game could not read, with the reason why; the rest of the list still plays.
_Avoid_: problem, error, invalid row

**Reading**:
The kana a learner types a word by; progress belongs to it, whatever list the word comes from.
_Avoid_: key, plain kana

**Placeholder**:
A wave dash (～) in a word list entry standing for something the learner fills in, such as a name in ～さん; shown, not typed.
_Avoid_: wildcard, blank

**Optional part**:
A part of a word list entry in brackets that may be left out, as in （どうぞ）よろしく; both the long and the short form count.
_Avoid_: optional word, bracket

**Variant**:
One of several writings of the same word in one entry, separated by //, as in あめりか // アメリカ.
_Avoid_: alternative, synonym

**Delete a word list**:
Removing a word list from the game for good; the learner's progress on its words stays.
_Avoid_: remove, uninstall

**Rename a word list**:
Giving a word list a new name; its lesson choice and its place as the source of new words go with it.
_Avoid_: move

## Practice

**Learning level**:
Progress from 0 (new) to 5 (learned) of a kana, or of a word's reading or meaning; a hit raises it by one, a mistake lowers it by two.
_Avoid_: score, mastery

**Reading level**:
A word's learning level for typing its reading.
_Avoid_: word level, kana level

**Meaning level**:
A word's learning level for typing its meaning; it starts at 0 and grows only from meanings typed.
_Avoid_: translation level, vocabulary level

**Learned**:
At learning level 5.
_Avoid_: known, secure, mastered

**Slipped**:
Made a mistake since it was last learned; shown with a red frame until learned again.
_Avoid_: flagged, failed

**Weak item**:
A kana or word that is learning (level 1–4) or slipped.
_Avoid_: problem item, difficult item

**Waiting word**:
An introduced word that cannot fall yet because one of its kana is not learned.
_Avoid_: paused word, blocked word

**Words-only mode**:
Practice in which only words fall, no single kana.
_Avoid_: vocabulary mode, word mode

**Meaning mode**:
Practice in which a word tile asks for the word's meaning instead of its reading; kana tiles still ask for the romaji.
_Avoid_: translation mode, vocabulary mode

**Reading practice**:
Practice in which any word whose kana are all learned may fall, introduced or not, to train reading speed; a word read counts for its kana, not for the word.
_Avoid_: reading mode, speed mode

**Lesson filter**:
The lessons of a word list chosen for practice; they decide which words fall and where new word packages come from.
_Avoid_: lesson selection, chapter
