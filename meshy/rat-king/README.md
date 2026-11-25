# Rat King (`ratking`)

Weapon: Sovereign's Decree. Tags: Scrappy, Regal, Tenacious.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/rat-king
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Scrap Grenade | Throw a grenade of scrap that detonates multiple times, dealing X and applying X. The grenade can be meleed to knock it… | `ratking_grenade_psd.vtex_c` | `models/abilities/ratking_scrap_geo_casing_fx_model.vmdl`, `models/abilities/ratking_scrap_geo_fx_model.vmdl` |
| Rat Swarm | Call forth a swarm of rats that leap onto enemies, dealing X and reducing their X with every bite. The more rats are… | `ratking_swarm_psd.vtex_c` |  |
| Royal Pestments | Adorn scrappy armor, gaining X based on your X. As long as the armor holds, bullets have a chance to be deflected back… | `ratking_shield_psd.vtex_c` | `models/abilities/ratking_scrap_geo_casing_fx_model.vmdl`, `models/abilities/ratking_scrap_geo_fx_model.vmdl` |
| Rule, Ratannia! | Charge up and summon the banner of Ratannia, granting all nearby allies increased X, immunity from slows, and reduced… | `ratking_flag_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_wip/ratking/ratking.vmdl` | 249 units tall |
| Animation skeleton | `models/heroes_wip/ratking/ratking.vnmskel` | |
| Animation clips | `models/heroes_wip/ratking/clips/` | 198 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/ratking_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/ratking_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/ratking_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/ratking_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/ratking_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/ratking_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/ratking_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/ratking_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/ratking/ratking_flag_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/ratking/ratking_grenade_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/ratking/ratking_nibble_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/ratking/ratking_shield_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/ratking/ratking_swarm_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/ratking.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/ratking_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/ratking.vpk` | |
| Ability model | `models/abilities/ratking_scrap_geo_casing_fx_model.vmdl` | |
| Ability model | `models/abilities/ratking_scrap_geo_fx_model.vmdl` | |
| Effects | `particles/abilities/ratking/` | |
| Effects | `particles/weapon_fx/ratking/` | |
| Name and tag text | `Steam_RP_hero_ratking`, `hero_ratking:n`, `hero_ratking_search:n`, `hero_ratking_sort:n`, `Citadel_Ratking_HeroTag_1`, `Citadel_Ratking_HeroTag_2`, `Citadel_Ratking_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Rat King's own 198 clips on whatever model is installed, so a replacement is rigged to Rat King's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `out_of_combat_idle`, `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`, `reload_stand_idle`.
- Melee, reloads and casts are laid over the movement through bone masks (`Ability_idle`, `Cast_UpperBody`, `FullBody`, `melee_UpperBody`, `Parry_UpperBody`, `Reload_Crouch`, `Reload_GunOnly`, `Reload_Haspath`, `Reload_HasPathCrouch`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `Reload_Zoomed`, `Root`, `Throw_UpperBody`, `Turn_LowerBody`, `UpperBody_Fidget`, `UpperBodyReplace_AirDash`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_Asleep_Channeling`, `UpperBodyReplace_GroundDash`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Shopping`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `zero`, `Cast_UpperBody_Rat`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model ratking assets/model.glb
dlmod select ratking bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons ratking 1=assets/icon_....png
dlmod image panorama/images/heroes/ratking_card_psd.vtex_c assets/card.png
dlmod prop models/abilities/ratking_scrap_geo_casing_fx_model.vmdl assets/prop_ratking_scrap_geo_casing_fx_model.glb
dlmod paint ratking
```
