# McGinnis (`forge`)

Controls the battle with combat turrets. Precise placement is everything for McGinnis, whether she's creating a killzone or splitting the enemy team. She can help keep her team running making them hard to dislodge when entrenched.

Weapon: Services Rendered. Tags: Inventor, Support, Disruption.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/mcginnis
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Mini Turret | Deploy an automated turret | `engineer_turret_psd.vtex_c` |  |
| Medicinal Specter | Deploy a healing spirit | `engineer_resupply_psd.vtex_c` |  |
| Spectral Wall | Conjure a wall to block your foes | `engineer_fissure_2_psd.vtex_c` | `models/abilities/engineer_wall.vmdl`, `models/abilities/engineer_wall_physics.vmdl` |
| Heavy Barrage | Bombard an area with rockets | `engineer_rockets_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_wip/mcginnis/mcginnis.vmdl` | 109 units tall |
| Animation skeleton | `models/heroes_wip/mcginnis/mcginnis.vnmskel` | |
| Animation clips | `models/heroes_wip/mcginnis/clips/` | 268 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/engineer_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/engineer_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/engineer_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/engineer_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/engineer_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/engineer_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/mcginnis_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/engineer_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/engineer_fissure_2_activate_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/engineer_fissure_2_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/engineer_resupply_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/engineer_rockets_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/engineer_turret_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/mcginnis.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/mcginnis_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/forge.vpk` | |
| Ability model | `models/abilities/engineer_wall.vmdl` | |
| Ability model | `models/abilities/engineer_wall_physics.vmdl` | |
| Effects | `particles/abilities/engineer/` | |
| Effects | `particles/abilities/mcginnis/` | |
| Effects | `particles/weapon_fx/engineer/` | |
| Effects | `particles/weapon_fx/forge/` | |
| Effects | `particles/weapon_fx/mcginnis/` | |
| Name and tag text | `Steam_RP_hero_forge`, `hero_forge:n`, `hero_forge_search:n`, `Citadel_Engineer_HeroTag_1`, `Citadel_Engineer_HeroTag_2`, `Citadel_Engineer_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays McGinnis's own 268 clips on whatever model is installed, so a replacement is rigged to McGinnis's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`Ability_idle`, `Cast_UpperBody`, `FullBody`, `melee_UpperBody`, `Parry_UpperBody`, `Reload_Crouch`, `Reload_GunOnly`, `Reload_Haspath`, `Reload_HasPathCrouch`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `Reload_Zoomed`, `root`, `Throw_UpperBody`, `Turn_LowerBody`, `UpperBody_Fidget`, `UpperBodyReplace_AirDash`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_Asleep_Channeling`, `UpperBodyReplace_Shopping`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `zero`, `Rocket_Pack`, `Rocket_Pack_Start`, `Ability_UpperBody`, `Ability_Upperbody_Toss`, `Ability_idle_sentry`, `Ability_idle_arm_adjust`, `Ability_Upperbody_HasPath`, `UpperBodyReplace_InAir`, `UpperBodyReplace_GroundDash`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model forge assets/model.glb
dlmod select forge bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons forge 1=assets/icon_....png
dlmod image panorama/images/heroes/engineer_card_psd.vtex_c assets/card.png
dlmod prop models/abilities/engineer_wall.vmdl assets/prop_engineer_wall.glb
dlmod paint forge
```
