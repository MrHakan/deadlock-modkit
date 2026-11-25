# Venator (`priest`)

Weapon: 51:20. Tags: Devout, Arms Expert, Tactical.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/venator
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Consecrating Grenade | Fire a grenade that bounces before exploding, dealing X and setting enemies on fire. Burning targets deal X to enemies… | `priest_grenade_psd.vtex_c` |  |
| Gutshot | Fire a blast with your shotgun, dealing X and pushing enemies back. Enemies near a wall receive X and take X. | `priest_shotgun_psd.vtex_c` |  |
| Hex-Lined Snap Trap | Kick a trap that arms after a brief delay. The trap springs on the first enemy it touches, dealing X, applying X, and… | `priest_trap_psd.vtex_c` |  |
| Ira Domini | Load your crossbow with 3 stakes, dealing massively increased X. The final stake is blessed, dealing bonus X and… | `priest_crossbow_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_wip/priest/priest.vmdl` | 114 units tall |
| Animation skeleton | `models/heroes_wip/priest/priest.vnmskel` | |
| Animation clips | `models/heroes_wip/priest/clips/` | 354 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/priest_card_psd.vtex_c` | 240x320 |
| Card at low health | `panorama/images/heroes/priest_card_critical_psd.vtex_c` | 240x320 |
| Card after a kill | `panorama/images/heroes/priest_card_gloat_psd.vtex_c` | 240x320 |
| Tall portrait | `panorama/images/heroes/priest_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/priest_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/priest_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/priest_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/priest_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/priest/priest_crossbow_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/priest/priest_grenade_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/priest/priest_shotgun_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/priest/priest_trap_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/priest.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/priest_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/priest.vpk` | |
| Effects | `particles/abilities/priest/` | |
| Effects | `particles/weapon_fx/priest/` | |
| Name and tag text | `Steam_RP_hero_priest`, `hero_priest:n`, `hero_priest_search:n`, `hero_priest_sort:n`, `Citadel_Priest_HeroTag_1`, `Citadel_Priest_HeroTag_2`, `Citadel_Priest_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Venator's own 354 clips on whatever model is installed, so a replacement is rigged to Venator's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `out_of_combat_idle`, `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`ability_knockback_upperbody`, `UpperBodyReplace_InAir`, `Cast_UpperBody`, `FullBody`, `melee_dash`, `melee_UpperBody`, `Parry_UpperBody`, `Reload_Crouch`, `Reload_Idle`, `Root`, `Throw_UpperBody`, `Turn_LowerBody`, `UpperBodyReplace_AirDash`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_GroundDash`, `UpperBodyReplace_Asleep_Channeling`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `zero`, `Reload_Haspath`, `Reload_InAir`, `Reload_Slide`, `Reload_HasPathCrouch`, `ability_knockback_crouch`, `ability_incense_throw`, `ability_crossbow`, `ability_trap_inair`, `ability_trap_haspath`, `ability_trap_crossbow`, `ability_flashbang`, `ability_knockback_sliding`, `ability_crossbow_shoot`, `Right_Arm`, `ability_knockback_dash_ground`, `Reload_Crouch_Crossbow`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model priest assets/model.glb
dlmod select priest bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons priest 1=assets/icon_....png
dlmod image panorama/images/heroes/priest_card_psd.vtex_c assets/card.png
dlmod paint priest
```
