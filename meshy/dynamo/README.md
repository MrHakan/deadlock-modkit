# Dynamo (`dynamo`)

Locks down the enemy team. Dynamo keeps himself and his allies healthy while waiting for his moment. Few things can warp a team fight more than a well-coordinated use of Singularity.

Weapon: The Vonnegun. Tags: Teamplay, Initiator, Clutch.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/dynamo
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Kinetic Pulse | Release a ground pulse that knocks enemies into the air | `sumo_stomp_psd.vtex_c` |  |
| Quantum Entanglement | Blink in and out of existence | `sumo_quantum_psd.vtex_c` |  |
| Rejuvenating Aurora | Heal yourself and those around you | `sumo_pork_bun_psd.vtex_c` |  |
| Singularity | Pull in and stun the enemy team | `sumo_vacuum_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_wip/dynamo/dynamo.vmdl` | 103 units tall |
| Animation skeleton | `models/heroes_wip/dynamo/dynamo.vnmskel` | |
| Animation clips | `models/heroes_wip/dynamo/clips/` | 194 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/sumo_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/sumo_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/sumo_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/sumo_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/sumo_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/sumo_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/dynamo_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/dynamo_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/sumo/sumo_charge_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/sumo/sumo_pork_bun_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/sumo/sumo_quantum_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/sumo/sumo_stomp_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/sumo/sumo_vacuum_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/dynamo.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/dynamo_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/prof_dynamo.vpk` | |
| Effects | `particles/abilities/dynamo/` | |
| Effects | `particles/abilities/sumo/` | |
| Effects | `particles/weapon_fx/dynamo/` | |
| Effects | `particles/weapon_fx/sumo/` | |
| Name and tag text | `Steam_RP_hero_dynamo`, `hero_dynamo:n`, `hero_dynamo_search:n`, `Citadel_Dynamo_HeroTag_1`, `Citadel_Dynamo_HeroTag_2`, `Citadel_Dynamo_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Dynamo's own 194 clips on whatever model is installed, so a replacement is rigged to Dynamo's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `FullBody`, `melee_dash`, `Parry_UpperBody`, `Reload_Crouch`, `Reload_GunOnly`, `Reload_Haspath`, `Reload_HasPathCrouch`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `Reload_Zoomed`, `Root`, `Throw_UpperBody`, `Turn_LowerBody`, `UpperBodyReplace_AirDash`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_Asleep_Channeling`, `UpperBodyReplace_GroundDash`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Shopping`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `zero`, `melee_UpperBody`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model dynamo assets/model.glb
dlmod select dynamo bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons dynamo 1=assets/icon_....png
dlmod image panorama/images/heroes/sumo_card_psd.vtex_c assets/card.png
dlmod paint dynamo
```
