# Silver (`werewolf`)

Weapon: Hair of the Dog. Tags: Feral, Hot Mess, Transformation.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/silver
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Slam Fire | Instantly reload your gun. You gain X and deal bonus X based on the target's health for a limited number of shots, but… | `werewolf_slamfire_psd.vtex_c` |  |
| Boot Kick |  | `werewolf_jumpback_psd.vtex_c` |  |
| Entangling Bola | Throw a bola, dealing X, applying X and preventing movement abilities or stamina. | `werewolf_bola_psd.vtex_c` |  |
| Lycan Curse | Passive: Dealing damage generates bloodlust, increased at low health. At max bloodlust, you automatically cast Lycan… | `werewolf_lycancurse_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_wip/werewolf/werewolf.vmdl` | 110 units tall |
| Animation skeleton | `models/heroes_wip/werewolf/werewolf.vnmskel` | |
| Animation clips | `models/heroes_wip/werewolf/clips/` | 223 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/werewolf_card_psd.vtex_c` | 240x320 |
| Card at low health | `panorama/images/heroes/werewolf_card_critical_psd.vtex_c` | 240x320 |
| Card after a kill | `panorama/images/heroes/werewolf_card_gloat_psd.vtex_c` | 240x320 |
| Tall portrait | `panorama/images/heroes/werewolf_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/werewolf_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/werewolf_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/werewolf_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/werewolf_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/werewolf/werewolf_bite_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/werewolf/werewolf_bola_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/werewolf/werewolf_jumpback_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/werewolf/werewolf_lycancurse_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/werewolf/werewolf_slamfire_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/werewolf/werewolf_tailwhack_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/werewolf/werewolf_wolfforward_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/werewolf.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/werewolf_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/werewolf.vpk` | |
| Effects | `particles/abilities/werewolf/` | |
| Effects | `particles/weapon_fx/werewolf/` | |
| Name and tag text | `Steam_RP_hero_werewolf`, `hero_werewolf:n`, `hero_werewolf_search:n`, `hero_werewolf_sort:n`, `Citadel_Werewolf_HeroTag_1`, `Citadel_Werewolf_HeroTag_2`, `Citadel_Werewolf_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Silver's own 223 clips on whatever model is installed, so a replacement is rigged to Silver's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `out_of_combat_idle`, `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`Cast_UpperBody`, `FullBody`, `melee_dash`, `melee_UpperBody`, `NULL`, `Parry_UpperBody`, `Reload_Crouch`, `Reload_Crouch_ALT`, `Reload_Haspath`, `Reload_HasPath_ALT`, `Reload_HasPathCrouch`, `Reload_HasPathCrouch_ALT`, `Reload_Idle`, `Reload_InAir`, `Reload_Slide`, `Reload_Zoomed`, `Throw_UpperBody`, `Turn_LowerBody`, `UpperBodyReplace_AirDash`, `UpperBodyReplace_Asleep`, `UpperBodyReplace_Asleep_Channeling`, `UpperBodyReplace_GroundDash`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Slide`, `UpperBodyReplace_Turn`, `Ability_SlamFire_UpperBody`, `Ability_EntanglingBola_UpperBody`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model werewolf assets/model.glb
dlmod select werewolf bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons werewolf 1=assets/icon_....png
dlmod image panorama/images/heroes/werewolf_card_psd.vtex_c assets/card.png
dlmod paint werewolf
```
