# Celeste (`unicorn`)

Weapon: Carny Life. Tags: Performer, Disruptive, Dazzling.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/celeste
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Light Eater | Blast enemies in a cone in front of you with a flare of light. Blasted enemies take X when attacked by X and provides… | `unicorn_flare_psd.vtex_c` |  |
| Dazzling Trick | Surround yourself in a protective prism. If the barrier is destroyed, it silences nearby enemies and deals a portion of… | `unicorn_shield_psd.vtex_c` |  |
| Radiant Daggers | Call down a beam of light from the sky. After a short duration, the beam will fully form, causing an explosion that… | `unicorn_luminousflux_psd.vtex_c` | `models/abilities/unicorn_star_dagger.vmdl` |
| Shining Wonder | Launch a deadly orb of light that deals X and applies X and reduces Dash Distance on impact. The orb then bounces to… | `unicorn_orb_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_wip/unicorn/unicorn.vmdl` | 117 units tall |
| Animation skeleton | `models/heroes_wip/unicorn/unicorn.vnmskel` | |
| Animation clips | `models/heroes_wip/unicorn/clips/` | 225 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/unicorn_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/unicorn_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/unicorn_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/unicorn_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/unicorn_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/unicorn_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/unicorn_bg_psd.vtex_c` | 4096x2312 |
| Ability icon | `panorama/images/hud/abilities/unicorn/unicorn_flare_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/unicorn/unicorn_luminousflux_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/unicorn/unicorn_orb_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/unicorn/unicorn_shield_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/unicorn.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/unicorn_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/unicorn.vpk` | |
| Ability model | `models/abilities/unicorn_star_dagger.vmdl` | |
| Effects | `particles/abilities/unicorn/` | |
| Effects | `particles/weapon_fx/unicorn/` | |
| Name and tag text | `Steam_RP_hero_unicorn`, `hero_unicorn:n`, `hero_unicorn_search:n`, `Citadel_Unicorn_HeroTag_1`, `Citadel_Unicorn_HeroTag_2`, `Citadel_Unicorn_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Celeste's own 225 clips on whatever model is installed, so a replacement is rigged to Celeste's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `out_of_combat_idle`, `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `FullBody`, `melee_dash`, `melee_UpperBody`, `Parry_UpperBody`, `Reload_Crouch`, `Reload_Haspath`, `Reload_HasPathCrouch`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `Reload_Zoomed`, `root`, `shoot_crouch`, `shoot_haspath`, `shoot_Idle`, `shoot_inair`, `shoot_slide`, `shoot_turn`, `Throw_UpperBody`, `Turn_LowerBody`, `UpperBodyReplace_AirDash`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_Asleep_Channeling`, `UpperBodyReplace_GroundDash`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Shopping`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `zero`, `luminous_hasPath`, `orb_hasPath`, `Prismatic_hasPath`, `Reload_Idle_ALT`, `Ability_RadiantBlast_Turn`, `shoot_dash_air`, `Reload_Crouch_ALT`, `Reload_HasPathCrouch_ALT`, `Reload_InAir_ALT`, `Baton`, `Reload_HasPath_ALT`, `Reload_Slide_ALT`, `Ability_Radiantblast_dash`, `Ability_Radiantblast_inair`, `Reload_Additive`, `Dash_Air_Hand`, `Prismatic_InAir`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model unicorn assets/model.glb
dlmod select unicorn bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons unicorn 1=assets/icon_....png
dlmod image panorama/images/heroes/unicorn_card_psd.vtex_c assets/card.png
dlmod prop models/abilities/unicorn_star_dagger.vmdl assets/prop_unicorn_star_dagger.glb
dlmod paint unicorn
```
