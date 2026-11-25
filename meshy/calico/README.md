# Calico (`nano`)

Wreaks havoc from the sidelines. Calico uses the shadows to her advantage, weaving in and out of combat to hunt down her enemies one by one.

Weapon: Sweetest Lie. Tags: Tricksy, Slippery, Burst Damage.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/calico
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Gloom Bombs | Throw out cluster bombs | `nano_clustergrenade_psd.vtex_c` |  |
| Leaping Slash | Dash forward and slash enemies | `nano_dash_psd.vtex_c` |  |
| Ava | Take control of Ava | `nano_catform_psd.vtex_c` | `models/abilities/nano_catform_model.vmdl` |
| Return to Shadows | Turn to Shadows, gaining movespeed and dealing delayed damage | `nano_shadow_pulse_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_staging/nano/nano_v2/nano.vmdl` | 95 units tall |
| Animation skeleton | `models/heroes_staging/nano/nano_v2/nano.vnmskel` | |
| Animation clips | `models/heroes_staging/nano/nano_v2/clips/` | 205 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/nano_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/nano_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/nano_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/nano_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/nano_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/nano_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/calico_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/nano_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/nano/nano_catform_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/nano/nano_clustergrenade_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/nano/nano_dash_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/nano/nano_humanform_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/nano/nano_replicate_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/nano/nano_roller_bomb_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/nano/nano_shadow_pulse_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/nano/nano_tunnel_vision_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/nano/nano_warp_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/calico.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/calico_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/nano.vpk` | |
| Ability model | `models/abilities/nano_catform_model.vmdl` | |
| Ability model | `models/abilities/nano_ritual_target_swirl.vmdl` | |
| Effects | `particles/abilities/nano/` | |
| Effects | `particles/weapon_fx/nano/` | |
| Name and tag text | `Steam_RP_hero_nano`, `hero_nano:n`, `hero_nano_search:n`, `Citadel_Nano_HeroTag_1`, `Citadel_Nano_HeroTag_2`, `Citadel_Nano_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Calico's own 205 clips on whatever model is installed, so a replacement is rigged to Calico's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `melee_UpperBody`, `Parry_UpperBody`, `Reload_Crouch`, `Reload_Haspath`, `Reload_HasPath_ALT`, `Reload_HasPathCrouch`, `Reload_HasPathCrouch_ALT`, `Reload_Idle`, `Reload_InAir`, `Reload_InAir_ALT`, `Reload_Slide`, `Throw_UpperBody`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `Cat_Only`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model nano assets/model.glb
dlmod select nano bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons nano 1=assets/icon_....png
dlmod image panorama/images/heroes/nano_card_psd.vtex_c assets/card.png
dlmod prop models/abilities/nano_catform_model.vmdl assets/prop_nano_catform_model.glb
dlmod paint nano
```
