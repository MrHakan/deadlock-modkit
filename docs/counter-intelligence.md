# Counter Intelligence HUD

An experimental in-game HUD that suggests counter items against the enemy lineup. Open the scoreboard or shop to see:

- **Counter items for the enemy lineup.** The three items from the reference counter table that answer the most enemy heroes, each with its icon, soul price, what it does against them (for example *Anti-heal* or *CC immunity*), and the portraits of the enemy heroes it counters. Item icons and portraits come from the game's own textures. Items you already own are left out.
- **What to buy now.** A **BUY NOW** badge marks the first suggestion your souls can pay for, and prices you cannot reach yet turn grey.
- **What match data says.** Each row shows the estimated change in the buyer's win rate against the current lineup and the minute the item is usually bought, for example `+0.6 WR · ~14m`. A fourth row adds the item with the largest measured gain against the lineup that the table does not already suggest.
- **Lane counters.** For the first ten minutes the fourth row switches to the best early item against the enemies in your lane.
- **Patch warnings.** When a game patch changes an enemy hero after the counter table was last reviewed, a **PATCHED** badge appears next to the title; when it changes a suggested item, that row gets a **!** mark. Hover either one to see what changed, taken from Valve's patch notes or, when Valve changed values without notes, from the game data. The warnings stay until the counters are reviewed for that patch.
- **Enemy counter-purchase alerts.** When an enemy buys an item from the counter list of a hero on your team, a banner names the buyer, the item, and the allies it counters. It stays up for 8 seconds and turns red when the item counters your own hero. Each enemy purchase alerts once per match.
- **Counters against you.** Items enemies own that are on your own hero's counter list are listed in red under the panel title, for example *Countering you: Healbane (Haze)*.

## Reference counter table

`src/counters.json` holds five counter items for every hero, with a short tag per item written from the in-game item description. For 42 heroes they come from the *Counter Eşyalar: kime karşı ne alınır?* series. Pocket and Rat King are not in the series. Their lists come from [Deadlock Item Builder's counter guides](https://deadlockitembuilder.com/counter/pocket), keeping the picks that match data does not contradict, and `otherSources` in the file records how each was chosen. Each item counts once for every enemy hero whose list includes it. Ties go to the item with the larger measured win-rate gain against the lineup, then to the item listed earlier, then to the cheaper item.

To change the reference, edit the item class names in `src/counters.json`. The tests check that every entry exists in the catalog and has a tag.

`reviewed` in the same file records the date the table was last checked against the game and the game build whose data it was checked against, with a later date and build for heroes checked since. `tools/patch_notes.py` collects two kinds of changes into `src/patches.json`:

- **Patch notes.** Every gameplay patch published after the review date, read from deadlock-api's copy of Valve's posts. The tool keeps the lines of the form `- <Hero or item>: <change>`.
- **Changes without patch notes.** Valve sometimes changes values without saying so: build 6833, between the 10-30 and 11-04 updates, raised Spiritual Overflow's spirit lifesteal from 13% to 15%, and no post mentions it. The tool compares the hero, ability, weapon and item values of the reviewed build with those of the current build, as deadlock-api reads them from each build's game files. Every hero or item whose values changed and that no patch note already warns about is listed under *an update without patch notes*, with lines such as `Sovereign's Decree - Bullet damage 5.7 → 5.8`. Abilities most heroes share, such as the jump, are left out, and so are weapon values the game computes from others, such as damage per second.

Those heroes and items carry the PATCHED and **!** warnings. After checking a hero's counters against newer notes or data, set its date and build to the current ones and run the tool again, and its warnings disappear. Right now the series' lists are taken to match the 10-30-2026 update, whose data builds 6826 to 6829 carry. The 11-04-2026 changes to Apollo, Baba, Celeste, Deadman Danny, Mo & Krill, Nurse Harrow and Healbane are flagged, and so is Spiritual Overflow's unannounced change. Rat King and Pocket were checked against match data on build 6864. The 11-18-2026 update (builds 6863 and 6864) changed no hero, ability, weapon or item value and no interface file, so it adds no warnings. Its download was large because it rebuilt about 4,600 hero animation files and replaced Baba's ability effects. It also replaced `gameinfo.gi`, which turns mods off until the search paths are added again (`r` in dlmod).

## Match data

`src/stats.json` is generated by `tools/update_counter_stats.py` from the public [deadlock-api](https://api.deadlock-api.com/docs) item statistics, using its `enemy_hero_ids` filter (ranked and unranked matches at all ranks). The window starts at the latest gameplay patch: the tool reads the patch notes and takes the newest post that changes an item or a hero, so the numbers describe the items as they are now. The current window starts with the 11-04-2026 update, which nerfed Healbane a week after the 10-30 update changed 12 of the counter items. For each item and enemy hero it takes the item's win rate against that hero and subtracts:

1. the item's win rate in all matches, which removes buyer bias (late items are bought by players who are already ahead);
2. the change every buyer sees when that hero is on the enemy team, which removes the hero's overall strength;
3. the average remaining gain of the other items of the same price, which removes the hero's power curve (against a hero who falls off late, every late item looks good).

Changes smaller than two standard errors count as zero, and items need 20,000 purchases against the hero. Steps 1 and 2 reproduce the per-hero numbers published by [Deadlock Academy](https://deadlockacademy.com/counter-items); step 3 is this project's addition. The HUD adds the values over the enemy lineup, which assumes the effects add up.

Lane counters repeat the measurement for players in the same lane as the enemy hero (`same_lane_filter`), counting only items bought in the first ten minutes and costing at most 3,200 souls. The typical buy minute of each item is the average purchase time in the same matches.

[`counter-data.md`](counter-data.md) compares the table with the data for every hero and lists the lane counters. In the current window, 70 of the 220 reference picks raise the buyer's win rate measurably, 92 show no significant change, and 58 lower it. Defensive items bought in reaction to a hero that is already winning look worse than they are, so treat a negative value as "the data does not back this pick", not "the item is useless".

## Research

Counter sites take one of two approaches, and this mod combines both:

- **Curated coverage.** [Deadlock Item Builder's counter helper](https://deadlockitembuilder.com/counter-item-helper) ranks items by how many of up to six enemies they answer and explains each by threat type (anti-heal, gun damage, crowd control, spirit burst). The reference table and tags follow this approach.
- **Match statistics.** [Deadlock Academy](https://deadlockacademy.com/counter-items) and [Worldlock](https://worldlock.org/en/counters/) list the items whose win rate rises most against each hero. [Statlocker](https://statlocker.gg/items) warns that a raw item win rate mostly reflects who buys the item, which is why the match data above corrects for buyer bias, hero strength, and item timing. [Statlocker's matchups](https://statlocker.gg/heroes/matchups), [Deadlock Labs](https://deadlocklabs.gg/heroes/matchups/), and [Counterlock](https://counterlock.gg/) cover hero-against-hero matchups, which matter in the draft rather than in the shop.

## How it reads the game

The suggestion panel sits above the ability bar on the scoreboard and below Quick Buy on the left side of the shop. Everything comes from what the game already shows you, mostly the top bar (panel names from Valve's interface files, as published by [GameTracking-Deadlock](https://github.com/SteamTracking/GameTracking-Deadlock)):

- Both teams' heroes come from the player cards, including their rendered hero names. The card marked `LocalPlayer` identifies your side, and enemy cards marked `IsInLocalPlayerLane` are your lane opponents.
- Your spendable souls come from the shop or HUD balance (`CurrentGoldAmount`), and the match time from the top bar clock (`GameTime`).
- Items you own come from the shop's own item cards while the shop is open: the game marks the cards of items you own (or own as part of an upgrade) with `owned` or `usedAsComponent`, and names the item in their `modName` label. Your own rows in the shop's recent-purchase list count too. An owned item stays out of the suggestions for the rest of the match.
- Enemy purchases come from the shop's recent-purchase list, which the mod reads every half second while the shop is open, so alerts appear when you open the shop. Items enemies already own are read from the item icons on their top-bar cards (`mod_universal`), which the scoreboard shows, through the `{s:mod_name}` dialog variable the shop's cards use. If the game does not give the icons that variable, the *Countering you* line stays empty.

The PATCHED badge and the **!** marks are the only parts of the panel that react to the mouse; everything else lets clicks through to the game. They never take keyboard focus (`acceptsfocus="false"`, like Valve's own HUD buttons), so movement and the shop key keep working. Hovering needs a mouse cursor, which the shop always has. The mod's class listener sits in a panel of its own, so the `gShopOpen` and `gScoreboardOpen` classes it needs never reach the game's panels; under the stock HUD panel they once matched a Valve style rule and moved the shop's quick-buy list.

When the enemy lineup changes, the mod treats it as a new match and clears its alert history and owned items. Suggestions are only recomputed when the lineup, the lane phase or your items change; souls only move the badge.

The recommendation engine, HUD state changes, and archive format are covered by local tests. Displaying the HUD in Deadlock still requires in-game verification. If one of the panels above turns out to be named differently in game, its feature switches itself off: no badge without souls, no lane row without lane opponents, and owned items stay listed. The HUD does not track item sales. When the script loads, the console shows `[DCI] Counter items HUD loaded`. During a match, the mod works from the visible HUD and the bundled catalog, counter table, and match data. It makes no network requests.

## Catalog

The catalog in `src/catalog.json` holds the shop items and heroes of one game build (the build is in the main README, which the daily game update pipeline updates). The HUD displays English names; Turkish aliases help match item and hero names read from the game client, and ambiguous labels are skipped. When targeting a new build, refresh the catalog with `tools/update_catalog.py <build>` and use the stock HUD from that build.

## Build the VPK

Building requires Python 3.9+, a Steam installation of Deadlock, and [Source2Viewer-CLI](https://github.com/ValveResourceFormat/ValveResourceFormat/releases). The build reads the base HUD layout from the installed game.

```powershell
py -3 tools/build.py --deadlock-dir "C:\Program Files (x86)\Steam\steamapps\common\Deadlock" --vrf "C:\Tools\Source2Viewer-CLI.exe"
```

If you already have the extracted stock layout, use `--hud-xml path/to/hud.xml`. [GameTracking-Deadlock](https://github.com/SteamTracking/GameTracking-Deadlock/blob/master/game/citadel/pak01_dir/panorama/layout/hud.xml) publishes it for the live build. The build produces `dist/deadlock-counter-intelligence-experimental.zip` with one `pak01_dir.vpk` inside.

The project lives on [GitLab](https://gitlab.com/mrhakan/deadlock-modkit), and [`.gitlab-ci.yml`](../.gitlab-ci.yml) runs its pipelines:

- **Tests** run on every push and merge request: the HUD tests, a build against the live stock HUD, and dlmod's format check, vet and tests.
- **Releases.** Pushing a version tag such as `v0.6.0` runs the tests, builds the ZIP from the GameTracking layout (and stops if that layout is from a different build than the catalog), builds dlmod for Windows and Linux, and publishes a release with the notes in `release-notes/<version>.md` ([`tools/publish-release.sh`](../tools/publish-release.sh)). Every file goes to the project's package registry with a `.sha256` file next to it, as `sha256sum` writes it; `dlmod update` refuses a HUD whose release has none. To publish a file by hand, add `sha256sum <file> > <file>.sha256` (or, in PowerShell, `(Get-FileHash <file>).Hash > <file>.sha256`) to the release as well.
- **Game updates.** A daily pipeline schedule on `main` checks Deadlock's client build. When it changes, the pipeline refreshes the catalog, the match data and the patch warnings (from the notes and from the game data), runs the HUD tests, builds the mod against the new stock HUD, commits the update to `main` and pushes the next patch tag, which starts the release, so a HUD for the new build is usually available within a day of a game update. If deadlock-api does not have the new build yet, it tries again the next day; if a test or the build fails, nothing is released. It pushes with `RELEASE_TOKEN`, a masked CI/CD variable holding a project access token with the `write_repository` scope. Run it by hand with **Build → Pipelines → Run pipeline** and the variable `GAME_UPDATE=true` (add `FORCE=true` to release even if the build did not change).

## Testing in game

Use a private lobby. With the console open, loading a match prints `[DCI] Counter items HUD loaded`; if that line is missing, the script did not load, and the panel stays at its layout defaults. Check that the suggestions match the enemy lineup, that the fourth row shows a lane pick before minute 10 and a match-data pick after it, that the BUY NOW badge follows your souls, that an item you buy disappears from the suggestions, that hovering PATCHED (for example with Celeste or Baba on the enemy team) shows their changes and that the panel does not block clicks in the shop, that an enemy buying an item from an ally's counter list (for example Healbane against Infernus) shows the banner, that an allied purchase is ignored, and that disabling the mod restores the stock HUD.

If two mods change the same file, only one of them takes effect, and guides disagree on whether the lower or the higher `pakNN` number wins. This mod replaces `panorama/layout/hud.vxml_c`, so do not combine it with another HUD mod; `dlmod list` shows such conflicts. A HUD mod built for an older game build can stop Deadlock from starting after a large update (for example with an "Unable to find child" error). Switch it off and use the release built for the new game build.
