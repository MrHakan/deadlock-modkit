# Vyper (`viper`)

Slither in and assassinate enemies. With a powerful weapon (and deadly venom to match), Vyper can slither into the back of teams and assassinate high-priority targets in a storm of bullets.

Weapon: The Rattler. Tags: Gunner, Slippery, Rat-a-Tat.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/vyper
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Screwjab Dagger | Throw a slowing dagger | `viper_debuffdagger_psd.vtex_c` |  |
| Lethal Venom | Inject your enemies with a deadly venom | `viper_venom_psd.vtex_c` |  |
| Slither | Increase the agility of your Slide | `viper_snakedash_psd.vtex_c` |  |
| Petrifying Bola | Toss a petrifying bola | `viper_petrifybola_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_staging/viper/viper.vmdl` | 108 units tall |
| Animation skeleton | `models/heroes_staging/viper/viper.vnmskel` | |
| Animation clips | `models/heroes_staging/viper/clips/` | 222 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/kali_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/kali_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/kali_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/kali_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/kali_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/kali_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/vyper_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/viper_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/kali/kali_oscilloblade_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/kali/kali_speed_burst_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/kali/kali_whirling_blade_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/viper/viper_debuffdagger_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/viper/viper_petrifybola_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/viper/viper_snakedash_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/viper/viper_venom_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/vyper.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/vyper_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/viper.vpk` | |
| Ability model | `models/abilities/kali_boomerang.vmdl` | |
| Effects | `particles/abilities/kali/` | |
| Effects | `particles/abilities/viper/` | |
| Effects | `particles/weapon_fx/kali/` | |
| Effects | `particles/weapon_fx/viper/` | |
| Name and tag text | `Steam_RP_hero_kali`, `Steam_RP_hero_viper`, `hero_kali:n`, `hero_kali_search:n`, `hero_viper:n`, `hero_viper_search:n`, `Citadel_Viper_HeroTag_1`, `Citadel_Viper_HeroTag_2`, `Citadel_Viper_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Vyper's own 222 clips on whatever model is installed, so a replacement is rigged to Vyper's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_idle`, `out_of_combat_idle`, `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`, `reload_stand_idle`, `weapon_shoot`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `melee_UpperBody`, `Parry_UpperBody`, `Reload_Crouch`, `Reload_Haspath`, `Reload_HasPathCrouch`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `Throw_UpperBody`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_GroundDash`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Shopping`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `Ability_Dagger`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model viper assets/model.glb
dlmod select viper bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons viper 1=assets/icon_....png
dlmod image panorama/images/heroes/kali_card_psd.vtex_c assets/card.png
dlmod prop models/abilities/kali_boomerang.vmdl assets/prop_kali_boomerang.glb
dlmod paint viper
```
