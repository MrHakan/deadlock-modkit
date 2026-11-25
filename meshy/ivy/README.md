# Ivy (`tengu`)

Sets up a power duo with a lucky ally. Ivy has disruptive combat tricks that slow and stun enemies. She's most effective when linking up with an ally, empowering her and her ally's guns as well as sharing heals.

Weapon: Protector's Hand. Tags: Team-up, Disruptor, Rock Solid.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/ivy
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Entangling Thorns | Damage and slow enemies in the area | `tengu_storm_flask_psd.vtex_c` | `models/abilities/ivy_thornbomb_model.vmdl` |
| Kudzu Connection | Empower yourself and a nearby ally | `tengu_tether_psd.vtex_c` |  |
| Stone Form | Turn into impervious stone and stun enemies nearby | `tengu_stone_form_psd.vtex_c` |  |
| Air Drop | Death from Above! Fly and drop an ally on enemies | `tengu_lightning_crash_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_wip/ivy/ivy.vmdl` | 112 units tall |
| Animation skeleton | `models/heroes_staging/tengu/tengu_v2/dmx/mesh/ivy.vnmskel` | |
| Animation clips | `models/heroes_staging/tengu/tengu_v2/clips/` | 160 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/tengu_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/tengu_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/tengu_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/tengu_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/tengu_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/tengu_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/ivy_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/ivy_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/tengu/tengu_lightning_crash_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/tengu/tengu_stone_form_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/tengu/tengu_storm_flask_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/tengu/tengu_tether_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/ivy.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/ivy_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/tengu.vpk` | |
| Ability model | `models/abilities/ivy_bramble_model_fx.vmdl` | |
| Ability model | `models/abilities/ivy_thornbomb_model.vmdl` | |
| Effects | `particles/abilities/tengu/` | |
| Effects | `particles/weapon_fx/tengu/` | |
| Name and tag text | `Steam_RP_hero_tengu`, `hero_tengu:n`, `hero_tengu_search:n`, `Citadel_Tengu_HeroTag_1`, `Citadel_Tengu_HeroTag_2`, `Citadel_Tengu_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Ivy's own 160 clips on whatever model is installed, so a replacement is rigged to Ivy's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_idle`, `out_of_combat_idle`, `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`, `reload_stand_idle`, `weapon_shoot`.
- Melee, reloads and casts are laid over the movement through bone masks (`Throw_UpperBody`, `Cast_UpperBody`, `Reload_Crouch`, `Reload_Haspath`, `Reload_HasPathCrouch`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `melee_UpperBody`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_GroundDash`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Shopping`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `Parry_UpperBody`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model tengu assets/model.glb
dlmod select tengu bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons tengu 1=assets/icon_....png
dlmod image panorama/images/heroes/tengu_card_psd.vtex_c assets/card.png
dlmod prop models/abilities/ivy_bramble_model_fx.vmdl assets/prop_ivy_bramble_model_fx.glb
dlmod paint tengu
```
