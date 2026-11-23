# Graves (`necro`)

Weapon: The Teacher. Tags: Morbid, Area Denial, Necromancer.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/graves
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Jar of Dead | Passive: Collect death when anything dies nearby and store it in your Jar of Dead. Active: Throw a jar to summon… | `necro_skull_psd.vtex_c` |  |
| Grasping Hands | Unearth a line of grasping hands, summoning a Ghoul and leaving behind a rift. Enemies who pass through take X and… | `necro_hands_psd.vtex_c` |  |
| Essence Theft | Your weapon steals X and X over time, up to a maximum amount per target. | `necro_siphon_psd.vtex_c` |  |
| Borrowed Decree |  | `necro_gravestone_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_wip/necro/necro.vmdl` | 98 units tall |
| Animation skeleton | `models/heroes_wip/necro/necro.vnmskel` | |
| Animation clips | `models/heroes_wip/necro/clips/` | 269 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/necro_card_psd.vtex_c` | 240x320 |
| Card at low health | `panorama/images/heroes/necro_card_critical_psd.vtex_c` | 240x320 |
| Card after a kill | `panorama/images/heroes/necro_card_gloat_psd.vtex_c` | 240x320 |
| Tall portrait | `panorama/images/heroes/necro_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/necro_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/necro_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/necro_bg_psd.vtex_c` | 4096x2312 |
| Ability icon | `panorama/images/hud/abilities/necro/necro_coffin_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/necro/necro_gravestone_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/necro/necro_hands_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/necro/necro_siphon_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/necro/necro_skull_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/necro.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/necro_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/necro.vpk` | |
| Ability model | `models/abilities/necro_jar_model_fx.vmdl` | |
| Effects | `particles/abilities/necro/` | |
| Effects | `particles/weapon_fx/necro/` | |
| Name and tag text | `Steam_RP_hero_necro`, `hero_necro:n`, `hero_necro_search:n`, `hero_necro_sort:n`, `Citadel_Necro_HeroTag_1`, `Citadel_Necro_HeroTag_2`, `Citadel_Necro_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Graves's own 269 clips on whatever model is installed, so a replacement is rigged to Graves's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `melee_UpperBody`, `Reload_Crouch`, `Reload_Haspath`, `Reload_HasPathCrouch`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `shoot_crouch`, `shoot_haspath`, `shoot_Idle`, `shoot_inair`, `shoot_slide`, `shoot_turn`, `Throw_UpperBody`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `shoot_start_haspath`, `Reload_Additive`, `ability_gravestone`, `ability_grasping_hands`, `ability_hand`, `ability_grasping_hands_idle`, `ability_jar_of_dead`, `ability_jar_of_dead_haspath`, `shoot_to_tether_haspath`, `shoot_to_search`, `Parry_UpperBody`, `ability_gravedigging`, `ability_gravedigging_start`, `hand_fidget`, `ability_grasping_hands_crouch`, `UpperBody_Weapon_Shoot_Run`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model necro assets/model.glb
dlmod select necro bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons necro 1=assets/icon_....png
dlmod image panorama/images/heroes/necro_card_psd.vtex_c assets/card.png
dlmod prop models/abilities/necro_jar_model_fx.vmdl assets/prop_necro_jar_model_fx.glb
dlmod paint necro
```
