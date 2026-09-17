# Current official life-profession exchange binding scope — 2026-09-17

This checkpoint freezes the subset for which current resource topology and an official Snail rule agree. It does not guess the remaining wire integers.

## Official rule

- Snail notice (2021-06-09): `https://9yin.woniu.com/news/sysnotice/2021/0609/50525.html`
- Maintenance notice: `https://9yin.woniu.com/news/sysnotice/2021/0609/50560.html`
- For the listed old life-profession items: unbound old item -> unbound material package or new item; bound old item -> bound material package or new item.
- Material-package contents become bound when the package is opened, regardless of whether the package itself was bound.

## Exact current-resource join

- `Shop_dy_20206`: 39 mode-3 rows -> new items
- `Shop_dy_20216`: 39 mode-3 rows -> material packages
- exact shared coordinate set: 39/39
- all 78 referenced `ExchangeItem.ini` sections author `BindStatus=1`
- `Condition`, `Condition2`, `Filters`, `Prop`: empty across these 78 definitions
- official item population: 毒师 8 + 药师 2 + 厨师 29 = 39
- material-package requirement distribution: qty 1 = 4, qty 2 = 8, qty 5 = 27; exactly matches the official table
- current `DropBind=1` result set: 39/39 exactly equals the `Shop_dy_20216` package result set; no extras, no misses
- display-name cross-check: 35/39 current input labels equal the official notice label exactly. Four differ (`duyao90019`, `duyao91003`, `cs_shejian_1001`, `cs_shejian_1002`), so those four are linked by current shop/config topology and paired result IDs, not name identity alone.

## What this proves / does not prove

Proven for these 78 rows: final *semantic* binding follows the consumed old item binding, and the package-output variant has a distinct post-open binding rule for its contents.

Still not proven: the exact numeric S2C 557 values for `ShowBind` and `ExchangeBind`. Official documentation proves that the red binding notice is shown, but it does not expose the literal wire integers. Therefore production stays fail-closed until a runtime 557 capture supplies those fields.

## 39-item mapping

| Profession | Official notice | Current input label | old ConfigID | New-item ExchangeData | Package ExchangeData | Package qty | New result | Package result |
|---|---|---|---|---:|---:|---:|---|---|
| 毒师 | 赤炼秘药 | 赤炼秘药 | `duyao90004` | 12593 | 12556 | 2 | `newduyao90004` | `box_newduyao90004` |
| 毒师 | 肠青秘药 | 肠青秘药 | `duyao10283` | 12594 | 12557 | 2 | `newduyao10283` | `box_newduyao10283` |
| 毒师 | 金蚕秘药 | 金蚕秘药 | `duyao90009` | 12595 | 12558 | 2 | `newduyao90009` | `box_newduyao90009` |
| 毒师 | 天虫秘药 | 天虫秘药 | `duyao10033` | 12596 | 12559 | 2 | `newduyao10033` | `box_newduyao10033` |
| 毒师 | 毒功奇应秘药 | 毒功奇应秘药 | `duyao10298` | 12597 | 12560 | 2 | `newduyao10298` | `box_newduyao10298` |
| 毒师 | 龙飞凤舞秘药 | 金蛇狂舞秘药 | `duyao90019` | 12598 | 12561 | 1 | `newduyao90019` | `box_newduyao90019` |
| 毒师 | 精炼龙飞凤舞秘药 | 精炼金蛇狂舞秘药 | `duyao91003` | 12599 | 12562 | 1 | `newduyao91003` | `box_newduyao91003` |
| 毒师 | 断筋腐骨秘药 | 断筋腐骨秘药 | `duyao10285` | 12600 | 12563 | 2 | `newduyao10285` | `box_newduyao10285` |
| 药师 | 定神护心丹 | 定神护心丹 | `item_wgm_huxindan` | 12632 | 12630 | 2 | `newitem_wgm_huxindan` | `box_newitem_wgm_huxindan` |
| 药师 | 十香返生丸 | 十香返生丸 | `item_wgm_fanshenwan` | 12633 | 12631 | 2 | `newitem_wgm_fanshenwan` | `box_newitem_wgm_fanshenwan` |
| 厨师 | 佳肴·煎饼卷大葱 | 舌尖·煎饼卷大葱 | `cs_shejian_1001` | 12601 | 12564 | 1 | `newcs_shejian_1001` | `box_newcs_shejian_1001` |
| 厨师 | 佳肴·乐山嫩豆花 | 舌尖·乐山嫩豆花 | `cs_shejian_1002` | 12602 | 12565 | 1 | `newcs_shejian_1002` | `box_newcs_shejian_1002` |
| 厨师 | 板栗烧鸡 | 板栗烧鸡 | `caiyao10169` | 12603 | 12566 | 5 | `newcaiyao10169` | `box_newcaiyao10169` |
| 厨师 | 酸辣鱼 | 酸辣鱼 | `caiyao10120` | 12604 | 12567 | 5 | `newcaiyao10120` | `box_newcaiyao10120` |
| 厨师 | 川味棒棒鸡 | 川味棒棒鸡 | `caiyao10130` | 12605 | 12568 | 5 | `newcaiyao10130` | `box_newcaiyao10130` |
| 厨师 | 过桥米线 | 过桥米线 | `caiyao10134` | 12606 | 12569 | 5 | `newcaiyao10134` | `box_newcaiyao10134` |
| 厨师 | 麻婆豆腐 | 麻婆豆腐 | `caiyao10131` | 12607 | 12570 | 5 | `newcaiyao10131` | `box_newcaiyao10131` |
| 厨师 | 锅烧羊肉 | 锅烧羊肉 | `caiyao10166` | 12608 | 12571 | 5 | `newcaiyao10166` | `box_newcaiyao10166` |
| 厨师 | 酥锅鱼冻 | 酥锅鱼冻 | `caiyao10171` | 12609 | 12572 | 5 | `newcaiyao10171` | `box_newcaiyao10171` |
| 厨师 | 黑椒银鱼 | 黑椒银鱼 | `caiyao10172` | 12610 | 12573 | 5 | `newcaiyao10172` | `box_newcaiyao10172` |
| 厨师 | 剁椒鱼头 | 剁椒鱼头 | `caiyao10132` | 12611 | 12574 | 5 | `newcaiyao10132` | `box_newcaiyao10132` |
| 厨师 | 牛肉藏面 | 牛肉藏面 | `caiyao10138` | 12612 | 12575 | 5 | `newcaiyao10138` | `box_newcaiyao10138` |
| 厨师 | 裂腹鱼 | 裂腹鱼 | `caiyao10140` | 12613 | 12576 | 5 | `newcaiyao10140` | `box_newcaiyao10140` |
| 厨师 | 红汤爆鱼面 | 红汤爆鱼面 | `caiyao10197` | 12614 | 12577 | 5 | `newcaiyao10197` | `box_newcaiyao10197` |
| 厨师 | 醉槽乌骨鸡 | 醉槽乌骨鸡 | `caiyao10187` | 12615 | 12578 | 5 | `newcaiyao10187` | `box_newcaiyao10187` |
| 厨师 | 武当大曲鸭 | 武当大曲鸭 | `caiyao10148` | 12616 | 12579 | 5 | `newcaiyao10148` | `box_newcaiyao10148` |
| 厨师 | 杭州酱鸭 | 杭州酱鸭 | `caiyao10184` | 12617 | 12580 | 5 | `newcaiyao10184` | `box_newcaiyao10184` |
| 厨师 | 腊肉烤方 | 腊肉烤方 | `caiyao10135` | 12618 | 12581 | 5 | `newcaiyao10135` | `box_newcaiyao10135` |
| 厨师 | 冰糖八宝粥 | 冰糖八宝粥 | `caiyao10136` | 12619 | 12582 | 5 | `newcaiyao10136` | `box_newcaiyao10136` |
| 厨师 | 撒尿牛丸 | 撒尿牛丸 | `caiyao10200` | 12620 | 12583 | 5 | `newcaiyao10200` | `box_newcaiyao10200` |
| 厨师 | 朝阳羊宴 | 朝阳羊宴 | `caiyao10175` | 12621 | 12584 | 5 | `newcaiyao10175` | `box_newcaiyao10175` |
| 厨师 | 小葱拌豆腐 | 小葱拌豆腐 | `caiyao10208` | 12622 | 12585 | 5 | `newcaiyao10208` | `box_newcaiyao10208` |
| 厨师 | 桂花粥 | 桂花粥 | `caiyao10209` | 12623 | 12586 | 5 | `newcaiyao10209` | `box_newcaiyao10209` |
| 厨师 | 木耳炒山药 | 木耳炒山药 | `caiyao10210` | 12624 | 12587 | 5 | `newcaiyao10210` | `box_newcaiyao10210` |
| 厨师 | 红烧牛腩 | 红烧牛腩 | `caiyao10211` | 12625 | 12588 | 5 | `newcaiyao10211` | `box_newcaiyao10211` |
| 厨师 | 油焖茄子 | 油焖茄子 | `caiyao10212` | 12626 | 12589 | 5 | `newcaiyao10212` | `box_newcaiyao10212` |
| 厨师 | 琅琊酥糖 | 琅琊酥糖 | `caiyao10213` | 12627 | 12590 | 5 | `newcaiyao10213` | `box_newcaiyao10213` |
| 厨师 | 葱油莴笋丝 | 葱油莴笋丝 | `caiyao10214` | 12628 | 12591 | 5 | `newcaiyao10214` | `box_newcaiyao10214` |
| 厨师 | 酸辣白菜 | 酸辣白菜 | `caiyao10215` | 12629 | 12592 | 5 | `newcaiyao10215` | `box_newcaiyao10215` |
