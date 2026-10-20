# Changelog

## [0.4.0](https://github.com/Comradin/kana/compare/v0.3.0...v0.4.0) (2026-10-20)


### Features

* **fyne:** import a word list from the settings ([7a96ef6](https://github.com/Comradin/kana/commit/7a96ef6c77e42b4b1e84e8c2505b09036fa61172))
* **fyne:** reload word lists at runtime and create their folder ([40f8c36](https://github.com/Comradin/kana/commit/40f8c368cc3e0fe136ae373d857a8b8376f9ed04))
* **vocab:** check and store a word list for import ([f661147](https://github.com/Comradin/kana/commit/f66114789c471ee9cdcdae99d75f1f17acf5e1ac))


### Bug Fixes

* name every file a word list import replaces ([5ee95bf](https://github.com/Comradin/kana/commit/5ee95bfff8478fb1b393bec0263ffd84d4228684))

## [0.3.0](https://github.com/Comradin/kana/compare/v0.2.0...v0.3.0) (2026-10-19)


### Features

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


### Bug Fixes

* **fyne:** keep package refills in their lesson and retry failed saves ([e0ea7f9](https://github.com/Comradin/kana/commit/e0ea7f90fb3005ad88db59bab25cf0fda50da23e))
* **fyne:** keep the status labels at a steady width ([a26b762](https://github.com/Comradin/kana/commit/a26b762e4e2d885f0209b3297a48db0f60a9c620))
* **fyne:** preview the kana fonts at tile size with telling kana ([015a1ed](https://github.com/Comradin/kana/commit/015a1eddc2270cad669e858d46124aa886b05609))
* **fyne:** show き in the kana font preview ([d3e6ff3](https://github.com/Comradin/kana/commit/d3e6ff3bd7ba999f4f1b0d9fba9fc92691761cfa))
* ship the font licences with the release packages ([bebb427](https://github.com/Comradin/kana/commit/bebb42771e613d920d5829318bad147d8e30f604))
* **vocab:** keep word packages within one lesson ([19862db](https://github.com/Comradin/kana/commit/19862db477eb6728ac8fb11c5d3b0e5bd3e7dedf))

## [0.2.0](https://github.com/Comradin/kana/compare/v0.1.0...v0.2.0) (2026-10-15)


### Features

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


### Bug Fixes

* **fyne:** address stage 3 review ([b0d9224](https://github.com/Comradin/kana/commit/b0d9224bdcd8da4325a3a58b9130a9f23e9c9d4b))
* **vocab:** address stage 2 review ([7be7316](https://github.com/Comradin/kana/commit/7be7316060f5bfde4e3bc6da677cb274401043da))
* **vocab:** throttle words until word packages arrive ([6e37fd7](https://github.com/Comradin/kana/commit/6e37fd7f0eefc1236b4f27cd8794109b3ab5a18f))

## 0.1.0 (2026-10-13)


### Features

* **fyne:** add app metadata and icon ([d3660c5](https://github.com/Comradin/kana/commit/d3660c570b8dcb133ced5d8504fe77f32ad24527))
* **fyne:** show the app version ([1814d79](https://github.com/Comradin/kana/commit/1814d7980b60fcfaa946955b234e05224fda652e))
* **release:** add the macOS release script ([9a0372b](https://github.com/Comradin/kana/commit/9a0372b55798851050f644f59786336b33f5ca0c))
* **release:** build releases with release-please and GitHub Actions ([c390eb6](https://github.com/Comradin/kana/commit/c390eb64475c575b4e3a1ec78a32b716d8c59fc2))
