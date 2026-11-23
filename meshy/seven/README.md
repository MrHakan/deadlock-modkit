# Seven (`gigawatt`)

Electrocutes crowds of enemies. Seven thrives in a skirmish waiting for the time to strike. Then he rolls into the fight like a storm and batters his enemies with a cascade of lightning.

Weapon: Cold Calculus. Tags: High Voltage, Merciless, Area Denial.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/seven
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Lightning Ball | Launch a damaging ball of lightning | `giga_ball_psd.vtex_c` |  |
| Static Charge | Apply a delayed area stun to an enemy hero | `giga_static_psd.vtex_c` |  |
| Power Surge | Create electric arcs with your shots | `giga_chain_psd.vtex_c` |  |
| Storm Cloud | Fry all enemies in a large area | `giga_storm_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_staging/gigawatt_prisoner/gigawatt_prisoner.vmdl` | 113 units tall |
| Animation skeleton | `models/heroes_staging/gigawatt_prisoner/gigawatt_prisoner.vnmskel` | |
| Animation clips | `models/heroes_staging/gigawatt_prisoner/clips/` | 190 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/gigawatt_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/gigawatt_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/gigawatt_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/gigawatt_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/gigawatt_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/gigawatt_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/seven_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/gigawatt_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/giga_ball_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/giga_chain_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/giga_static_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/giga_storm_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/seven.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/seven_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/gigawatt.vpk` | |
| Effects | `particles/abilities/gigawatt/` | |
| Effects | `particles/weapon_fx/gigawatt/` | |
| Name and tag text | `Steam_RP_hero_gigawatt`, `hero_gigawatt:n`, `hero_gigawatt_search:n`, `Citadel_Gigawatt_HeroTag_1`, `Citadel_Gigawatt_HeroTag_2`, `Citadel_Gigawatt_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Seven's own 190 clips on whatever model is installed, so a replacement is rigged to Seven's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `out_of_combat_idle`, `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `melee_dash`, `melee_UpperBody`, `Parry_UpperBody`, `Reload_Crouch`, `Reload_Haspath`, `Reload_Crouch_ALT`, `Reload_HasPath_ALT`, `Reload_HasPathCrouch`, `Reload_HasPathCrouch_ALT`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `Reload_Zoomed`, `Throw_UpperBody`, `Turn_LowerBody`, `UpperBodyReplace_AirDash`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_Asleep_Channeling`, `UpperBodyReplace_GroundDash`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `FullBody`, `NULL`, `Power_Surge_UpperBody`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model gigawatt assets/model.glb
dlmod select gigawatt bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons gigawatt 1=assets/icon_....png
dlmod image panorama/images/heroes/gigawatt_card_psd.vtex_c assets/card.png
dlmod paint gigawatt
```
