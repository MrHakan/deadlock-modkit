# Billy (`punkgoat`)

Weapon: Biquette. Tags: Punk, Chaotic, G.O.A.T..

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/billy
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Bashdown | Slam your bat into the ground, X enemies down and dealing X. The slam creates a shockwave that deals X and applies X. | `goat_sigilslam_psd.vtex_c` |  |
| Rising Ram | Charge head-first into an enemy and send them into the air along with Billy. Cooldown reduced by 50% on impact. | `goat_risingram_psd.vtex_c` |  |
| Blasted | Passive: Melee hits restore ammo and inflict wrecked on the victim for Xs. | `goat_blasted_psd.vtex_c` |  |
| Chain Gang | Chain nearby enemies to you. Chained enemies cannot use movement abilities and receive a heavy X when they pull on the… | `goat_chaingang_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_wip/punkgoat/punkgoat.vmdl` | 150 units tall |
| Animation skeleton | `models/heroes_wip/punkgoat/punkgoat.vnmskel` | |
| Animation clips | `models/heroes_wip/punkgoat/clips/` | 243 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/punkgoat_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/punkgoat_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/punkgoat_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/punkgoat_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/punkgoat_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/punkgoat_mm_psd.vtex_c` | 50x51 |
| Background | `panorama/images/heroes/backgrounds/billy_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/punkgoat_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/punkgoat/goat_blasted_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/punkgoat/goat_chaingang_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/punkgoat/goat_risingram_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/punkgoat/goat_sigilslam_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/punkgoat/punkgoat_capricious_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/billy.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/billy_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/punkgoat.vpk` | |
| Effects | `particles/abilities/punkgoat/` | |
| Effects | `particles/weapon_fx/punkgoat/` | |
| Name and tag text | `Steam_RP_hero_punkgoat`, `hero_punkgoat:n`, `hero_punkgoat_search:n`, `hero_punkgoat_sort:n`, `Citadel_Punkgoat_HeroTag_1`, `Citadel_Punkgoat_HeroTag_2`, `Citadel_Punkgoat_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Billy's own 243 clips on whatever model is installed, so a replacement is rigged to Billy's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`melee_UpperBody`, `Parry_UpperBody`, `Reload_Crouch`, `Reload_Haspath`, `Reload_HasPathCrouch`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `Throw_UpperBody`, `Turn_LowerBody`, `Cast_UpperBody`, `arm_L`, `ability_capricious_mask`, `all_upper_body`, `arm_L_ONLY`, `ult_tether_pull`, `fidget_chewing`, `fidget_ear_flicks`, `arm_R`, `UpperBodyReplace_Turn`, `both_arms`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model punkgoat assets/model.glb
dlmod select punkgoat bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons punkgoat 1=assets/icon_....png
dlmod image panorama/images/heroes/punkgoat_card_psd.vtex_c assets/card.png
dlmod paint punkgoat
```
