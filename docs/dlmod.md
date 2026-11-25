# dlmod

`dlmod` is an 8 MB program in [`tools/dlmod`](../tools/dlmod). It uses only Go's standard library and has no installer or background service. It finds Deadlock through Steam, copies mods into `game/citadel/addons` under free `pakNN_dir.vpk` names, and edits `gameinfo.gi` so the game loads that folder. It works with any VPK mod and installs `.zip`, `.rar` and `.7z` downloads. It goes online only when you browse or download mods or update the HUD, through the `curl` that comes with Windows 10 and later, which keeps dlmod half the size it would be with its own HTTPS code. `.rar` and `.7z` archives are opened with the `tar.exe` that also comes with Windows (or 7-Zip, if installed).

It also builds mods of its own: [custom characters](custom-characters.md) (models, hero select screens, cards, icons, ability models, effects, text) and custom sounds.

## The mod list

Double-click dlmod to open its full-screen view of every mod, with its slot, file count, size, and conflicts or update warnings. It updates by itself when something changes outside it: a game update, a mod added by hand, or Deadlock starting.

![dlmod showing seven mods: a HUD, skins, a custom model, custom sounds, one conflict and one mod switched off](images/dlmod.png)

The selected mod, *Big Numbers HUD*, changes the same file as the Counter Intelligence HUD, so both carry a `!` and the line below explains which file. *Kill Sounds* is switched off: it waits in `game/dlmod` until you switch it on.

| Key | Action |
| --- | --- |
| `↑` `↓` (or `j` `k`) | Select a mod |
| `Space` or `Enter` | Switch the selected mod on or off. It keeps its place in the list. |
| Drag a file into the window | Install it straight away. `i` lets you type a path instead. |
| `b` | Browse mods online and install them (see below) |
| `s` | Listen to the game's sounds and replace them with your own (see below) |
| `d` | Delete the selected mod, after a confirmation |
| `f` | Profiles: switch to a saved set of mods, save the mods that are on now, or delete a profile |
| `u` | Download the Counter Intelligence HUD for your game build and replace the old one (asks first) |
| `r` | Repair `gameinfo.gi` after a game update |
| `x` | Undo dlmod's `gameinfo.gi` change; mods stay installed but stop loading |
| `p` | Start Deadlock through Steam |
| `?` | Help |
| `q` or `Esc` | Quit |

Deadlock updates and Steam's *Verify integrity of game files* replace `gameinfo.gi`, which turns all mods off. The screen then shows **MODS OFF**: press `r`, or start the game with `dlmod play`, which repairs first. Consoles older than Windows 10 get a numbered menu instead, which `-plain` also selects.

## Browsing mods online

`b` opens the catalog Deadlock Mod Manager uses: about 12,000 mods mirrored from GameBanana by [deadlockmods.app](https://deadlockmods.app), each with its hero and category. dlmod downloads the list once (about 15 MB) and keeps it for six hours; `R` downloads it again. It opens on skins, most downloaded first. `←` `→` pick a hero, `c` cycles the category, `/` searches names and authors, and `n` shows NSFW mods, which are hidden otherwise. `Enter` downloads the selected mod, checks its published MD5, and installs it like a dropped file, asking which file when a mod has several (often with or without voice lines). The mod's files come from deadlockmods.app, or from GameBanana's own API when the mirror has none.

![dlmod's online browser listing the most downloaded Haze skins, with the selected skin's picture drawn in sixel](images/dlmod-browse.png)

The selected mod's preview is drawn in the terminal. dlmod draws pictures itself, with no extra library:

| Preview mode (`m`) | Where it works |
| --- | --- |
| blocks | Any terminal with 24-bit colour: Windows Terminal, the Windows 10+ console, macOS and Linux terminals. Each character shows two pixels with `▀`, so pictures are coarse. |
| sixel | Full resolution in Windows Terminal 1.22 or later, WezTerm, foot, mlterm and xterm (`-ti vt340`). Each picture gets its own 256-colour palette (median cut) with Floyd–Steinberg dithering, and is drawn at the pane's exact pixel size, which dlmod asks the terminal for. |
| kitty | Full resolution in kitty, Ghostty and WezTerm. |

dlmod starts with kitty in kitty and Ghostty and sixel in WezTerm and foot. Elsewhere it starts with blocks and asks the terminal which graphics it draws (its device attributes). Windows Terminal 1.22 and later answer that they draw sixel, and previews switch to full resolution on their own. `v` (or `Space`) shows a big preview that fills the screen, which makes block pictures much sharper; a smaller terminal font gives more cells and sharper blocks too. Set `DLMOD_IMAGES=sixel` (or `blocks`, `kitty`, `off`) to choose a mode. Previews are cached in `game/dlmod/catalog/previews`.

![dlmod's big preview (v) of the same skin in sixel](images/dlmod-preview.png)

## Replacing sounds

`s` lists every sound in the game (about 95,000), grouped by hero: voice lines in `sounds/vo/<hero>` and ability sounds in `sounds/abilities/<hero>`. `←` `→` pick a hero and `/` searches by name. `Enter` plays the game's sound in your default player. To replace the selected sound, drop a `.wav` (8- or 16-bit PCM, mono or stereo) or `.mp3` file onto the window, or press `r` and type its path. `p` plays your version and `x` restores the original.

![The sounds screen filtered to Haze and "kill", with two voice lines replaced by a dropped .wav file](images/dlmod-sounds.png)

dlmod compiles the file into the game's sound format itself, with no Valve tools, Blender or Source 2 Viewer. The format follows [ValveResourceFormat](https://github.com/ValveResourceFormat/ValveResourceFormat): an empty `DATA` block, a `CTRL` block describing the audio in binary KeyValues3, and the audio after it. Compiling the MP3 inside one of Valve's own sounds reproduces every field Valve's compiler wrote. All replacements go into one mod, *Custom sounds*, which dlmod rebuilds and reinstalls after every change. The list of replaced sounds is kept in `game/dlmod/sounds`.

## Commands

| Command | What it does |
| --- | --- |
| `dlmod <file.zip, .rar, .7z or .vpk>` | Install mods, including multi-part sets (`pakNN_000.vpk`). If a newer version changes the same files, dlmod offers to replace the old one in its slot. |
| `dlmod list` | Show mods, conflicts between them, and whether the game loads mods. |
| `dlmod off <mod>` / `dlmod on <mod>` | Move a mod out of `addons` without deleting it, and back. |
| `dlmod remove <mod>` | Delete a mod. |
| `dlmod patch` / `dlmod unpatch` | Add or remove dlmod's `gameinfo.gi` lines. `unpatch` restores Valve's file exactly. |
| `dlmod play` | Patch if needed, then start Deadlock through Steam. |
| `dlmod play <hero>` | The same, straight into the Hero Sandbox as that hero, to check a character (Steam asks before it uses the launch options). |
| `dlmod update` | Download the Counter Intelligence HUD built for the installed game build from the project's [GitLab releases](https://gitlab.com/mrhakan/deadlock-modkit/-/releases), check it against the `.sha256` file published with it (a release without one is refused), and install it in place of the old version. |
| `dlmod search <words> [-hero H] [-category C]` | Search the online catalog, for example `dlmod search -hero wraith` or `dlmod search catgirl`. |
| `dlmod get <id or GameBanana address>` | Download and install a mod, for example `dlmod get https://gamebanana.com/mods/634614`. |
| `dlmod profile save <name>` / `use <name>` / `delete <name>` | Remember which mods are on, switch on exactly that set later (everything else is switched off, not deleted), or forget a profile. `dlmod profile` lists them. |
| `dlmod extract <path> [folder]` | Copy files out of the game's archive, for example `dlmod extract models/heroes_staging/haze`. |
| `dlmod studio`, or drop a model on dlmod | Open the model studio to put a `.glb`, `.gltf` or `.obj` model on a hero. |
| `dlmod model <hero> <file>` | Put a model on a hero without the studio. |
| `dlmod select <hero> [bg=… name=… logo=… tags=… colors=… scene=hide]` | Set the hero select screen for a replacement character. |
| `dlmod image <game image> <picture>` / `dlmod icons <hero> [n=picture]` | Replace cards, portraits and ability icons. |
| `dlmod prop <game model> <file>` / `dlmod paint <hero> [#main,#accent,#glow]` / `dlmod effects <hero> <#colour>` | Replace an ability model, recolour ability effects (with up to three colours of your own, or the cobalt palette). |
| `dlmod text KEY=VALUE` | Change a line of the English hero text. |
| `dlmod meshy <hero or all> [folder] [concept=…]` / `dlmod apply <folder or pack>` | Write per-hero Meshy prompts for a full replacement character, and install what was generated, or a character pack. |
| `dlmod pack <hero or folder> [file]` / `dlmod unpack <file.dlmodchar>` | Pack a character into one `.dlmodchar` file to keep or share, and make a pack the hero's character in the creator. |

The character commands are explained in [Custom characters](custom-characters.md).

`<mod>` is a number from `dlmod list`, a slot such as `pak03`, or part of the mod's name. dlmod finds the game through Steam's library folders. If it cannot, pass `-game "D:\SteamLibrary\steamapps\common\Deadlock"`. It keeps its list of mods, switched-off mods, and a copy of the previous `gameinfo.gi` in `game/dlmod`, which the game never reads.

The full-screen view needs about 5–8 MB of memory. While it waits it checks four file timestamps (and whether the game is running) once a second, and redraws only when something changed. Key presses take under a millisecond to show. On Windows, dlmod reads Steam's location from the registry and checks for the game through the system API, without starting other programs. Listing 40 mods, two of them with 6,000 files each, takes about 5 ms.

Windows SmartScreen may warn about the program because it is not signed. Choose **More info → Run anyway**, or build it yourself with `go build` in `tools/dlmod`.
