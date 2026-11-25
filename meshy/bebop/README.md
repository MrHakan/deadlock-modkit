# Bebop (`bebop`)

Hooks an enemy for a knockout combo. Bebop can string together his abilities in many different ways. Disrupting enemy positions, saving an ally or dealing bursts of damage. When the situation doesn't require subtlety his giant laser still solves many problems.

Weapon: Humble Pie. Tags: Hook, Bomb, Punch.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/bebop
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Exploding Uppercut | Punch someone away from you | `bebop_uppercut_psd.vtex_c` |  |
| Sticky Bomb | Plant a bomb on an enemy | `bebop_sticky_bomb_psd.vtex_c` |  |
| Grapple Arm | Pull in a target to you | `bebop_hook_psd.vtex_c` |  |
| Hyper Beam | Unleash a massive damaging beam | `bebop_hyper_beam_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_staging/bebop/bebop.vmdl` | 111 units tall |
| Animation skeleton | `models/heroes_staging/bebop/bebop.vnmskel` | |
| Animation clips | `models/heroes_staging/bebop/clips/` | 207 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/bebop_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/bebop_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/bebop_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/bebop_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/bebop_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/bebop_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/bebop_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/bebop_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/bebop/bebop_hook_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/bebop/bebop_hyper_beam_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/bebop/bebop_sticky_bomb_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/bebop/bebop_uppercut_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/bebop.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/bebop_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/bebop.vpk` | |
| Effects | `particles/abilities/bebop/` | |
| Effects | `particles/weapon_fx/bebop/` | |
| Name and tag text | `Steam_RP_hero_bebop`, `hero_bebop:n`, `hero_bebop_search:n`, `hero_bebop_sort:n`, `Citadel_Bebop_HeroTag_1`, `Citadel_Bebop_HeroTag_2`, `Citadel_Bebop_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Bebop's own 207 clips on whatever model is installed, so a replacement is rigged to Bebop's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `out_of_combat_idle`, `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `FullBody`, `melee_dash`, `melee_UpperBody`, `Parry_UpperBody`, `Reload_Crouch`, `Reload_HasPathCrouch`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `Reload_Zoomed`, `Root`, `Throw_UpperBody`, `UpperBodyReplace_AirDash`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_Asleep_Channeling`, `UpperBodyReplace_GroundDash`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Shopping`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `Reload_Haspath`, `weapon_only`, `Hook_UpperBody`, `Hyperbeam_Upperbody`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model bebop assets/model.glb
dlmod select bebop bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons bebop 1=assets/icon_....png
dlmod image panorama/images/heroes/bebop_card_psd.vtex_c assets/card.png
dlmod paint bebop
```
