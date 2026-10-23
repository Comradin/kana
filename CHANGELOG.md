# Changelog

## [0.6.0](https://github.com/Comradin/kana/compare/v0.5.0...v0.6.0) (2026-10-23)


### New

* accept particle readings in phrases and o for を ([9bb7110](https://github.com/Comradin/kana/commit/9bb71103b288c4d6f1aa0ef35c80b17646629c73))
* **fyne:** accept every reading of words with optional parts, variants ([5c0edfc](https://github.com/Comradin/kana/commit/5c0edfca75584ce3c31683f09793f09cc36851bc))
* **fyne:** import word lists dropped on the window ([7600802](https://github.com/Comradin/kana/commit/760080248b820fcf9eef06204b64f63ab810d43a))
* **fyne:** manage word lists from the settings ([f2191ab](https://github.com/Comradin/kana/commit/f2191abd9f844ec99f8d0ae315c57af697d3bf4f))
* **fyne:** reload, delete and rename word lists in the game ([9149648](https://github.com/Comradin/kana/commit/9149648c712824308ef6a2d892d3145e59d7fa24))
* **fyne:** scroll long lists of skipped rows ([b3c9f59](https://github.com/Comradin/kana/commit/b3c9f591647f1fbd57e736dc7abae68b8078104d))
* **fyne:** show words as their word list writes them ([44f2b88](https://github.com/Comradin/kana/commit/44f2b88caceab52a6541b4d7f0527edeafa75ba7))
* **listcheck:** add a command that lists every skipped row of a list ([f14c3e3](https://github.com/Comradin/kana/commit/f14c3e353074756780dada5aa901a51c02e2466c))
* **listcheck:** print shown forms and other readings with -words ([b8e8dac](https://github.com/Comradin/kana/commit/b8e8dacc21f1a4cfc7fd3ad5dbb0c77bb051acc9))
* **store:** move a lesson filter to a renamed list ([b18f8dd](https://github.com/Comradin/kana/commit/b18f8dd30655958b9ec4af763de83b33a9ddf6de))
* **vocab:** delete and rename a word list's file ([3d1ccd1](https://github.com/Comradin/kana/commit/3d1ccd1cbb499af5af54918ee5ad1ab76ece4869))
* **vocab:** keep 100 skipped-row messages, all with InspectAll ([c6c4adf](https://github.com/Comradin/kana/commit/c6c4adff1b483a77d1836a05218bd7c70cc89fe1))
* **vocab:** merge shown forms and accepted readings per entry ([ee5a016](https://github.com/Comradin/kana/commit/ee5a01666265065126c9f32e3a380d51f1340734))
* **vocab:** parse placeholders, optional parts and variants ([e4ccba7](https://github.com/Comradin/kana/commit/e4ccba7a95ec3a3acd234e95bb9e0a12413d4db7))
* **vocab:** read the kana notation in CSV and JSON word lists ([6909a5b](https://github.com/Comradin/kana/commit/6909a5be4cd1556bb6fdb957d83e3968457d83d0))


### Improvements and fixes

* **fyne:** keep dialogs under a dropped file's preview, check names ([e3224a0](https://github.com/Comradin/kana/commit/e3224a0a2f9114fc30f64c3d62481301fde7d1ec))
* list every skipped row, keep rows whose short form is no word ([2184c73](https://github.com/Comradin/kana/commit/2184c7346924c3a0ff0e8b437f542b4e251ac75a))

## [0.5.0](https://github.com/Comradin/kana/compare/v0.4.1...v0.5.0) (2026-10-20)

**Practise your course's vocabulary.** Four new settings help you drill exactly the words you are working on.

- **Words only** (*Settings → Words → Tiles*): only words fall, no single kana. Every word you were introduced to can fall, even if you haven't learned all its kana yet – only its script needs to be switched on. Your kana progress waits meanwhile (no new rows, no katakana offer) and goes on when you switch back. If no word can fall, kana fall instead and the status panel says why.
- **Choose lessons** (*Settings → Words*): for your own word list, tick the lessons you want to practise. Only their words fall, and new word packages come from the earliest chosen lesson. Each list remembers its choice.
- **Meanings in** (*Settings → Words*): pick the language meanings are shown and accepted in – Deutsch, English, Español, Français and more, whatever your word lists contain. English answers always count too.
- **Focus on weak items** (*Settings → Game*): half of the tiles come from what you are still learning or recently got wrong; the ones you got wrong come twice as often.

### Commits

#### New

* **fyne:** add tiles, lessons, meaning language and focus settings ([5b79f8b](https://github.com/Comradin/kana/commit/5b79f8b97752b2b93094067dccd6b24486152c96))
* **fyne:** add words-only mode in which the kana path rests ([ccfc058](https://github.com/Comradin/kana/commit/ccfc05828660a148fbc1c0d8998b5278908a314e))
* **fyne:** choose the meaning language and explain words-only status ([6243d7a](https://github.com/Comradin/kana/commit/6243d7a7db0dc360cc5975322d6b5dab9980c4a7))
* **fyne:** focus on weak items and filter a list's lessons ([9b321aa](https://github.com/Comradin/kana/commit/9b321aa978406d401e3bef5bb5d8b29644792bfc))
* **store:** store words-only mode, focus and lesson filters ([8fa7777](https://github.com/Comradin/kana/commit/8fa777793c5b2cc01fcd7fb83d48bd1bc4a5b436))
* **vocab:** choose lessons of a list and list its lessons and languages ([40a27f1](https://github.com/Comradin/kana/commit/40a27f1b27129af515225200d725251fbe47beca))


#### Improvements and fixes

* **fyne:** ignore lessons a list no longer has in its lesson filter ([73a5ba3](https://github.com/Comradin/kana/commit/73a5ba37f0130ce0b77f944399e56b65c277aec1))

## [0.4.1](https://github.com/Comradin/kana/compare/v0.4.0...v0.4.1) (2026-10-20)

**Answer words by their meaning.** A falling word now also accepts its meaning: type `Regenschirm` or `umbrella` instead of `kasa`. Your meaning language and English both count. Capitals, text in parentheses, a leading "to" or article, and umlauts written as `ae`, `oe`, `ue`, `ss` don't matter; typos do. After a hit by meaning the reading is shown as well, so you pick it up along the way.

**Why didn't any words fall?** A word only falls once all its kana are learned. After a progress reset most words wait until their kana are learned again – the status panel now tells you how many words are waiting for kana. And when only a few words are playable they come more often: at least one tile in twenty is a word.

### Commits

#### Improvements and fixes

* **fyne:** accept a word's meaning as its answer ([853ec96](https://github.com/Comradin/kana/commit/853ec96db3a05072dcb42886e3e8671ce3a56fde))
* **fyne:** count words waiting for kana, raise the word share ([4fe8052](https://github.com/Comradin/kana/commit/4fe805252fcb96e6fe9e4a5dba2a267ba9887d40))

## [0.4.0](https://github.com/Comradin/kana/compare/v0.3.0...v0.4.0) (2026-10-20)

**Import your own word lists.** *Settings → Words → Import word list…* lets you pick a CSV, TSV, TXT or JSON file, for example a vocabulary list from your textbook. The game checks it, shows a preview – number of words and lessons, meaning languages, skipped rows and the first few words – and keeps its own copy. No more copying files into hidden folders, no restart.

- Importing a list with the same name replaces it; your progress is kept.
- Files Excel saves as "Unicode Text" work too. For spreadsheets (`.xlsx`, `.ods`) and files in other encodings the game explains how to save them as "CSV UTF-8".
- The imported list is selected as the source of new words, and the Words tab shows where your word lists are kept.

### Commits

#### New

* **fyne:** import a word list from the settings ([7a96ef6](https://github.com/Comradin/kana/commit/7a96ef6c77e42b4b1e84e8c2505b09036fa61172))
* **fyne:** reload word lists at runtime and create their folder ([40f8c36](https://github.com/Comradin/kana/commit/40f8c368cc3e0fe136ae373d857a8b8376f9ed04))
* **vocab:** check and store a word list for import ([f661147](https://github.com/Comradin/kana/commit/f66114789c471ee9cdcdae99d75f1f17acf5e1ac))


#### Improvements and fixes

* name every file a word list import replaces ([5ee95bf](https://github.com/Comradin/kana/commit/5ee95bfff8478fb1b393bec0263ffd84d4228684))

## [0.3.0](https://github.com/Comradin/kana/compare/v0.2.0...v0.3.0) (2026-10-19)

**Word packages.** New words now arrive in small packages of 3, 5 or 10 words. Each package is introduced in a dialog with the kana, romaji and meaning before its words start to fall. Tick "Already known" for a word you know and a replacement comes instead. Words come from your own list, lesson by lesson, or from the JLPT list (N5 first, the most common words first); when a JLPT level is done, the game offers the next one.

**Learning levels.** Every kana and every word has a level from 0 to 5: a hit raises it by one, a mistake lowers it by two, and at 5 it is learned. A wrong answer counts as a mistake when only one tile is on the field. The status panel colours the kana by level – new, learning, learned – and keeps a red frame on kana that slipped until you have learned them again.

**Start a script over.** The Kana tab of the settings can reset your hiragana or katakana progress and restart its learning path.

**Settings in tabs, and your choice of kana font.** The settings are split into Kana, Words, Game and Sources. Choose between Noto Sans JP, M PLUS Rounded 1c and the system font, each previewed at tile size.

**Keyboard first.** The introductions of new kana rows and word packages close with Enter, so you never have to reach for the mouse.

Also: the status labels no longer jump around, and the font licences ship with the downloads.

### Commits

#### New

* **fyne:** colour the kana grid by level and keep the slipped frame ([29ad8a0](https://github.com/Comradin/kana/commit/29ad8a0273a40c1b2af68526b6c337fd8c18bee1))
* **fyne:** confirm the kana and word intros with Enter ([9c771ca](https://github.com/Comradin/kana/commit/9c771ca4dc18eb440bbe0b2ed1a55d0e5ff95019))
* **fyne:** count a wrong answer on the only tile as a mistake ([a2afd90](https://github.com/Comradin/kana/commit/a2afd90547d5bf74f8992627abbd6cc6897834c4))
* **fyne:** decide kana mastery and playable words by learning level ([86891f5](https://github.com/Comradin/kana/commit/86891f57597f40565e7dd64d29f546ad1997f5b7))
* **fyne:** decide learned words by level and mark known words learned ([3f43212](https://github.com/Comradin/kana/commit/3f4321250fe939d28ab8f83b563bd0652dba6e4d))
* **fyne:** introduce word packages in a dialog ([b72658b](https://github.com/Comradin/kana/commit/b72658b02bed0d6b4770383c2d0f4dedb808b769))
* **fyne:** let the learner choose the kana font ([87eb055](https://github.com/Comradin/kana/commit/87eb055cbfe0b07dc2fae3ab245a4b3e07ac580c))
* **fyne:** offer word packages and the next JLPT level ([cee91ff](https://github.com/Comradin/kana/commit/cee91ffae3545784344bd1eedbbb787c0ac4a47c))
* **fyne:** play only introduced words ([27ce0c7](https://github.com/Comradin/kana/commit/27ce0c7d389fd9fdf70ff0c4ebd653723d495936))
* **fyne:** reset a script's progress from the settings ([517a729](https://github.com/Comradin/kana/commit/517a729b6d6bddbd413542fb9b6f15ccd8e5b82a))
* **fyne:** reset one script's progress and learning path ([a8817ca](https://github.com/Comradin/kana/commit/a8817ca357f160f5da9b0331ed51d3b86f9dd24a))
* **fyne:** split the settings into tabs and add a Sources tab ([c54aa62](https://github.com/Comradin/kana/commit/c54aa62aacfba86bdbec08233b78c8084776c400))
* **fyne:** track learning levels for kana and words ([db4bcc7](https://github.com/Comradin/kana/commit/db4bcc7f6a616653be079c6d00b46fdc83db1153))
* **store:** add introduced words and word package settings ([fa7b54f](https://github.com/Comradin/kana/commit/fa7b54f707147c4742ff6b2bd7609149fbbc0cab))
* **store:** add learning progress table seeded from stats ([6385f26](https://github.com/Comradin/kana/commit/6385f269f51b334acb36c628942881fc2e768114))
* **store:** delete a set of characters' progress ([052c6f2](https://github.com/Comradin/kana/commit/052c6f27995c108e3768b46a08394df51c7e550b))
* **vocab:** add a frequency rank to the word data ([c6e86fe](https://github.com/Comradin/kana/commit/c6e86feb6da8fd3448f26ed8d3689129c8b5853e))
* **vocab:** pick the next word package in learning order ([99266d8](https://github.com/Comradin/kana/commit/99266d85ade68705befa889ed1ed5b29f92d21cb))


#### Improvements and fixes

* **fyne:** keep package refills in their lesson and retry failed saves ([e0ea7f9](https://github.com/Comradin/kana/commit/e0ea7f90fb3005ad88db59bab25cf0fda50da23e))
* **fyne:** keep the status labels at a steady width ([a26b762](https://github.com/Comradin/kana/commit/a26b762e4e2d885f0209b3297a48db0f60a9c620))
* **fyne:** preview the kana fonts at tile size with telling kana ([015a1ed](https://github.com/Comradin/kana/commit/015a1eddc2270cad669e858d46124aa886b05609))
* **fyne:** show き in the kana font preview ([d3e6ff3](https://github.com/Comradin/kana/commit/d3e6ff3bd7ba999f4f1b0d9fba9fc92691761cfa))
* ship the font licences with the release packages ([bebb427](https://github.com/Comradin/kana/commit/bebb42771e613d920d5829318bad147d8e30f604))
* **vocab:** keep word packages within one lesson ([19862db](https://github.com/Comradin/kana/commit/19862db477eb6728ac8fb11c5d3b0e5bd3e7dedf))

## [0.2.0](https://github.com/Comradin/kana/compare/v0.1.0...v0.2.0) (2026-10-15)

**Words join the kana.** Once you know enough kana, words made only of kana you know start falling among the single characters. Type the whole reading – `sensei`, `kitte`, `koohii` or `ko-hi-` all work, spaces are ignored. A hit shows the meaning; a missed word shows its romaji and meaning.

**Bring your own word lists.** Put a CSV, TSV or JSON file, for example from your textbook, into the game's `words` folder; lessons and German or English meanings are understood, and common column names such as "Japanische Schrift" or "Lektion" are recognised.

**A bundled JLPT word list** built from JMdict (CC BY-SA) with meanings in several languages, refreshed monthly. It is off by default so beginners are not flooded; words make up at most a tenth of the falling tiles.

The status panel shows how many words you have and where they come from.

### Commits

#### New

* **fyne:** add word tiles ([2f3e297](https://github.com/Comradin/kana/commit/2f3e29741597345421f1c02abf190569feb4d18b))
* **fyne:** answer and miss word tiles ([b3d24eb](https://github.com/Comradin/kana/commit/b3d24ebd7d1eb2e95bd887b207c049ce5a2db436))
* **fyne:** keep word progress per reading ([f641058](https://github.com/Comradin/kana/commit/f6410589d4617fb0c662eec3c63952289403f755))
* **fyne:** load words and decide which are playable ([687ff45](https://github.com/Comradin/kana/commit/687ff45c87a964e0ad6d9ffdf8ec3d0ff8d5e31c))
* **fyne:** show the meaning after a word hit or miss ([99cc8c9](https://github.com/Comradin/kana/commit/99cc8c922447797b05b3a32d3fdb3c44a3c4613a))
* **fyne:** show words and their source in the status panel ([2f9c822](https://github.com/Comradin/kana/commit/2f9c822db9d803edddef9c475b09b0086f8bb4c8))
* **fyne:** spawn words among the kana ([c7c85cd](https://github.com/Comradin/kana/commit/c7c85cdce154cfab0ae289b06046aed899cb58d0))
* **store:** add word stats and word settings ([6fce227](https://github.com/Comradin/kana/commit/6fce227100859c045fd4b933af05cefe172c68c8))
* **vocab:** add the word model and meaning lists ([7922678](https://github.com/Comradin/kana/commit/7922678f197cf898a62900772ff76eb667006d4a))
* **vocab:** bundle the JLPT word list ([0bfc7b2](https://github.com/Comradin/kana/commit/0bfc7b249759e28d853d6a2038d77a3fbe17beac))
* **vocabgen:** generate the bundled word list from JMdict and JLPT lists ([70067f8](https://github.com/Comradin/kana/commit/70067f84b8422719132a405b7e09d690455e8337))
* **vocab:** load private word lists from a directory ([34cc1f9](https://github.com/Comradin/kana/commit/34cc1f975df77a0bcf004092de16fe6a9bf47321))
* **vocab:** merge sources into one index and filter playable words ([3ef0cd8](https://github.com/Comradin/kana/commit/3ef0cd80be211ebea3e79d2691d5adbb798ded48))
* **vocab:** read CSV and TSV word lists ([190a214](https://github.com/Comradin/kana/commit/190a214753b43a3f12d0d16cbdabc847d8117b34))
* **vocab:** update the word data monthly ([e84f3b4](https://github.com/Comradin/kana/commit/e84f3b4352c55c110e0586228233310ff7df3bc5))


#### Improvements and fixes

* **fyne:** address stage 3 review ([b0d9224](https://github.com/Comradin/kana/commit/b0d9224bdcd8da4325a3a58b9130a9f23e9c9d4b))
* **vocab:** address stage 2 review ([7be7316](https://github.com/Comradin/kana/commit/7be7316060f5bfde4e3bc6da677cb274401043da))
* **vocab:** throttle words until word packages arrive ([6e37fd7](https://github.com/Comradin/kana/commit/6e37fd7f0eefc1236b4f27cd8794109b3ab5a18f))

## 0.1.0 (2026-10-13)


### Features

* **fyne:** add app metadata and icon ([d3660c5](https://github.com/Comradin/kana/commit/d3660c570b8dcb133ced5d8504fe77f32ad24527))
* **fyne:** show the app version ([1814d79](https://github.com/Comradin/kana/commit/1814d7980b60fcfaa946955b234e05224fda652e))
* **release:** add the macOS release script ([9a0372b](https://github.com/Comradin/kana/commit/9a0372b55798851050f644f59786336b33f5ca0c))
* **release:** build releases with release-please and GitHub Actions ([c390eb6](https://github.com/Comradin/kana/commit/c390eb64475c575b4e3a1ec78a32b716d8c59fc2))
