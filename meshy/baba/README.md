# Baba (`baba`)

Weapon: Bunko-Night Special. Tags: Cackling, Beady, Hexing.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/baba
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Threadsap | Threadsap | `threadsap_psd.vtex_c` |  |
| Granny Long Legs | Granny Long Legs: jump high and kick anyone nearby with spirit-charged legs. | `strafing_run_psd.vtex_c` |  |
| Baba's Brew | Brew Baba's special pot of tea, then throw it to ignite or silence enemies, or shield allies | `teapot_brew_psd.vtex_c` |  |
| Feed The Birds | Cover targets in front of you with bread, adding a stacking X . When the channel ends a flock of pigeons bombard the… | `pigeon_launcher_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_wip/baba/baba.vmdl` | 104 units tall |
| Animation skeleton | `models/heroes_wip/baba/baba.vnmskel` | |
| Animation clips | `models/heroes_wip/baba/clips/` | 249 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/baba_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/baba_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/baba_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/baba_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/baba_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/baba_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/baba_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/baba_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/baba/pigeon_hex_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/baba/pigeon_launcher_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/baba/strafing_run_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/baba/teapot_brew_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/baba/threadsap_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/baba.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/baba_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/baba.vpk` | |
| Ability model | `models/abilities/baba_model_teapot_fx.vmdl` | |
| Effects | `particles/abilities/baba/` | |
| Effects | `particles/weapon_fx/baba/` | |
| Name and tag text | `Steam_RP_hero_baba`, `hero_baba:n`, `hero_baba_search:n`, `hero_baba_sort:n`, `Citadel_Baba_HeroTag_1`, `Citadel_Baba_HeroTag_2`, `Citadel_Baba_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Baba's own 249 clips on whatever model is installed, so a replacement is rigged to Baba's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `out_of_combat_idle`, `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`Throw_UpperBody`, `melee_UpperBody`, `Reload_Idle`, `Reload_Haspath`, `threadsap_Haspath`, `UpperBodyReplace_Turn`, `Reload_Crouch`, `Reload_HasPathCrouch`, `Parry_UpperBody`, `hexing_brew_Haspath`, `melee_FullBody`, `melee_BenchAndBabaPelvis`, `UpperBodyReplace_Asleep`, `Cast_UpperBody`, `UpperBodyReplace_Slide`, `Reload_Slide`, `Reload_InAir_01`, `Reload_InAir`, `Pigeon_only`, `Pigeon_only_01`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model baba assets/model.glb
dlmod select baba bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons baba 1=assets/icon_....png
dlmod image panorama/images/heroes/baba_card_psd.vtex_c assets/card.png
dlmod prop models/abilities/baba_model_teapot_fx.vmdl assets/prop_baba_model_teapot_fx.glb
dlmod paint baba
```
