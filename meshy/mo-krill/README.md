# Mo & Krill (`krill`)

Burrows underneath his enemies. By burrowing through the ground, Mo and Krill can get the drop on almost anyone; and once they get in close the results can be devastating.

Weapon: Yellow Canary. Tags: Tag-Team, Initiator, Burrower.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/mo-krill
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Scorn | Damage enemies nearby, healing from each hit | `grappler_regen_psd.vtex_c` |  |
| Burrow | Travel underground, emerging with spin damage | `grappler_spin_psd.vtex_c` |  |
| Sand Blast | Disarm enemies in front of you | `grappler_throw_sand_psd.vtex_c` |  |
| Combo | Hold the enemy hero in place and pummel them | `grappler_combo_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_staging/digger/digger.vmdl` | 96 units tall |
| Animation skeleton | `models/heroes_staging/digger/mo_krill.vnmskel` | |
| Animation clips | `models/heroes_staging/digger/clips/` | 148 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/digger_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/digger_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/digger_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/digger_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/digger_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/digger_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/krill_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/digger_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/duo/duo_attack_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/duo/duo_cancel_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/grappler/grappler_regen_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/grappler/grappler_spin_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/grappler/grappler_throw_sand_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/grappler/grappler_combo_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/mo_krill.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/mo_krill_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/digger.vpk` | |
| Effects | `particles/abilities/digger/` | |
| Effects | `particles/abilities/duo/` | |
| Effects | `particles/weapon_fx/digger/` | |
| Name and tag text | `Steam_RP_hero_duo`, `Steam_RP_hero_krill`, `hero_duo:n`, `hero_duo_search:n`, `hero_krill:n`, `hero_krill_search:n`, `Citadel_Digger_HeroTag_1`, `Citadel_Digger_HeroTag_2`, `Citadel_Digger_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Mo & Krill's own 148 clips on whatever model is installed, so a replacement is rigged to Mo & Krill's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`, `reload_stand_idle`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `Reload_Haspath`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `Throw_UpperBody`, `melee_UpperBody`, `Reload_Crouch`, `Reload_HasPathCrouch`, `Parry_UpperBody`, `melee_UpperBody_01`, `UpperBodyReplace_Turn`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model krill assets/model.glb
dlmod select krill bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons krill 1=assets/icon_....png
dlmod image panorama/images/heroes/digger_card_psd.vtex_c assets/card.png
dlmod paint krill
```
