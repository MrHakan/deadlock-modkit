# Pocket (`synth`)

Infiltrates and disrupts the backline. With a powerful shotgun and a thin frame, Pocket relies on their ability to briefly escape into a mystic suitcase and teleport via a flying cloak in order to survive close encounters.

Weapon: The Black Sheep. Tags: Trickster, Burst Damage, Frogs.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/pocket
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Barrage | Air glide while firing multiple projectiles | `synth_barrage_psd.vtex_c` |  |
| Flying Cloak | Launch a damaging cloak that you can teleport to | `synth_plasma_flux_psd.vtex_c` |  |
| Enchanter's Satchel | Briefly escape into your suitcase, then deal damage | `synth_pulse_psd.vtex_c` | `models/abilities/pocket_briefcase_model.vmdl` |
| Affliction | Apply damage over time to enemies nearby | `synth_affliction_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_wip/pocket/pocket.vmdl` | 94 units tall |
| Animation skeleton | `models/heroes_staging/synth/synth.vnmskel` | |
| Animation clips | `models/heroes_staging/synth/clips/` | 178 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/synth_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/synth_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/synth_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/synth_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/synth_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/synth_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/pocket_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/synth_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/synth/synth_affliction_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/synth/synth_barrage_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/synth/synth_grasp_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/synth/synth_plasma_flux_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/synth/synth_pulse_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/pocket.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/pocket_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/pocket.vpk` | |
| Ability model | `models/abilities/pocket_briefcase_model.vmdl` | |
| Ability model | `models/abilities/pocket_frog_small.vmdl` | |
| Ability model | `models/abilities/synth_deployable.vmdl` | |
| Effects | `particles/abilities/pocket/` | |
| Effects | `particles/abilities/synth/` | |
| Effects | `particles/weapon_fx/pocket/` | |
| Effects | `particles/weapon_fx/synth/` | |
| Name and tag text | `Steam_RP_hero_synth`, `hero_synth:n`, `hero_synth_search:n`, `Citadel_Synth_HeroTag_1`, `Citadel_Synth_HeroTag_2`, `Citadel_Synth_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Pocket's own 178 clips on whatever model is installed, so a replacement is rigged to Pocket's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `out_of_combat_idle`, `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `melee_UpperBody`, `Parry_UpperBody`, `Reload_Crouch`, `Reload_Haspath`, `Reload_HasPathCrouch`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `Throw_UpperBody`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `ability_barrage`, `zero`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model synth assets/model.glb
dlmod select synth bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons synth 1=assets/icon_....png
dlmod image panorama/images/heroes/synth_card_psd.vtex_c assets/card.png
dlmod prop models/abilities/pocket_briefcase_model.vmdl assets/prop_pocket_briefcase_model.glb
dlmod paint synth
```
