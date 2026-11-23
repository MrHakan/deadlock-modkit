# Shiv (`shiv`)

Bleeds out foes, then goes in for a kill. Shiv has zero interest in playing fair, which is why he relies on hit and run tactics to weaken enemies before he dives in for the kill.

Weapon: Busted Flush. Tags: Rage, Bleed, Repeat.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/shiv
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Serrated Knives | Apply stacking bleed to the target | `shiv_toss_psd.vtex_c` |  |
| Slice and Dice | Dash ahead, damaging all enemies along the path | `shiv_flash_psd.vtex_c` |  |
| Bloodletting | Defer a portion of damage taken | `shiv_bloodletting_psd.vtex_c` |  |
| Killing Blow | Travel to a low health hero and instantly kill it | `shiv_killing_blow_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_staging/shiv/shiv.vmdl` | 108 units tall |
| Animation skeleton | `models/heroes_staging/shiv/shiv.vnmskel` | |
| Animation clips | `models/heroes_staging/shiv/clips/` | 182 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/shiv_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/shiv_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/shiv_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/shiv_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/shiv_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/shiv_mm_psd.vtex_c` | 50x51 |
| Background | `panorama/images/heroes/backgrounds/shiv_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/shiv_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/shiv/shiv_bloodletting_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/shiv/shiv_flash_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/shiv/shiv_gather_foes_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/shiv/shiv_killing_blow_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/shiv/shiv_prison_dance_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/shiv/shiv_toss_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/shiv/shiv_trophy_kill_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/shiv.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/shiv_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/shiv.vpk` | |
| Effects | `particles/abilities/shiv/` | |
| Effects | `particles/weapon_fx/shiv/` | |
| Name and tag text | `Steam_RP_hero_shiv`, `hero_shiv:n`, `hero_shiv_search:n`, `hero_shiv_sort:n`, `Citadel_Shiv_HeroTag_1`, `Citadel_Shiv_HeroTag_2`, `Citadel_Shiv_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Shiv's own 182 clips on whatever model is installed, so a replacement is rigged to Shiv's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `FullBody`, `melee_dash`, `melee_UpperBody`, `Parry_UpperBody`, `Reload_GunOnly`, `Reload_Idle`, `Root`, `Throw_UpperBody`, `Turn_LowerBody`, `UpperBodyReplace_AirDash`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_Asleep_Channeling`, `UpperBodyReplace_GroundDash`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Shopping`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `zero`, `Reload_InAir`, `Reload_Haspath`, `Reload_Slide`, `Reload_Zoomed`, `Reload_HasPathCrouch`, `Reload_Crouch`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model shiv assets/model.glb
dlmod select shiv bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons shiv 1=assets/icon_....png
dlmod image panorama/images/heroes/shiv_card_psd.vtex_c assets/card.png
dlmod paint shiv
```
