# The Doorman (`doorman`)

Weapon: Service With A Smile. Tags: Disorienting, Map Control, Mind Games.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/the-doorman
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Call Bell | Throw out a call bell that deals X on impact. After a short delay it explodes, dealing additional X and causing… | `doorman_bell_psd.vtex_c` |  |
| Doorway | Place two connected doors in the world. Players and most projectiles entering one door will exit out through the other… | `doorman_doorway_psd.vtex_c` |  |
| Luggage Cart | Send out a Luggage Cart that deals X and X enemy heroes along its path. Can be alt cast with X to target friendly… | `doorman_luggagetrolley_psd.vtex_c` |  |
| Hotel Guest | Send the target's physical body to be a guest at the Baroness Hotel. The guest is to promptly make their way to the… | `doorman_elevator_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_wip/doorman_v2/doorman.vmdl` | 126 units tall |
| Animation skeleton | `models/heroes_wip/doorman_v2/doorman.vnmskel` | |
| Animation clips | `models/heroes_wip/doorman_v2/clips/` | 224 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/doorman_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/doorman_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/doorman_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/doorman_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/doorman_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/doorman_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/doorman_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/doorman_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/doorman/doorman_bell_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/doorman/doorman_doorway2_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/doorman/doorman_doorway_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/doorman/doorman_doorwayclose_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/doorman/doorman_elevator_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/doorman/doorman_luggagetrolley_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/doorman.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/doorman_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/doorman.vpk` | |
| Ability model | `models/abilities/doorman_key_fx_model.vmdl` | |
| Effects | `particles/abilities/doorman/` | |
| Effects | `particles/weapon_fx/doorman/` | |
| Name and tag text | `Steam_RP_hero_doorman`, `hero_doorman:n`, `hero_doorman_search:n`, `hero_doorman_sort:n`, `Citadel_Doorman_HeroTag_1`, `Citadel_Doorman_HeroTag_2`, `Citadel_Doorman_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays The Doorman's own 224 clips on whatever model is installed, so a replacement is rigged to The Doorman's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `FullBody`, `melee_dash`, `melee_UpperBody`, `Parry_UpperBody`, `Reload_Crouch`, `Reload_Haspath`, `Reload_HasPathCrouch`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `Reload_Zoomed`, `Root`, `Throw_UpperBody`, `Turn_LowerBody`, `UpperBodyReplace_AirDash`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_Asleep_Channeling`, `UpperBodyReplace_GroundDash`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Shopping`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `zero`, `Arm_L_Only`, `Call_Bell_UpperBody`, `Doorway_Precast_UpperBody`, `Doorway_Close_UpperBody`, `Reload_Pause_Overlay_Full`, `Reload_Pause_Overlay_Door_Gun`, `Reload_Pause_Overlay_Door`, `Reload_Pause_Overlay_Hand`, `Reload_Additive`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model doorman assets/model.glb
dlmod select doorman bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons doorman 1=assets/icon_....png
dlmod image panorama/images/heroes/doorman_card_psd.vtex_c assets/card.png
dlmod prop models/abilities/doorman_key_fx_model.vmdl assets/prop_doorman_key_fx_model.glb
dlmod paint doorman
```
