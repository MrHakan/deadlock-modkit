# Paige (`bookworm`)

Weapon: Chapter's End. Tags: Helpful, Protector, Booksmart.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/paige
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Bookwyrm | Conjure a dragon that appears at the target location before flying forward, dealing X and leaving a burning path in its… | `bookworm_dragon_psd.vtex_c` | `models/abilities/bookworm_dragon_model.vmdl` |
| Plot Armor | Grant an ally a X. While the barrier holds, they gain X. | `bookworm_defendandfight_psd.vtex_c` |  |
| Captivating Read | Target an area with latent magic, applying X. The magic detonates after a delay, dealing X and applying X. | `bookworm_blindinglight_psd.vtex_c` |  |
| Rallying Charge | Release a wave of spectral knights and steeds that charge across the entire city, X allies and dealing X to enemies… | `bookworm_charge_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_wip/bookworm/bookworm.vmdl` | 90 units tall |
| Animation skeleton | `models/heroes_wip/bookworm/bookworm.vnmskel` | |
| Animation clips | `models/heroes_wip/bookworm/clips/` | 172 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/bookworm_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/bookworm_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/bookworm_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/bookworm_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/bookworm_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/bookworm_mm_psd.vtex_c` | 50x50 |
| Weapon picture | `panorama/images/heroes/guns/bookworm_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/bookworm/bookworm_blindinglight_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/bookworm/bookworm_charge_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/bookworm/bookworm_defendandfight_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/bookworm/bookworm_dragon_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/paige.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/paige_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/bookworm.vpk` | |
| Ability model | `models/abilities/bookworm_butterfly_model.vmdl` | |
| Ability model | `models/abilities/bookworm_dragon_model.vmdl` | |
| Effects | `particles/abilities/bookworm/` | |
| Effects | `particles/weapon_fx/bookworm/` | |
| Name and tag text | `Steam_RP_hero_bookworm`, `hero_bookworm:n`, `hero_bookworm_search:n`, `Citadel_Bookworm_HeroTag_1`, `Citadel_Bookworm_HeroTag_2`, `Citadel_Bookworm_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Paige's own 172 clips on whatever model is installed, so a replacement is rigged to Paige's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `out_of_combat_idle`, `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`shoot_haspath`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Turn`, `Cast_UpperBody`, `Throw_UpperBody`, `melee_UpperBody`, `UpperBodyReplace_Slide`, `Reload_Crouch`, `Reload_Haspath`, `Reload_HasPathCrouch`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `shoot_turn`, `shoot_inair`, `shoot_crouch`, `shoot_slide`, `shoot_Idle`, `excalibur_cast`, `UpperBodyReplace_Asleep`, `Parry_UpperBody`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model bookworm assets/model.glb
dlmod icons bookworm 1=assets/icon_....png
dlmod image panorama/images/heroes/bookworm_card_psd.vtex_c assets/card.png
dlmod prop models/abilities/bookworm_butterfly_model.vmdl assets/prop_bookworm_butterfly_model.glb
dlmod paint bookworm
```
