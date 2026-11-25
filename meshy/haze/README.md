# Haze (`haze`)

Sneaks in and sprays bullets. When Haze fixates on a target she can hold her own in a firefight. She prefers to create one sided exchanges, avoiding damage then picking off unsuspecting targets up close.

Weapon: Censure & Penance. Tags: Assassin, Stealthy, Lethal.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/haze
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Sleep Dagger | Put an enemy to sleep | `haze_sleep_dagger_psd.vtex_c` | `models/abilities/haze_dagger_model.vmdl` |
| Smoke Bomb | Fade out of sight | `haze_smoke_bomb_psd.vtex_c` |  |
| Fixation | Deal increasing bullet damage to a target | `haze_fixation_psd.vtex_c` |  |
| Bullet Dance | Unleash a flurry of bullets around you | `haze_bullet_flurry_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_staging/haze/haze.vmdl` | 95 units tall |
| Animation skeleton | `models/heroes_staging/haze/haze.vnmskel` | |
| Animation clips | `models/heroes_staging/haze/clips/` | 254 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/haze_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/haze_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/haze_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/haze_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/haze_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/haze_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/haze_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/haze_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/haze/haze_bullet_flurry_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/haze/haze_fixation_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/haze/haze_sleep_dagger_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/haze/haze_smoke_bomb_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/haze.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/haze_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/haze.vpk` | |
| Ability model | `models/abilities/haze_dagger_model.vmdl` | |
| Effects | `particles/abilities/haze/` | |
| Effects | `particles/weapon_fx/haze/` | |
| Name and tag text | `Steam_RP_hero_haze`, `hero_haze:n`, `hero_haze_search:n`, `hero_haze_sort:n`, `Citadel_Haze_HeroTag_1`, `Citadel_Haze_HeroTag_2`, `Citadel_Haze_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Haze's own 254 clips on whatever model is installed, so a replacement is rigged to Haze's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`shoot_R`, `shoot_L`, `Cast_UpperBody`, `FullBody`, `melee_dash`, `melee_UpperBody`, `Parry_UpperBody`, `Reload_Haspath`, `Reload_Idle`, `Root`, `Throw_UpperBody`, `UpperBodyReplace_AirDash`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_Asleep_Channeling`, `UpperBodyReplace_GroundDash`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `zero`, `Reload_Crouch`, `Reload_HasPathCrouch`, `Reload_Slide`, `Reload_InAir`, `Ability_Dagger`, `Holster_Guns`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model haze assets/model.glb
dlmod select haze bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons haze 1=assets/icon_....png
dlmod image panorama/images/heroes/haze_card_psd.vtex_c assets/card.png
dlmod prop models/abilities/haze_dagger_model.vmdl assets/prop_haze_dagger_model.glb
dlmod paint haze
```
