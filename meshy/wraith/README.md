# Wraith (`wraith`)

Melts isolated targets. Excelling at one-on-one combat, Wraith isolates targets, eliminating them with ruthless efficiency. Then teleporting away before anyone can retaliate.

Weapon: The Rake. Tags: Duelist, Isolator, Telekinetic.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/wraith
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Card Trick | Throw homing cards | `wraith_card_trick_psd.vtex_c` |  |
| Project Mind | Teleport to the targeted location | `wraith_teleport_psd.vtex_c` |  |
| Full Auto | Boost your fire rate | `wraith_aura_psd.vtex_c` |  |
| Telekinesis | Lift an enemy, interrupting and silencing them | `wraith_lift_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_wip/wraith/wraith.vmdl` | 116 units tall |
| Animation skeleton | `models/heroes_wip/wraith/wraith.vnmskel` | |
| Animation clips | `models/heroes_wip/wraith/clips/` | 195 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/wraith_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/wraith_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/wraith_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/wraith_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/wraith_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/wraith_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/wraith_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/wraith_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/wraith_aura_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/wraith_card_trick_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/wraith_daggers_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/wraith_lift_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/wraith_teleport_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/wraith.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/wraith_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/wraith.vpk` | |
| Effects | `particles/abilities/wraith/` | |
| Effects | `particles/weapon_fx/wraith/` | |
| Name and tag text | `Steam_RP_hero_wraith`, `hero_wraith:n`, `hero_wraith_search:n`, `Citadel_Wraith_HeroTag_1`, `Citadel_Wraith_HeroTag_2`, `Citadel_Wraith_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Wraith's own 195 clips on whatever model is installed, so a replacement is rigged to Wraith's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `melee_UpperBody`, `Parry_UpperBody`, `Reload_Crouch`, `Reload_Haspath`, `Reload_HasPath_ALT`, `Reload_HasPathCrouch`, `Reload_HasPathCrouch_ALT`, `Reload_Idle`, `Reload_InAir`, `Reload_InAir_ALT`, `Reload_Slide`, `Throw_UpperBody`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `FullAuto_HasPath_UpperBody`, `CardTrick_Add_UpperBody`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model wraith assets/model.glb
dlmod select wraith bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons wraith 1=assets/icon_....png
dlmod image panorama/images/heroes/wraith_card_psd.vtex_c assets/card.png
dlmod paint wraith
```
