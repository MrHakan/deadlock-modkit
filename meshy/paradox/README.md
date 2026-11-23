# Paradox (`chrono`)

Manipulates Space and Time. Manipulating time lets Paradox outpace an enemy in one on one combat. She thrives by subjecting her enemies to timely swaps into a waiting pulse grenade, her wall, or into the middle of her teammates.

Weapon: Trade Secret. Tags: Calculated, Disruptor, Tactician.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/paradox
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Pulse Grenade | Throw a grenade that pulses, dealing damage and time slowing | `chrono_time_bomb_psd.vtex_c` |  |
| Time Wall | Create a wall that greatly slows projectiles | `chrono_time_wall_psd.vtex_c` |  |
| Kinetic Carbine | Charge up your weapon to deal high damage and timestop | `duo_attack_psd.vtex_c` |  |
| Paradoxical Swap | Swap positions with an enemy hero. | `chrono_swap_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_staging/chrono/chrono.vmdl` | 101 units tall |
| Animation skeleton | `models/heroes_staging/chrono/chrono.vnmskel` | |
| Animation clips | `models/heroes_staging/chrono/clips/` | 226 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/chrono_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/chrono_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/chrono_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/chrono_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/chrono_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/chrono_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/paradox_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/chrono_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/chrono/chrono_swap_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/chrono/chrono_time_bomb_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/chrono/chrono_time_wall_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/duo/duo_attack_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/paradox.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/paradox_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/chrono.vpk` | |
| Effects | `particles/abilities/chrono/` | |
| Effects | `particles/weapon_fx/chrono/` | |
| Name and tag text | `Steam_RP_hero_chrono`, `hero_chrono:n`, `hero_chrono_search:n`, `Citadel_Chrono_HeroTag_1`, `Citadel_Chrono_HeroTag_2`, `Citadel_Chrono_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Paradox's own 226 clips on whatever model is installed, so a replacement is rigged to Paradox's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `melee_UpperBody`, `Parry_UpperBody`, `Reload_Crouch`, `Reload_Haspath`, `Reload_HasPath_ALT`, `Reload_HasPathCrouch`, `Reload_HasPathCrouch_ALT`, `Reload_Idle`, `Reload_InAir`, `Reload_InAir_ALT`, `Reload_Slide`, `Throw_UpperBody`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model chrono assets/model.glb
dlmod select chrono bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons chrono 1=assets/icon_....png
dlmod image panorama/images/heroes/chrono_card_psd.vtex_c assets/card.png
dlmod paint chrono
```
