# Deadman Danny (`deadpack`)

Weapon: Chimera Peace Bringer. Tags: Reckless, High-flying, Explosive.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/deadman-danny
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Strix Salvo | Launch a salvo of spirit missiles that lodge into the ground along a line. The missiles explode after a delay or upon… | `deadpack_salvo_psd.vtex_c` | `models/abilities/deadpack_missile_model.vmdl` |
| Chimera Mk1 | Hold X to fly, building up heat. When max heat is reached, your jetpack will begin to overheat. Overheating: After a… | `deadpack_chimera_psd.vtex_c` |  |
| Jettison Fuel | Spray fuel dealing X and shredding X, building up to a huge backblast that deals X and knocks you backwards. Vents the… | `deadpack_thermalmass_psd.vtex_c` |  |
| Close The Deal | Grab an enemy and give them a hands-on demonstration, flying high into the air before crashing back to the ground. You… | `deadpack_flight_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_wip/deadpack/deadpack.vmdl` | 137 units tall |
| Animation skeleton | `models/heroes_wip/deadpack/deadpack.vnmskel` | |
| Animation clips | `models/heroes_wip/deadpack/clips/` | 259 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/deadpack_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/deadpack_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/deadpack_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/deadpack_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/deadpack_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/deadpack_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/deadpack_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/deadpack_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/deadpack/deadpack_chimera_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/deadpack/deadpack_flight_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/deadpack/deadpack_salvo_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/deadpack/deadpack_thermalmass_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/deadpack/status_marked_psd.vtex_c` | 256x256 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/deadpack.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/deadpack_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/deadpack.vpk` | |
| Ability model | `models/abilities/deadpack_globules.vmdl` | |
| Ability model | `models/abilities/deadpack_missile_model.vmdl` | |
| Effects | `particles/abilities/deadpack/` | |
| Name and tag text | `Steam_RP_hero_deadpack`, `hero_deadpack:n`, `hero_deadpack_search:n`, `hero_deadpack_sort:n`, `Citadel_Deadpack_HeroTag_1`, `Citadel_Deadpack_HeroTag_2`, `Citadel_Deadpack_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Deadman Danny's own 259 clips on whatever model is installed, so a replacement is rigged to Deadman Danny's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `out_of_combat_idle`, `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`melee_UpperBody`, `Reload_Idle`, `Throw_UpperBody`, `UpperBodyReplace_AirDash`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_GroundDash`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Shopping`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `Parry_UpperBody`, `Reload_Crouch`, `Reload_HasPathCrouch`, `Reload_Haspath`, `Reload_InAir`, `Reload_Slide`, `Reload_Zoomed`, `FullBody`, `ArmsAndGun`, `Cast_UpperBody`, `JetpackShakingCrouched`, `Ult_Intro_Pose`, `FuelGauge`, `arm_R`, `wings`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model deadpack assets/model.glb
dlmod select deadpack bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons deadpack 1=assets/icon_....png
dlmod image panorama/images/heroes/deadpack_card_psd.vtex_c assets/card.png
dlmod prop models/abilities/deadpack_globules.vmdl assets/prop_deadpack_globules.glb
dlmod paint deadpack
```
