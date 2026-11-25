# Mirage (`mirage`)

Times his shots and catches fleeing targets. Mirage can envelop himself in a violent tornado to catch up to enemies and lift them up. His scarabs help him sustain against multiple enemies, and he's always ready to help an ally or punish an enemy anywhere.

Weapon: Promises Kept. Tags: Bodyguard, Traveller, Focused.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/mirage
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Fire Scarabs | Send out scarabs that steal life from enemies | `mirage_fire_beetles_psd.vtex_c` |  |
| Dust Devil | Transform yourself into a whirlwind that lifts enemies | `mirage_tornado_psd.vtex_c` |  |
| Djinn's Mark | Apply marks of escalating damage | `mirage_sand_phantom_psd.vtex_c` |  |
| Traveler | Teleport to an ally or enemy hero | `mirage_teleport_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_staging/mirage_v2/mirage.vmdl` | 104 units tall |
| Animation skeleton | `models/heroes_staging/mirage_v2/mirage.vnmskel` | |
| Animation clips | `models/heroes_staging/mirage_v2/clips/` | 201 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/mirage_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/mirage_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/mirage_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/mirage_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/mirage_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/mirage_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/mirage_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/mirage_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/mirage/mirage_djinns_reach_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/mirage/mirage_fire_beetles_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/mirage/mirage_sand_phantom_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/mirage/mirage_sunfire_cataclysm_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/mirage/mirage_teleport_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/mirage/mirage_tornado_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/mirage.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/mirage_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/mirage.vpk` | |
| Effects | `particles/abilities/mirage/` | |
| Effects | `particles/weapon_fx/mirage/` | |
| Name and tag text | `Steam_RP_hero_mirage`, `hero_mirage:n`, `hero_mirage_search:n`, `hero_mirage_sort:n`, `Citadel_Mirage_HeroTag_1`, `Citadel_Mirage_HeroTag_2`, `Citadel_Mirage_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Mirage's own 201 clips on whatever model is installed, so a replacement is rigged to Mirage's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`, `reload_stand_idle`.
- Melee, reloads and casts are laid over the movement through bone masks (`Reload_Crouch`, `Reload_Haspath`, `Reload_HasPathCrouch`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `UpperBodyReplace_Turn`, `FullBody`, `zero`, `root`, `melee_UpperBody`, `UpperBodyReplace_Asleep`, `Throw_UpperBody`, `Parry_UpperBody`, `Ability_Scarab`, `UpperBodyReplace_InAir`, `Cast_UpperBody`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model mirage assets/model.glb
dlmod select mirage bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons mirage 1=assets/icon_....png
dlmod image panorama/images/heroes/mirage_card_psd.vtex_c assets/card.png
dlmod paint mirage
```
