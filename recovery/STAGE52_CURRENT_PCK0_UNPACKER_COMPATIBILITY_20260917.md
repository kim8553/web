# Stage52 — 최신 Snail PCK0 언팩 도구 실제 호환성 검증

- 기준: `kim8553/web` / `stage37-current-recovery-20260916` 시작 HEAD `2ef78dec056687cd7311c8f6ca19a61db47a9a0b`.
- 원본 근거: 기존 Drive 원본 `ini.package`, `lua.package`, `share.package`를 작업 공간에서 읽기 전용 사용. 각각 SHA-256 및 실제 바이트 수와 헤더 항목 수·인덱스 경계를 고정값과 대조. 개인 게임 파일·복호화 키·출력 리소스는 공개 GitHub에 올리지 않는다.
- 소스 대조: [Wushu Utils Unpacker.cs](https://github.com/Ersanio/wushu-utils/blob/main/Wushu.Utils.Package/Unpacker.cs) blob `dc2e717ae43cdaeff3e87b5053b4d1e40b2bcc73`는 10바이트 고정 헤더만 허용. [JiuYinUnpackTool pck.rs](https://github.com/russell662/JiuYinUnpackTool/blob/main/src/pck.rs) blob `ee5722b5ec139d5254f9210791361ec7193c0af8`는 flag != 0 거부. [AOW Package Extractor Program.cs](https://github.com/ramazanaktolu/AOWPackageExtractor/blob/master/Program.cs) blob `6a402b0c4b9d734b16c557402a921b4e54ba7988`는 원시 인덱스의 첫 32비트 offset을 곧바로 파일 위치로 읽는다. [AOWPackageExtractorCpp](https://github.com/ramazanaktolu/AOWPackageExtractorCpp/blob/master/AOWPackageExtractorCpp.cpp) blob `f51d49ef5f135a8e8395e649002bc653438a4caf` 역시 인덱스 변환을 확인할 수 없는 원시 TOC 파서다. **실행 파일이 아닌 공개 소스 규칙 대조 + 로컬 원본 바이트 범위 검증**이며 이 도구들의 바이너리를 실행했다고 주장하지 않는다.

| 원본 | 선언 엔트리 수(해독 전) | 인덱스 끝 | 인덱스 헤더 6–7(u16) | 원시 첫 offset(u32 LE) | 실제 파일 크기 |
|---|---:|---:|---:|---:|---:|
| ini.package | 18,543 | 1,219,224 | 4 | 976,513,372 | 20,578,406 |
| lua.package | 2,286 | 187,177 | 4 | 699,817,639 | 21,840,086 |
| share.package | 9,726 | 790,911 | 4 | 895,702,521 | 40,680,972 |

- **세 원본 모두** Wushu Utils 구형 10바이트 헤더 검사 불일치, JiuYinUnpackTool flag==0 요건 불충족, AOW의 원시 오프셋 해석 시 첫 데이터 영역이 패키지 범위 밖. 두 인덱스 파서에서 보는 실제 원본 데이터의 첫 u64 offset도 파일 크기를 초과. 따라서 **헤더의 4만 0으로 패치하는 것은 해결책이 아님**: 원시 인덱스 바이트가 정상 파일 위치를 제공하지 않는다.
- 표식 4의 암호화 종류/키/복호화 알고리즘 및 헤더 8–9의 의미 **알 수 없음**. 반복적으로 관찰된 보호 섹션 호출은 앞선 Stage47처럼 역할이 미확정이므로 임의 키·XOR·AES를 발명하지 않는다. 원본 패키지 파일을 변경하거나 제3자 도구가 성공했다고 주장하지 않는다.
- 새 읽기 전용 도구 [`tools/stage52_pck0_tool_compatibility.py`](../tools/stage52_pck0_tool_compatibility.py): 원본 SHA·헤더 고정 검증 후 메타데이터만 출력, 불일치 시 거부. 합성 구형 인덱스 긍정 테스트와 표식 4의 **표식만으로 실제 인덱스 암호화를 증명할 수 없음** 테스트, 변경된 바이트 2건 거부 테스트 PASS. 실물 세 패키지 검증 PASS, **원래 이름으로 추출된 엔트리는 0건**.
- 복원·서버 경계: 기존 Stage48–51 익명 zlib·Lua 복원 성과는 유지한다. 기존 `9yin-go-server1.rar` 기반 `server/` 및 NPC 구매 차단은 수정하지 않는다. Go build/LIVE/E2E 미실행.

다음 증거는 최신 동판본 클라이언트의 **검증된 인덱스 변환 함수 또는 같은 원본을 읽는 도구의 구현/성공 로그**다. 그 전에는 완전 언팩 가능 또는 구매 기능 복구를 보장할 수 없다.
