# Lady Geist (`ghost`)

Sacrifices health then drains it back. Expending health to achieve devastating effects is the source of Lady Geist's power. When running low, she can drain the life out of her rivals.

Weapon: Johann's Gift. Tags: Lifesteal, Self Damage, Fatale.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/lady-geist
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Essence Bomb | Sacrifice some health to launch a damaging bomb | `blood_bomb_psd.vtex_c` |  |
| Life Drain | Steal health from an enemy over time | `life_drain_psd.vtex_c` |  |
| Malice | Stack damage amplification on an enemy | `geist_dagger_psd.vtex_c` |  |
| Soul Exchange | Swap your current health with an enemy hero's | `blood_exchange_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_wip/geist/geist.vmdl` | 105 units tall |
| Animation skeleton | `models/heroes_staging/ghost/ghost.vnmskel` | |
| Animation clips | `models/heroes_staging/ghost/clips/` | 181 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/spectre_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/spectre_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/spectre_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/spectre_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/spectre_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/spectre_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/geist_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/spectre_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/spectre/blood_bomb_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/spectre/blood_exchange_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/spectre/death_tax_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/spectre/geist_dagger_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/spectre/life_drain_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/lady_geist.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/lady_geist_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/geist.vpk` | |
| Effects | `particles/abilities/ghost/` | |
| Effects | `particles/abilities/spectre/` | |
| Effects | `particles/weapon_fx/ghost/` | |
| Name and tag text | `Steam_RP_hero_ghost`, `hero_ghost:n`, `hero_ghost_search:n`, `Citadel_Geist_HeroTag_1`, `Citadel_Geist_HeroTag_2`, `Citadel_Geist_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Lady Geist's own 181 clips on whatever model is installed, so a replacement is rigged to Lady Geist's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`, `reload_stand_idle`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `Reload_Idle`, `Reload_Idle_ALT`, `Reload_InAir`, `Reload_InAir_ALT`, `Reload_Haspath`, `Reload_HasPath_ALT`, `Reload_Slide`, `Reload_Slide_ALT`, `UpperBodyReplace_Asleep`, `LifeDrain_UpperBody`, `Hand_Layer`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `UpperBodyReplace_InAir`, `UpperBodyReplace_AirDash`, `UpperBodyReplace_GroundDash`, `UpperBodyReplace_Shopping`, `Throw_UpperBody`, `Reload_Crouch`, `Reload_HasPathCrouch`, `Parry_UpperBody`, `melee_UpperBody`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model ghost assets/model.glb
dlmod select ghost bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons ghost 1=assets/icon_....png
dlmod image panorama/images/heroes/spectre_card_psd.vtex_c assets/card.png
dlmod paint ghost
```
