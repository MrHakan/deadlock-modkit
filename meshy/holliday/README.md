# Holliday (`astro`)

Bounces about dispensing trick shots. Holliday depends on her skills as a crack shot, shooting things she's bounced into the air. With good aim she can lay out heavy damage. Her movement skills let her isolate her enemies or beat a hasty retreat.

Weapon: Doc's Orders. Tags: Crackshot, Explosive, Apprehender.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/holliday
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Powder Keg | Throw a barrel that explodes when shot | `holliday_powder_keg_psd.vtex_c` | `models/abilities/astro_explosive.vmdl` |
| Bounce Pad | Send yourself and others flying | `holliday_bounce_pad_psd.vtex_c` | `models/abilities/astro_explosive.vmdl` |
| Crackshot | Deal bonus damage with headshots | `holliday_crackshot_psd.vtex_c` |  |
| Spirit Lasso | Grab an enemy and drag them behind you | `holliday_spirit_lasso_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_staging/astro/astro.vmdl` | 92 units tall |
| Animation skeleton | `models/heroes_staging/astro/astro.vnmskel` | |
| Animation clips | `models/heroes_staging/astro/clips/` | 186 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/astro_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/astro_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/astro_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/astro_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/astro_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/astro_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/astro_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/astro_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/astro/bounce_pad_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/astro/gravity_lasso_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/astro/holliday_bounce_pad_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/astro/holliday_crackshot_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/astro/holliday_powder_keg_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/astro/holliday_spirit_lasso_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/holliday.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/holliday_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/astro.vpk` | |
| Ability model | `models/abilities/astro_duck_target.vmdl` | |
| Ability model | `models/abilities/astro_explosive.vmdl` | |
| Effects | `particles/abilities/astro/` | |
| Effects | `particles/weapon_fx/astro/` | |
| Name and tag text | `Steam_RP_hero_astro`, `hero_astro:n`, `hero_astro_search:n`, `Citadel_Astro_HeroTag_1`, `Citadel_Astro_HeroTag_2`, `Citadel_Astro_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Holliday's own 186 clips on whatever model is installed, so a replacement is rigged to Holliday's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`, `reload_stand_idle`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `Throw_UpperBody`, `UpperBodyReplace_Turn`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Slide`, `Reload_Idle`, `Reload_Haspath`, `Reload_Slide`, `Reload_InAir`, `UpperBodyReplace_Asleep`, `Parry_UpperBody`, `melee_UpperBody`, `Reload_HasPath_ALT`, `Reload_InAir_ALT`, `Reload_Crouch`, `Reload_HasPathCrouch`, `Reload_HasPathCrouch_ALT`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model astro assets/model.glb
dlmod select astro bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons astro 1=assets/icon_....png
dlmod image panorama/images/heroes/astro_card_psd.vtex_c assets/card.png
dlmod prop models/abilities/astro_duck_target.vmdl assets/prop_astro_duck_target.glb
dlmod paint astro
```
