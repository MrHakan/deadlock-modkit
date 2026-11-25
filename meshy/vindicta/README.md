# Vindicta (`hornet`)

Snipes the unsuspecting. Vindicta has the mobility to engage at her leisure, picking at her prey and bleeding them out slowly. Injured enemies at any distance should fear being picked off when she is on the map.

Weapon: Wallflower's Revenge. Tags: Sniper, Soaring, One Shot Kill.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/vindicta
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Stake | Tether your enemies to the ground | `vindicta_stake_psd.vtex_c` |  |
| Flight | Take flight and empower your weapon | `vindicta_flight_psd.vtex_c` |  |
| Crow Familiar | Make a target bleed over time | `hornet_crow_psd.vtex_c` |  |
| Assassinate | Snipe injured targets for massive damage | `hornet_assassinate_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_staging/hornet_v3/hornet.vmdl` | 91 units tall |
| Animation skeleton | `models/heroes_staging/hornet_v3/hornet.vnmskel` | |
| Animation clips | `models/heroes_staging/hornet_v3/clips/` | 153 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/hornet_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/hornet_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/hornet_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/hornet_vertical_psd.vtex_c` | 120x200 |
| Minimap icon | `panorama/images/heroes/hornet_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/vindicta_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/hornet_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/hornet/hornet_assassinate_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/hornet/hornet_crow_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/hornet/vindicta_flight_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/hornet/vindicta_stake_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/vindicta.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/vindicta_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/hornet.vpk` | |
| Ability model | `models/abilities/vindicta_nail_model_fx.vmdl` | |
| Effects | `particles/abilities/hornet/` | |
| Effects | `particles/weapon_fx/hornet/` | |
| Name and tag text | `Steam_RP_hero_hornet`, `hero_hornet:n`, `hero_hornet_search:n`, `hero_hornet_sort:n`, `Citadel_Vindicta_HeroTag_1`, `Citadel_Vindicta_HeroTag_2`, `Citadel_Vindicta_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Vindicta's own 153 clips on whatever model is installed, so a replacement is rigged to Vindicta's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`, `reload_stand_idle`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `melee_UpperBody`, `Reload_Crouch`, `Reload_Haspath`, `Reload_HasPathCrouch`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `Throw_UpperBody`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_GroundDash`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Shopping`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model hornet assets/model.glb
dlmod select hornet bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons hornet 1=assets/icon_....png
dlmod image panorama/images/heroes/hornet_card_psd.vtex_c assets/card.png
dlmod prop models/abilities/vindicta_nail_model_fx.vmdl assets/prop_vindicta_nail_model_fx.glb
dlmod paint hornet
```
