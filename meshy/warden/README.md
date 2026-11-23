# Warden (`warden`)

Weakens his enemies and chases them down. Warden leads from the front, jumping into the fray and hindering his opponents' ability to fight effectively. But what Warden considers brave, others consider reckless, and if he's not careful he can easily find himself in over his head.

Weapon: Freedom's Cost. Tags: Initiator, Fearless, One Man Army.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/warden
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Alchemical Flask | Damage and debuff enemies nearby | `warden_crowd_control_psd.vtex_c` |  |
| Willpower | Gain a Barrier and move speed | `warden_high_alert_psd.vtex_c` |  |
| Binding Word | Curse your target, rooting them if they don't escape | `warden_lock_down_psd.vtex_c` |  |
| Last Stand | Emit pulses that steal health from enemies | `warden_riot_protocol_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_staging/warden/warden.vmdl` | 109 units tall |
| Animation skeleton | `models/heroes_staging/warden/warden.vnmskel` | |
| Animation clips | `models/heroes_staging/warden/clip/` | 166 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/warden_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/warden_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/warden_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/warden_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/warden_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/warden_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/warden_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/warden_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/warden/warden_crowd_control_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/warden/warden_high_alert_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/warden/warden_lock_down_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/warden/warden_riot_protocol_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/warden.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/warden_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/warden.vpk` | |
| Ability model | `models/abilities/warden_ghost_model.vmdl` | |
| Ability model | `models/abilities/warden_soul_swirl.vmdl` | |
| Effects | `particles/abilities/warden/` | |
| Effects | `particles/weapon_fx/warden/` | |
| Name and tag text | `Steam_RP_hero_warden`, `hero_warden:n`, `hero_warden_search:n`, `Citadel_Warden_HeroTag_1`, `Citadel_Warden_HeroTag_2`, `Citadel_Warden_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Warden's own 166 clips on whatever model is installed, so a replacement is rigged to Warden's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `out_of_combat_idle`, `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `melee_UpperBody`, `Reload_Crouch`, `Reload_Haspath`, `Reload_HasPathCrouch`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `Reload_Zoomed`, `Throw_UpperBody`, `UpperBodyReplace_AirDash`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_GroundDash`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Shopping`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `Parry_UpperBody`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model warden assets/model.glb
dlmod select warden bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons warden 1=assets/icon_....png
dlmod image panorama/images/heroes/warden_card_psd.vtex_c assets/card.png
dlmod prop models/abilities/warden_ghost_model.vmdl assets/prop_warden_ghost_model.glb
dlmod paint warden
```
