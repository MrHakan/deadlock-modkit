# Drifter (`drifter`)

Weapon: Crimson Grasp. Tags: Stalker, Bloodthirsty, Cruel.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/drifter
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Rend | Swipe at enemies in a cone ahead of you, dealing X. If the enemy is in close range, deal bonus X. | `drifter_claw_psd.vtex_c` |  |
| Stalker's Mark | Send out a mark that bleeds the first target it hits, dealing X. While the enemy is bleeding you can re-activate… | `drifter_stalkersmark_psd.vtex_c` |  |
| Bloodscent |  | `drifter_thehunger_psd.vtex_c` |  |
| Eternal Night | Surround nearby enemy heroes in darkness, severely limiting their vision of other units. Affected enemies are briefly… | `darkness_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_wip/drifter/drifter.vmdl` | 109 units tall |
| Animation skeleton | `models/heroes_wip/drifter/drifter.vnmskel` | |
| Animation clips | `models/heroes_wip/drifter/clips/` | 254 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/drifter_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/drifter_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/drifter_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/drifter_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/drifter_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/drifter_mm_psd.vtex_c` | 50x52 |
| Background | `panorama/images/heroes/backgrounds/drifter_bg_psd.vtex_c` | 4096x2312 |
| Ability icon | `panorama/images/hud/abilities/drifter/darkness_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/drifter/drifter_claw_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/drifter/drifter_stalkersmark_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/drifter/drifter_teleport_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/drifter/drifter_terror_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/drifter/drifter_thehunger_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/drifter.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/drifter_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/drifter.vpk` | |
| Effects | `particles/abilities/drifter/` | |
| Effects | `particles/weapon_fx/drifter/` | |
| Name and tag text | `Steam_RP_hero_drifter`, `hero_drifter:n`, `hero_drifter_search:n`, `hero_drifter_sort:n`, `Citadel_Drifter_HeroTag_1`, `Citadel_Drifter_HeroTag_2`, `Citadel_Drifter_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Drifter's own 254 clips on whatever model is installed, so a replacement is rigged to Drifter's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `FullBody`, `melee_dash`, `melee_UpperBody`, `Parry_UpperBody`, `Reload_Crouch`, `Reload_Haspath`, `Reload_HasPathCrouch`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `root`, `Throw_UpperBody`, `Turn_LowerBody`, `UpperBodyReplace_AirDash`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_Asleep_Channeling`, `UpperBodyReplace_GroundDash`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `zero`, `Ability_BloodBlast_InAir`, `ability_bloodblast_haspath`, `Ability_Stalker_Claw_HasPath`, `Ability_Stalker_Claw_InAir`, `Claw_R_HasPath`, `Claw_L_HasPath`, `Claw_L_InAir`, `Claw_R_InAir`, `Claws_idle`, `OutOfCombat_Transition`, `Ability_Teleport`, `Claws_HasPath`, `Ability_Darkness_HasPath`, `Ability_Darkness_InAir`, `Claws_idle_crouch`, `Weapon_to_OOC`, `Reload_Idle_ALT`, `Claws_idle_ALT`, `Claw_R_DashAir`, `Run_fidget`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model drifter assets/model.glb
dlmod select drifter bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons drifter 1=assets/icon_....png
dlmod image panorama/images/heroes/drifter_card_psd.vtex_c assets/card.png
dlmod paint drifter
```
