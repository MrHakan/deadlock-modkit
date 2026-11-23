# Nurse Harrow (`nurse`)

Weapon: Code Blue. Tags: Strict, Terrifying, Savior.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/nurse-harrow
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Leech Therapy | Throw a jar of leeches that attach to enemies, dealing X and eating away at the duration of enemy buffs. | `leech_psd.vtex_c` |  |
| Wretched Vapors | Crack open a vial to release an ethereal vapor onto yourself or an ally that causes a rush, then a crash. Specter Rush… | `nurse_vapors_psd.vtex_c` |  |
| Healing Needles | Load your weapon with a full clip of syringes that X allies. Dealing damage recovers charges, and each leech you have… | `needle_psd.vtex_c` |  |
| Lurking Terrors | Open your bag and reveal the horrors you carry inside, dealing X to enemies in front of you. Enemies looking at you… | `medicine_bag_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_wip/nurse/nurse.vmdl` | 134 units tall |
| Animation skeleton | `models/heroes_wip/nurse/nurse.vnmskel` | |
| Animation clips | `models/heroes_wip/nurse/clips/` | 333 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/nurse_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/nurse_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/nurse_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/nurse_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/nurse_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/nurse_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/nurse_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/nurse_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/nurse/leech_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/nurse/medicine_bag_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/nurse/needle_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/nurse/nurse_vapors_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/nurse.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/nurse_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/nurse.vpk` | |
| Ability model | `models/abilities/nurse_smelling_salts_bottle_model.vmdl` | |
| Effects | `particles/abilities/nurse/` | |
| Effects | `particles/weapon_fx/nurse/` | |
| Name and tag text | `Steam_RP_hero_nurse`, `hero_nurse:n`, `hero_nurse_search:n`, `hero_nurse_sort:n`, `Citadel_Nurse_HeroTag_1`, `Citadel_Nurse_HeroTag_2`, `Citadel_Nurse_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Nurse Harrow's own 333 clips on whatever model is installed, so a replacement is rigged to Nurse Harrow's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `FullBody`, `melee_dash`, `melee_UpperBody`, `NULL`, `Parry_UpperBody`, `Reload_Crouch`, `Reload_Haspath`, `Reload_HasPathCrouch`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `Reload_Zoomed`, `Throw_UpperBody`, `Turn_LowerBody`, `UpperBodyReplace_AirDash`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_Asleep_Channeling`, `UpperBodyReplace_GroundDash`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `Ability_SlamFire_UpperBody`, `Ability_EntanglingBola_UpperBody`, `weapon_plunger`, `cape`, `Ability_healing_needles`, `ability_vapors_running`, `ability_leeches_running`, `ability_vapors_hand`, `ability_vapors_idle`, `ability_leeches_crouch`, `Reload_Idle_ALT`, `Reload_Crouch_ALT`, `Reload_HasPath_ALT`, `Reload_HasPathCrouch_ALT`, `eyeball`, `eyeball_leeches`, `Reload_Slide_ALT`, `ability_leeches_slide`, `ability_leeches_running_throw`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model nurse assets/model.glb
dlmod select nurse bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons nurse 1=assets/icon_....png
dlmod image panorama/images/heroes/nurse_card_psd.vtex_c assets/card.png
dlmod prop models/abilities/nurse_smelling_salts_bottle_model.vmdl assets/prop_nurse_smelling_salts_bottle_model.glb
dlmod paint nurse
```
