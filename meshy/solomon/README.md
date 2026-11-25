# Solomon (`chessmaster`)

Weapon: Alekhine's Gun. Tags: Strategist, Focused, Checkmate.

Make a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:

```powershell
dlmod apply meshy/solomon
```

Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.

## Abilities

| Ability | What it does | Icon | Ability models |
| --- | --- | --- | --- |
| Setup Bishop | Create a bishop that remains in place indefinitely. Move (X): Moves the bishop, applying a fading X and dealing X to… | `chessmaster_bishop_psd.vtex_c` |  |
| Kingside Knight | Promote an ally to be a knight, granting bonus stamina and X for a duration. If self-cast you develop yourself as king… | `chessmaster_knight_psd.vtex_c` |  |
| Rook Battery | Release a pair of rooks that travel forward, dealing X and applying X. If the rooks hit a wall, they reflect back in… | `chessmaster_rook_psd.vtex_c` | `models/abilities/chessmaster_rook_crown_fx_model.vmdl` |
| Develop Queen |  | `chessmaster_queen_psd.vtex_c` |  |

## Game files

| What | File | Size |
| --- | --- | --- |
| Model | `models/heroes_wip/chessmaster/chessmaster.vmdl` | 107 units tall |
| Animation skeleton | `models/heroes_wip/chessmaster/chessmaster.vnmskel` | |
| Animation clips | `models/heroes_wip/chessmaster/clips/` | 305 clips |
| Hero card (hero select, HUD) | `panorama/images/heroes/chessmaster_card_psd.vtex_c` | 280x380 |
| Card at low health | `panorama/images/heroes/chessmaster_card_critical_psd.vtex_c` | 280x380 |
| Card after a kill | `panorama/images/heroes/chessmaster_card_gloat_psd.vtex_c` | 280x380 |
| Tall portrait | `panorama/images/heroes/chessmaster_vertical_psd.vtex_c` | 120x200 |
| Small portrait | `panorama/images/heroes/chessmaster_sm_psd.vtex_c` | 128x128 |
| Minimap icon | `panorama/images/heroes/chessmaster_mm_psd.vtex_c` | 50x50 |
| Background | `panorama/images/heroes/backgrounds/chessmaster_bg_psd.vtex_c` | 4096x2312 |
| Weapon picture | `panorama/images/heroes/guns/chessmaster_gun_psd.vtex_c` | 600x300 |
| Ability icon | `panorama/images/hud/abilities/chess/chessmaster_bishop_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/chess/chessmaster_fork_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/chess/chessmaster_knight_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/chess/chessmaster_moveking_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/chess/chessmaster_movepiece_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/chess/chessmaster_queen_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/chess/chessmaster_rook_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/chess/chessmaster_undobishop_psd.vtex_c` | 137x137 |
| Ability icon | `panorama/images/hud/abilities/chess/chessmaster_undoqueen_psd.vtex_c` | 137x137 |
| Name logo (SVG) | `panorama/images/heroes/hero_names/solomon.vsvg_c` | |
| Name logo (SVG) | `panorama/images/heroes/hero_names/solomon_localized.vsvg_c` | |
| Hero select 3D scene | `game/citadel/maps/ui/hero_prefabs/chessmaster.vpk` | |
| Ability model | `models/abilities/chessmaster_rook_crown_fx_model.vmdl` | |
| Ability model | `models/abilities/chessmaster_tile_model.vmdl` | |
| Effects | `particles/abilities/chessmaster/` | |
| Effects | `particles/weapon_fx/chessmaster/` | |
| Name and tag text | `Steam_RP_hero_chessmaster`, `hero_chessmaster:n`, `hero_chessmaster_search:n`, `hero_chessmaster_sort:n`, `Citadel_Chessmaster_HeroTag_1`, `Citadel_Chessmaster_HeroTag_2`, `Citadel_Chessmaster_HeroTag_3` | `citadel_heroes`, `citadel_gc_hero_names` |

## Animations

The game plays Solomon's own 305 clips on whatever model is installed, so a replacement is rigged to Solomon's bones rather than given new animations. What the game does to those clips matters:

- Clips worth checking a rig against: `weapon_run_n`, `weapon_crouch_idle`, `melee_quick_1`, `melee_hit`.
- Melee, reloads and casts are laid over the movement through bone masks (`UpperBodyReplace_Turn`, `UpperBodyReplace_Slide`, `UpperBodyReplace_InAir`, `UpperBodyReplace_Asleep`, `UpperBody_Weapon_Shoot_Run`, `shoot_turn`, `shoot_slide`, `shoot_inair`, `shoot_Idle`, `shoot_haspath`, `shoot_crouch`, `Reload_Slide`, `Reload_InAir`, `Reload_Idle`, `Reload_HasPathCrouch`, `Reload_Haspath`, `Reload_Crouch`, `melee_movementMask`, `Game_Pieces`, `Game_Pieces_Ability_Knight`, `Game_Pieces_Ability_Bishop`, `Null_Mask`, `UpperBodyReplace_Ability`, `Cast_UpperBody`, `Throw_UpperBody`, `Turn_LowerBody`): only the upper body takes the clip, so a fit to the whole clip is wrong in game.
- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.
- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.
- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.

## Install by hand

```powershell
dlmod model chessmaster assets/model.glb
dlmod select chessmaster bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide
dlmod icons chessmaster 1=assets/icon_....png
dlmod image panorama/images/heroes/chessmaster_card_psd.vtex_c assets/card.png
dlmod prop models/abilities/chessmaster_rook_crown_fx_model.vmdl assets/prop_chessmaster_rook_crown_fx_model.glb
dlmod paint chessmaster
```
