# Apollo (`fencer`)

Weapon: Pride of Ixia. Tags: Finesse, Mobility, A Cut Above.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/apollo
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Disengaging Sigil | Draw a sigil sphere in front of you and then leap backwards as it explodes, damaging and slowing enemies caught in it. | `fencer_sigil_psd.vtex_c` |  |
| Riposte | Prepare to deflect the next incoming attack. On a successful deflection, briefly become invulnerable and target an… | `fencer_riposte_psd.vtex_c` |  |
| Flawless Advance | Perform a series of lunges in any direction, delivering piercing stabs ahead of you. Hold your ability key to time your… | `fencer_lungingstab_psd.vtex_c` |  |
| Itani Lo Sahn | Charge up and perform a long range slash. Struck enemies cannot take action or heal and are stuck in slow motion. When… | `fencer_ult_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_wip/fencer/fencer.vmdl` | 110 units tall |
| Animation skeleton | `models/heroes_wip/fencer/fencer.vnmskel` | |
| Animation clips | `models/heroes_wip/fencer/clips/` | 251 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/fencer_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/fencer_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/fencer_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/fencer_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/fencer_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/fencer_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/fencer_bg_psd.vtex_c` | 4096x2312 |
| Ability icon | `panorama/images/hud/abilities/fencer/fencer_lungingstab_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/fencer/fencer_riposte_attack_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/fencer/fencer_riposte_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/fencer/fencer_sigil_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/fencer/fencer_ult_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/fencer.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/fencer_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/apollo.vpk` | |
| Ability model | `models/abilities/fencer_ground_dust_fx_model.vmdl` | |
| Ability model | `models/abilities/fencer_ground_tunnel_fx_model.vmdl` | |
| Ability model | `models/abilities/fencer_mask_model_fx.vmdl` | |
| Effects | `particles/abilities/fencer/` | |
| Effects | `particles/weapon_fx/fencer/` | |
| Name and tag text | `Steam_RP_hero_fencer`, `hero_fencer:n`, `hero_fencer_search:n`, `Citadel_Fencer_HeroTag_1`, `Citadel_Fencer_HeroTag_2`, `Citadel_Fencer_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Apollo's own 251 clips on whatever model is installed, so a replacement is rigged to Apollo's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`, `reload_stand_idle`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `FullBody`, `melee_UpperBody`, `Parry_UpperBody`, `Reload_Crouch`, `Reload_Haspath`, `Reload_HasPathCrouch`, `Reload_Idle`, `Reload_Slide`, `root`, `Throw_UpperBody`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Turn`, `zero`, `Ability_UpperBody`, `Reload_InAir`, `Ability_Ult_Turn`, `shoot_mask`, `shoot_additive`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model fencer assets/model.glb
dlmod select fencer bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons fencer 1=assets/icon_....png
dlmod image panorama/images/heroes/fencer_card_psd.vtex_c assets/card.png
dlmod prop models/abilities/fencer_ground_dust_fx_model.vmdl assets/prop_fencer_ground_dust_fx_model.glb
dlmod paint fencer
```
