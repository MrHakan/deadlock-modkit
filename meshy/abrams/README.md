# Abrams (`atlas`)

Charges into close combat. Abrams has the bulk and sustain to lead from the front, often running into the middle of his foes and watching them scatter. If his enemies waste their fire on him, his backline teammates can lay out damage with impunity.

Weapon: Case Closed. Tags: Tank, Brawler, Bull-Headed.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/abrams
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Siphon Life | Drain health from nearby enemies | `bull_drain_psd.vtex_c` |  |
| Shoulder Charge | Charge into the fray, dragging enemies along | `bull_charge_psd.vtex_c` |  |
| Infernal Resilience | Regenerate some of the damage you receive | `bull_beef_psd.vtex_c` |  |
| Seismic Impact | Smash into enemies from above, stunning them | `bull_jump_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_wip/abrams/abrams.vmdl` | 131 units tall |
| Animation skeleton | `models/heroes_wip/abrams/abrams.vnmskel` | |
| Animation clips | `models/heroes_wip/abrams/clips/` | 238 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/bull_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/bull_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/bull_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/bull_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/bull_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/bull_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/abrams_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/bull_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/bull_beef_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/bull_charge_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/bull_drain_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/bull_jump_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/abrams.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/abrams_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/abrams.vpk` | |
| Effects | `particles/abilities/abrams/` | |
| Effects | `particles/abilities/bull/` | |
| Effects | `particles/weapon_fx/bull/` | |
| Name and tag text | `Steam_RP_hero_atlas`, `hero_atlas:n`, `hero_atlas_search:n`, `Citadel_Abrams_HeroTag_1`, `Citadel_Abrams_HeroTag_2`, `Citadel_Abrams_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Abrams's own 238 clips on whatever model is installed, so a replacement is rigged to Abrams's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `FullBody`, `melee_UpperBody`, `Parry_UpperBody`, `Reload_Crouch`, `Reload_Haspath`, `Reload_HasPathCrouch`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `Reload_Zoomed`, `Throw_UpperBody`, `Turn_LowerBody`, `UpperBodyReplace_AirDash`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_Asleep_Channeling`, `UpperBodyReplace_GroundDash`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `melee_dash`, `syphon_mask`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model atlas assets/model.glb
dlmod select atlas bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons atlas 1=assets/icon_....png
dlmod image panorama/images/heroes/bull_card_psd.vtex_c assets/card.png
dlmod paint atlas
```
