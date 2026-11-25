# Yamato (`yamato`)

Delivers charged attacks with finesse. Yamato uses precise strikes to decimate her rivals. However, such damaging attacks often will leave her exposed, so she needs to make every shot count.

Weapon: Sister's Lament. Tags: Relentless, Acrobatics, Pursuer.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/yamato
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Power Slash | Charge up an attack, dealing high damage on a full charge | `yamato_power_slash_psd.vtex_c` | `models/abilities/yamato_weapon_long_sword_tip_fx_model.vmdl` |
| Flying Slash | Fly towards an enemy target | `yamato_flying_strike_psd.vtex_c` |  |
| Crimson Slash | Damage enemies nearby, healing on hit | `yamato_crimson_slash_psd.vtex_c` |  |
| Shadow Transformation | Become invincible and more powerful | `yamato_blinding_steel_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_staging/yamato_v2/yamato.vmdl` | 116 units tall |
| Animation skeleton | `models/heroes_wip/yamato/yamato.vnmskel` | |
| Animation clips | `models/heroes_wip/yamato/clips/` | 182 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/yamato_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/yamato_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/yamato_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/yamato_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/yamato_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/yamato_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/yamato_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/yamato_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/yamato/yamato_blinding_steel_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/yamato/yamato_crimson_slash_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/yamato/yamato_flying_strike_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/yamato/yamato_power_slash_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/yamato.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/yamato_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/yamato.vpk` | |
| Ability model | `models/abilities/yamato_weapon_long_sword_tip_fx_model.vmdl` | |
| Effects | `particles/abilities/yamato/` | |
| Effects | `particles/weapon_fx/yamato/` | |
| Name and tag text | `Steam_RP_hero_yamato`, `hero_yamato:n`, `hero_yamato_search:n`, `Citadel_Yamato_HeroTag_1`, `Citadel_Yamato_HeroTag_2`, `Citadel_Yamato_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Yamato's own 182 clips on whatever model is installed, so a replacement is rigged to Yamato's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `out_of_combat_idle`, `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `FullBody`, `melee_dash`, `melee_UpperBody`, `Parry_UpperBody`, `Reload_Crouch`, `Reload_Haspath`, `Reload_HasPathCrouch`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `Reload_Zoomed`, `Root`, `Throw_UpperBody`, `Turn_LowerBody`, `UpperBodyReplace_AirDash`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_Asleep_Channeling`, `UpperBodyReplace_GroundDash`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Shopping`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `zero`, `shoot_crouch`, `shoot_haspath`, `shoot_Idle`, `shoot_inair`, `shoot_slide`, `shoot_turn`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model yamato assets/model.glb
dlmod select yamato bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons yamato 1=assets/icon_....png
dlmod image panorama/images/heroes/yamato_card_psd.vtex_c assets/card.png
dlmod prop models/abilities/yamato_weapon_long_sword_tip_fx_model.vmdl assets/prop_yamato_weapon_long_sword_tip_fx_model.glb
dlmod paint yamato
```
