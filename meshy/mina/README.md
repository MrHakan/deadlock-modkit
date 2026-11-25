# Mina (`vampirebat`)

Weapon: New Money. Tags: Harasser, Nimble, Vexing.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/mina
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Rake | Active: Slash with your umbrella, dealing X, increasing with their missing health. Receive a X for every target killed… | `vampirebat_rake_psd.vtex_c` |  |
| Sanguine Retreat | Briefly disperse, becoming untargetable and flying to a target location. Can be recast within a brief window. | `vampirebat_sanguine_retreat_psd.vtex_c` |  |
| Love Bites | Your bullets apply additional X. Dealing damage with your abilities builds up to a vicious bite, dealing a burst of X… | `vampirebat_love_bites_psd.vtex_c` |  |
| Nox Nostra | Active: Unleash a cloud of bats that seek out targets, each dealing X and applying X. Passive: Triggering Love Bites… | `vampirebat_nox_nostra_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_wip/vampirebat/vampirebat.vmdl` | 101 units tall |
| Animation skeleton | `models/heroes_wip/vampirebat/vampirebat.vnmskel` | |
| Animation clips | `models/heroes_wip/vampirebat/clips/` | 211 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/vampirebat_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/vampirebat_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/vampirebat_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/vampirebat_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/vampirebat_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/vampirebat_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/mina_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/vampirebat_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/vampirebat/love_bites_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/vampirebat/vampirebat_love_bites_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/vampirebat/vampirebat_nox_nostra_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/vampirebat/vampirebat_rake_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/vampirebat/vampirebat_sanguine_retreat_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/mina.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/mina_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/vampirebat.vpk` | |
| Effects | `particles/abilities/vampirebat/` | |
| Effects | `particles/weapon_fx/vampirebat/` | |
| Name and tag text | `Steam_RP_hero_vampirebat`, `hero_vampirebat:n`, `hero_vampirebat_search:n`, `hero_vampirebat_sort:n`, `Citadel_VampireBat_HeroTag_1`, `Citadel_VampireBat_HeroTag_2`, `Citadel_VampireBat_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Mina's own 211 clips on whatever model is installed, so a replacement is rigged to Mina's bones rather than given new animations. What the game does to those clips matters:

- Melee, reloads and casts are laid over the movement through bone masks (`Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `Reload_Haspath`, `Reload_HasPathCrouch`, `melee_UpperBody`, `Reload_Crouch`, `Parry_UpperBody`, `melee_dash`, `FullBody`, `Cast_UpperBody`, `root`, `Throw_UpperBody`, `Turn_LowerBody`, `UpperBodyReplace_AirDash`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_Asleep_Channeling`, `UpperBodyReplace_GroundDash`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Shopping`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `zero`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model vampirebat assets/model.glb
dlmod select vampirebat bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons vampirebat 1=assets/icon_....png
dlmod image panorama/images/heroes/vampirebat_card_psd.vtex_c assets/card.png
dlmod paint vampirebat
```
