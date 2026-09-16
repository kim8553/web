# 구음진경 source-base 전환 인계 — Stage36 (2026-09-12)

## 0. 이 문서의 목적
새 채팅에서 현재 작업을 처음부터 반복하지 않고, `9yin-go-server1` 전체 Go 소스를 구현 몸통으로 사용하면서 current `fee2df` EXE와 최신 Snail 클라이언트 근거를 계속 합치는 지점에서 바로 재개하기 위한 인계 문서다.

## 1. 권위 기준
### 최신 Snail 클라이언트 — 프로토콜/행동 의미의 최우선 권위
- fxgame.exe SHA256: `c7f0366026452d525dc0c74b8499aac30fdb69189591420978e12c7094b5fde3`
- FxGameLogic.dll SHA256: `3bfa3832c04291d9d7a44f6bc13cceb609aad55ba90e2ad937cc77221089ef96`
- fxnet2.dll SHA256: `0443d9f401fdcc6a92a391869c898780ffac93399913dc9a82c7cac8cf6c3fde`
- fxcore.dll SHA256: `ce0da0f52a80db5033e177a59be868c44b0baeca4a71226b0ccb2aca76cae724`
- 동일 current 클라이언트 Lua/resource만 동일 레벨의 근거로 사용.

### current 서버 구현 비교 기준
- `9yin-game-native-menu.exe` SHA256: `fee2df844b00e7df07508b9e67f64f0da9df20f00ace09befed790ef28b0d71c`
- current EXE `.text` SHA256: `f8c1d46bccd6629e0a974604df6e937cc24917c4cf4bbfa2a4df25420627cc93`
- 이 EXE는 current 서버 소스 차이 복원/DWARF/기계어 비교 기준이다.
- 이 EXE의 옛 gameplay semantics가 최신 Snail 클라이언트 의미보다 우선하지 않는다.

### editable source base
- 원본: `9yin-go-server1(3).rar`
- SHA256: `ddc2f6bc078660a40eedf077d43befd6402ed59478ccde987d44171c1d7aefa5`
- 전체 Go 소스, go.mod/go.sum, cmd/internal/tests 포함.
- 동일 프로젝트 계보의 이전 소스다.

## 2. 절대 작업 원칙
1. 추측 수정 금지.
2. current Snail 프로토콜/행동 의미는 current client PE/RTTI/vtable/기계어/CALL/xref/Lua/resource 근거만 사용.
3. current 서버 소스 차이는 exact `fee2df` EXE DWARF/기계어를 근거로 복원.
4. 옛 V37/V46/JYZJ/V2/과거 Go 동작을 최신 의미의 권위로 섞지 않는다.
5. 불확실하면 `확실하지 않음`, 모르면 `알 수 없습니다`.
6. STATIC PASS와 Windows/live/E2E PASS를 구분한다.
7. `.recovered` 파일은 완전한 원본 파일이라는 보장이 없다. **통째 덮어쓰기 금지. 함수 단위 병합만 허용.**
8. 현재 단계에서는 근거 없는 gameplay behavior patch를 만들지 않는다.
9. Age of Wushu 전투에는 별도 독립 MMORPG식 평타를 가정하지 않는다. 공격은 스킬/액션 기반으로 분석한다.

## 3. Git 상태
- baseline commit: `b393ce0` — user-supplied 9yin-go-server1 source baseline
- Stage27 observer/reconstruction commit: `b59abc8`
- Stage29~31 recovery commit: `bdbb8e3`
- branch: `source-observer-stage27`
- 이 인계 패키지 생성 시 Stage32~36 결과를 추가 checkpoint commit으로 고정한다. 최종 commit SHA는 `HANDOFF_MANIFEST.txt`에 기록된다.

## 4. Stage27~36 핵심 완료 상태
### Source observer
- Go source 기반 observe-only logging 추가.
- 211 V0~V13, V9 string, V10 target identity, 212 observe-only 테스트 PASS.
- 212 semantic handler는 추가하지 않음.

### current EXE source-path reconstruction
current `fee2df` DWARF가 가리키는 87개 프로젝트 소스 경로를 기준으로:
- 기존 source + recovered overlay를 합쳐 **87/87 경로 확보**.
- 이 수치는 `전체 파일 내용 current 동일`을 뜻하지 않는다.

최근 복원 완료:
- drop_table.go
- item_catalog.go
- equip_catalog.go
- equip_grant.go
- fwz_card.go
- map_path.go
- npc_path.go
- npc_transport.go
- internal/auth/field_cipher.go

### 최근 정적 검증
- `npc_path.go`: synthetic parser tests + 보유 `.path` 리소스 전수 적합성 검사 수행.
- `npc_transport.go`: synthetic + 실제 resource 기반 독립 테스트 PASS.
- `field_cipher.go`: current EXE에서 fcCookey/fcDeskey/fcDesCore/fcTransform128/fcFieldSubkeys/encryptField/EncryptFields 복원. Go `crypto/des` 기반 독립 구현과 DES core 및 최종 필드 암호화 교차검증 PASS.
- 위 결과는 STATIC/독립 테스트이며 Windows login E2E 성공을 의미하지 않는다.

## 5. Stage36 함수 단위 병합 감사 — 현재 가장 중요한 상태
파일 단위 덮어쓰기가 위험하다는 것을 확인해서 함수 단위 병합표로 전환했다.

`evidence/stage36/function_merge_inventory.txt` 집계:
- current named source functions: **739**
- baseline 또는 recovered에서 이름 대응됨: **674**
- 추가 확인 필요: **65**

중요: `65 = 모두 새로 역복원해야 함`이 아니다. 이름/형태 차이, 기존 소스에 다른 구조로 존재, 인라인/분리 변화 등을 current EXE와 확인해야 한다.

남은 65개의 우선 구역:
- `cmd/protocol-probe/equip_wear.go`: 20/23 missing — 최우선
- `cmd/protocol-probe/scene_lifecycle.go`: 13/50 missing
- `cmd/protocol-probe/shortcut_records.go`: 8/15 missing
- `gm_panel.go`: 3
- `main.go`: 3
- `messages.go`: 3
- `npc_catalog.go`: 2
- `internal/role/json_m2_repository.go`: 2
- `internal/role/mysql_repository.go`: 2
- 그 외 각 1개 수준

정확한 목록은 `evidence/stage36/function_merge_inventory.csv` / `.txt`를 사용한다.

## 6. 다음 작업 — 새 채팅에서 바로 시작할 것
### FIRST TASK: `equip_wear.go`의 missing 20개 함수 함수단위 병합 감사
현재 missing 목록:
- `(*playerActor).addEquipItem`
- `(*playerActor).hasEquipItem`
- `(*playerActor).peekBagItem`
- `(*playerActor).restoreEquip`
- `(*playerActor).takeBagItem`
- `(*playerActor).takeEquipItem`
- `applyBagMove`
- `applyEquip`
- `applyUnequip`
- `applyUseBuffItem`
- `applyUseConsumable`
- `bagItemProps`
- `bagSlotFor`
- `handleArrangeItemCustom`
- `handleDeleteItemCustom`
- `handleMoveItemCustom`
- `handleUseItemCustom`
- `openGiftBox`
- `persistBagEquip`
- `writeFrames`

진행 방법:
1. baseline `equip_wear.go`의 기존 함수/호출 구조를 inventory 한다.
2. current `fee2df` DWARF 함수 signature/line map/VA를 확인한다.
3. recovered evidence와 machine code를 대조한다.
4. 이름만 다른 동일 함수인지 / current 추가 함수인지 / semantic change인지 분류한다.
5. 함수별 `EXACT_MATCH / EQUIVALENT_RENAME_OR_INLINE / NEEDS_RECONSTRUCTION / UNCERTAIN` 상태를 남긴다.
6. 확정된 current 함수만 함수 단위로 integration candidate에 병합한다.
7. 전체 파일 덮어쓰기 금지.
8. 끝나면 739 기준 coverage를 다시 계산한다.

그 다음 순서:
- `scene_lifecycle.go` 13개
- `shortcut_records.go` 8개
- 작은 잔여 파일들
- 함수 병합 audit이 충분히 닫힌 뒤 integration branch 생성
- go test / Windows amd64 build
- source observer로 최신 client live capture
- 그 후 evidence-backed gameplay changes

## 7. 이미 끝난 분석을 반복하지 말 것
- current client 211 exact V0~V13 contract
- 212 payload layout: visual X/Y/Z + visual_rotation[1] + int32 flag; flag 공식 의미명은 아직 미확정
- 211 target identity V10, target positions V6~8/V11~13
- V9 condition-dependent string
- combat native SkillEffect→trigger_hit→hit-reaction path
- Stage24 exact-source build gate
- Stage25/26 fee2df observer 설계
- source-base 전환 판단
- 87개 source-path 확보
- Stage36 function-level merge inventory 생성

## 8. live/E2E 경계
- latest-source-base Windows executable build: 아직 최종 PASS 아님.
- latest Snail client ↔ 새 source-base live login/combat/item E2E: 아직 미검증.
- 따라서 `고쳐졌다/완성됐다` 표현 금지.

## 9. 핵심 파일
- `evidence/stage36/function_merge_inventory.csv`
- `evidence/stage36/function_merge_inventory.txt`
- `evidence/stage36/project_functions_from_exe.csv`
- `reconstruction/fee2df_overlay/`
- `docs/HANDOFF_STAGE36_20260912_KO.md`
- `docs/NEXT_CHAT_START_STAGE36_20260912.txt`
