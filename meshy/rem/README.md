# Rem (`familiar`)

Weapon: Long Night. Tags: Helpful, Tiny, zZzzZ.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/rem
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Pillow Toss | Throw your trusty pillow inflicting X and heavy X. Landing hits reduces the cooldown of your other abilities. | `familiar_pillow_psd.vtex_c` |  |
| Tag Along | Jump to an ally and nap alongside them. Both you and the ally receive a burst X based on missing health followed by… | `familiar_tag_along_psd.vtex_c` | `models/abilities/familiar/familiar_helper_attached.vmdl` |
| Lil Helpers | Signal a Helper to lend a hand. They can collect boxes, do Sinner's Sacrifices, or be sent to support allies and… | `familiar_helpers_psd.vtex_c` | `models/abilities/familiar/familiar_helper_attached.vmdl`, `models/abilities/familiar/familiar_helper_npc.vmdl`, `models/abilities/familiar/familiar_helpinghands_aura_statue.vmdl` |
| Naptime | Cast your gaze forward, piercing walls and floors. Enemies in the gaze are slowed, have reduced dash distance, and… | `familiar_sleep_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_wip/familiar/familiar_wip.vmdl` | 66 units tall |
| Animation skeleton | `models/heroes_wip/familiar/familiar.vnmskel` | |
| Animation clips | `models/heroes_wip/familiar/clips/` | 120 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/familiar_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/familiar_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/familiar_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/familiar_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/familiar_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/familiar_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/familiar_bg_psd.vtex_c` | 4096x2312 |
| Ability icon | `panorama/images/hud/abilities/familiar/familiar_helpers_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/familiar/familiar_pillow_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/familiar/familiar_sleep_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/familiar/familiar_tag_along2_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/familiar/familiar_tag_along_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/familiar.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/familiar_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/familiar.vpk` | |
| Ability model | `models/abilities/familiar/familiar_helper_attached.vmdl` | |
| Ability model | `models/abilities/familiar/familiar_helper_npc.vmdl` | |
| Ability model | `models/abilities/familiar/familiar_helpinghands_aura_statue.vmdl` | |
| Effects | `particles/abilities/familiar/` | |
| Effects | `particles/weapon_fx/familiar/` | |
| Name and tag text | `Steam_RP_hero_familiar`, `hero_familiar:n`, `hero_familiar_search:n`, `Citadel_Familiar_HeroTag_1`, `Citadel_Familiar_HeroTag_2`, `Citadel_Familiar_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Rem's own 120 clips on whatever model is installed, so a replacement is rigged to Rem's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `FullBody`, `melee_dash`, `melee_UpperBody`, `Reload_Crouch`, `Reload_GunOnly`, `Reload_Haspath`, `Reload_HasPathCrouch`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `Reload_Zoomed`, `root`, `Throw_UpperBody`, `Turn_LowerBody`, `UpperBodyReplace_AirDash`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_Asleep_Channeling`, `UpperBodyReplace_GroundDash`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Shopping`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `zero`, `Parry_UpperBody`, `Cast_Crouch_UpperBody`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model familiar assets/model.glb
dlmod select familiar bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons familiar 1=assets/icon_....png
dlmod image panorama/images/heroes/familiar_card_psd.vtex_c assets/card.png
dlmod prop models/abilities/familiar/familiar_helper_attached.vmdl assets/prop_familiar_helper_attached.glb
dlmod paint familiar
```
