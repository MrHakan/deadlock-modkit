# Violet (`artist`)

Weapon: Creative Solutions. Tags: Ambitious, Nimble, Creative.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/violet
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Living Ink |  | `artist_livingink_psd.vtex_c` |  |
| Flow State | Drag a brush that continues to travel forward, leaving a trail that grants bonus slide distance for allies and a fading… | `artist_brush_psd.vtex_c` | `models/abilities/artist_ability1_brush_heroselect_model.vmdl`, `models/abilities/artist_ability1_brush_model.vmdl`, `models/abilities/artist_charm_brush_model_fx.vmdl`, `models/abilities/artist_ground_slide_streak_fx.vmdl`, `models/abilities/artist_slide_streak_fx.vmdl`, `models/abilities/artist_turtle_brush_model_fx.vmdl` |
| Still Life | Choose a hero to be your subject, dealing X as long as you are looking at them. If the sketch is completed they take… | `artist_portrait_psd.vtex_c` |  |
| Abstract Landscape | Create a circular display of ink that blocks vision and grants allies X. Enemies inside are revealed, and leaving… | `artist_landscape_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_wip/artist/artist.vmdl` | 108 units tall |
| Animation skeleton | `models/heroes_wip/artist/artist.vnmskel` | |
| Animation clips | `models/heroes_wip/artist/clips/` | 237 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/artist_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/artist_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/artist_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/artist_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/artist_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/artist_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/artist_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/artist_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/artist/artist_brush_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/artist/artist_brushflick_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/artist/artist_landscape_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/artist/artist_landscapeend_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/artist/artist_livingink_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/artist/artist_portrait_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/violet.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/violet_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/artist.vpk` | |
| Ability model | `models/abilities/artist_ability1_brush_heroselect_model.vmdl` | |
| Ability model | `models/abilities/artist_ability1_brush_model.vmdl` | |
| Ability model | `models/abilities/artist_charm_brush_model_fx.vmdl` | |
| Ability model | `models/abilities/artist_globules.vmdl` | |
| Ability model | `models/abilities/artist_ground_slide_streak_fx.vmdl` | |
| Ability model | `models/abilities/artist_papers_model_fx.vmdl` | |
| Ability model | `models/abilities/artist_papers_oriented_model_fx.vmdl` | |
| Ability model | `models/abilities/artist_slide_streak_fx.vmdl` | |
| Ability model | `models/abilities/artist_tiger_model_fx.vmdl` | |
| Ability model | `models/abilities/artist_turtle_brush_model_fx.vmdl` | |
| Effects | `particles/abilities/artist/` | |
| Effects | `particles/weapon_fx/artist/` | |
| Name and tag text | `Steam_RP_hero_artist`, `hero_artist:n`, `hero_artist_search:n`, `hero_artist_sort:n`, `Citadel_Artist_HeroTag_1`, `Citadel_Artist_HeroTag_2`, `Citadel_Artist_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Violet's own 237 clips on whatever model is installed, so a replacement is rigged to Violet's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `melee_UpperBody`, `Parry_UpperBody`, `Reload_Additive`, `Reload_Crouch`, `Reload_Haspath`, `Reload_HasPathCrouch`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `shoot_haspath`, `shoot_Idle`, `shoot_slide`, `shoot_inair`, `shoot_crouch`, `Throw_UpperBody`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `Ability_masterwork_haspath`, `Reload_HasPath_ALT`, `weapon_ooc_run`, `weapon_ooc_run_e`, `ability_pullbrush`, `Reload_Slide_ALT`, `Ability_StillLife_haspath`, `Ability_Living_Ink_Start_haspath`, `Ability_Living_Ink_Start_inair`, `Ability_Living_Ink_Start_slide`, `weapon_ooc_crouch_run`, `Ability_Living_Ink_slide`, `Ability_Ink_Wash_haspath`, `ability_pullbrush_haspath`, `ability_pullbrush_slide`, `Ability_Ink_Wash_in_air`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model artist assets/model.glb
dlmod select artist bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons artist 1=assets/icon_....png
dlmod image panorama/images/heroes/artist_card_psd.vtex_c assets/card.png
dlmod prop models/abilities/artist_ability1_brush_heroselect_model.vmdl assets/prop_artist_ability1_brush_heroselect_model.glb
dlmod paint artist
```
