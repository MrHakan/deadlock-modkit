# Sinclair (`magician`)

Misdirect and hex enemies. With the help of their unwilling assistant, The Magnificent Sinclair can confuse enemies with curving hexes and rapid bolts.

Weapon: The Prestige. Tags: Trickster, Copycat, Versatile.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/sinclair
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Vexing Bolt | Launch a guided bolt of magic | `magician_magicbolt_psd.vtex_c` |  |
| Spectral Assistant | Summon your faithful assistant | `magician_cloneturret_psd.vtex_c` | `models/abilities/magician_cloneturret_model.vmdl` |
| Rabbit Hex | Hex a target area and transform all enemies into a Rabbit for a limited duration. Rabbits are small and move faster… | `magician_animalcurse_psd.vtex_c` | `models/abilities/magician_animalcurse.vmdl`, `models/abilities/magician_rabbit_model.vmdl` |
| Audience Participation | Copy an enemy's Ultimate for a limited time | `magician_copyult_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_staging/magician_v2/magician.vmdl` | 105 units tall |
| Animation skeleton | `models/heroes_staging/magician_v2/magician.vnmskel` | |
| Animation clips | `models/heroes_staging/magician_v2/dmx/animation/ag2/` | 82 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/magician_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/magician_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/magician_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/magician_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/magician_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/magician_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/magician_bg_psd.vtex_c` | 4096x2312 |
| Ability icon | `panorama/images/hud/abilities/magician/magician_animalcurse_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/magician/magician_cloneturret_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/magician/magician_copyult_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/magician/magician_magicbolt_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/sinclair.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/sinclair_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/magician.vpk` | |
| Ability model | `models/abilities/magician_animalcurse.vmdl` | |
| Ability model | `models/abilities/magician_cloneturret_model.vmdl` | |
| Ability model | `models/abilities/magician_rabbit_model.vmdl` | |
| Effects | `particles/abilities/magician/` | |
| Effects | `particles/weapon_fx/magician/` | |
| Name and tag text | `Steam_RP_hero_magician`, `hero_magician:n`, `hero_magician_search:n`, `hero_magician_sort:n`, `Citadel_Magician_HeroTag_1`, `Citadel_Magician_HeroTag_2`, `Citadel_Magician_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Sinclair's own 82 clips on whatever model is installed, so a replacement is rigged to Sinclair's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_run_n`, `weapon_crouch_idle`, `reload_stand_idle`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `Throw_UpperBody`, `Reload_Idle`, `Reload_Haspath`, `Reload_InAir`, `melee_UpperBody`, `UpperBodyReplace_Turn`, `Reload_Crouch`, `Reload_HasPathCrouch`, `Parry_UpperBody`, `shoot`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model magician assets/model.glb
dlmod select magician bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons magician 1=assets/icon_....png
dlmod image panorama/images/heroes/magician_card_psd.vtex_c assets/card.png
dlmod prop models/abilities/magician_animalcurse.vmdl assets/prop_magician_animalcurse.glb
dlmod paint magician
```
