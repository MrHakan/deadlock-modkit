# Grey Talon (`orion`)

Delivers long range skill-shots. A cunning predator, Grey Talon uses a combination of deadly accuracy, traps, and clever positioning to bring down the toughest of opponents.

Weapon: The Mourning Comes. Tags: Precision, Hunter, Area Denial.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/grey-talon
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Charged Shot | Charge up a long range shot | `archer_charged_shot_psd.vtex_c` |  |
| Rain of Arrows | Jump high up in the air and unleash multishots | `archer_power_jump_psd.vtex_c` |  |
| Spirit Snare | Place a trap that restricts enemy movement | `imobolize_trap_psd.vtex_c` |  |
| Guided Owl | Fly your owl to damage enemies from afar | `archer_guided_arrow_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_staging/archer/archer.vmdl` | 118 units tall |
| Animation skeleton | `models/heroes_staging/archer/archer.vnmskel` | |
| Animation clips | `models/heroes_staging/archer/clips/` | 222 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/archer_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/archer_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/archer_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/archer_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/archer_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/archer_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/grey_talon_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/archer_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/archer/archer_charged_shot_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/archer/archer_guided_arrow_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/archer/archer_power_jump_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/archer/imobolize_trap_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/grey_talon.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/grey_talon_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/archer.vpk` | |
| Effects | `particles/abilities/archer/` | |
| Effects | `particles/abilities/orion/` | |
| Effects | `particles/weapon_fx/archer/` | |
| Name and tag text | `Steam_RP_hero_orion`, `hero_orion:n`, `hero_orion_search:n`, `Citadel_Orion_HeroTag_1`, `Citadel_Orion_HeroTag_2`, `Citadel_Orion_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Grey Talon's own 222 clips on whatever model is installed, so a replacement is rigged to Grey Talon's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`FullBody`, `melee_UpperBody`, `Parry_UpperBody`, `Reload_Crouch`, `Reload_Haspath`, `Reload_HasPathCrouch`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `Root`, `Throw_UpperBody`, `UpperBodyReplace_AirDash`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_Asleep_Channeling`, `UpperBodyReplace_GroundDash`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `zero`, `shoot_Idle`, `shoot_run`, `ability_charged_shot`, `ability_guided_arrow`, `Cast_UpperBody`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model orion assets/model.glb
dlmod select orion bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons orion 1=assets/icon_....png
dlmod image panorama/images/heroes/archer_card_psd.vtex_c assets/card.png
dlmod paint orion
```
