# Infernus (`inferno`)

Lights up enemies and watches them burn. Infernus has many ways to deal damage over time, burning foes before swooping in for the kill. Due to Infernus' blazing speed, his enemies won't be able to escape the flames.

Weapon: Incendiary Remarks. Tags: Arsonist, Explosive, Burn Rubber.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/infernus
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Napalm | Spew napalm that amplifies your damage | `inferno_molotov_psd.vtex_c` |  |
| Flame Dash | Blaze forward, leaving a flame trail in your wake | `inferno_dash_psd.vtex_c` |  |
| Afterburn | Shoot enemies to set them on fire | `inferno_deflect_psd.vtex_c` |  |
| Concussive Combustion | Stun all nearby enemies after a short delay | `inferno_bomb_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_wip/inferno/inferno.vmdl` | 115 units tall |
| Animation skeleton | `models/heroes_wip/inferno/inferno.vnmskel` | |
| Animation clips | `models/heroes_wip/inferno/clips/` | 198 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/inferno_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/inferno_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/inferno_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/inferno_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/inferno_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/inferno_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/infernus_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/inferno_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/inferno_bomb_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/inferno_dash_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/inferno_deflect_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/inferno_molotov_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/inferno_passive_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/infernus.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/infernus_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/inferno.vpk` | |
| Effects | `particles/abilities/inferno/` | |
| Effects | `particles/weapon_fx/inferno/` | |
| Name and tag text | `Steam_RP_hero_inferno`, `hero_inferno:n`, `hero_inferno_search:n`, `hero_inferno_sort:n`, `Citadel_Inferno_HeroTag_1`, `Citadel_Inferno_HeroTag_2`, `Citadel_Inferno_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Infernus's own 198 clips on whatever model is installed, so a replacement is rigged to Infernus's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `FullBody`, `melee_dash`, `melee_UpperBody`, `Parry_UpperBody`, `Reload_Haspath`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `root`, `Throw_UpperBody`, `UpperBodyReplace_AirDash`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_Asleep_Channeling`, `UpperBodyReplace_GroundDash`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `zero`, `Reload_HasPathCrouch`, `Reload_Crouch`, `ability_flame_dash`, `ability_flame_dash_start`, `ability_flame_dash_exit`, `Item_right_arm`, `Ability_Napalm`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model inferno assets/model.glb
dlmod select inferno bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons inferno 1=assets/icon_....png
dlmod image panorama/images/heroes/inferno_card_psd.vtex_c assets/card.png
dlmod paint inferno
```
